package consume

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/auth"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/jobdb"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	getter "github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

var mock bool

func init() {
	mock = false
	v := os.Getenv("CVMFS_MOCK_JOB_CONSUMER")
	if v == "true" || v == "yes" || v == "on" {
		mock = true
	}
}

// Run - runs the job consumer
func Run(qCfg *queue.Config, jCfg *jobdb.Config, tempDir string, maxJobRetries int) error {
	keys, err := auth.ReadKeys(jCfg.KeyDir)
	if err != nil {
		return errors.Wrap(err, "could not read API keys from file")
	}

	// Create temporary dir
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return errors.Wrap(err, "could not create temp dir")
	}
	defer os.RemoveAll(tempDir)

	conn, err := queue.NewConnection(qCfg)
	if err != nil {
		return errors.Wrap(err, "could not create job queue connection")
	}
	defer conn.Close()

	jobs, err := conn.Chan.Consume(
		conn.Queue.Name, queue.ConsumerName, false, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "could not start consuming jobs")
	}

	go func() {
		ch := conn.Chan.NotifyClose(make(chan *amqp.Error))
		err := <-ch
		log.Error.Println(errors.Wrap(err, "connection to job queue closed"))
		os.Exit(1)
	}()

	var prefix string
	if !strings.HasPrefix(jCfg.Host, "http://") {
		prefix = "http://"
	}
	jobPostURL := fmt.Sprintf("%s%s:%v/jobs", prefix, jCfg.Host, jCfg.Port)

	log.Info.Println("Waiting for jobs")

	for j := range jobs {
		startTime := time.Now()

		var desc job.Unprocessed
		if err := json.Unmarshal(j.Body, &desc); err != nil {
			log.Error.Println(
				errors.Wrap(err, "could not unmarshal job description"))
			j.Nack(false, false)
			continue
		}

		log.Info.Println("Start publishing job:", desc.ID.String())

		task := func() error {
			return processTransaction(&desc, tempDir)
		}

		success := false
		errMsg := ""
		retry := 0
		for retry <= maxJobRetries {
			err := RunTransaction(desc, task)
			if err != nil {
				wrappedErr := errors.Wrap(err, "could not run CVMFS transaction")
				errMsg = wrappedErr.Error()
				log.Error.Println(wrappedErr)
				retry++
				log.Info.Printf("Transaction failed.")
				if retry <= maxJobRetries {
					log.Info.Printf(" Retrying: %v/%v\n", retry, maxJobRetries)
				}
			} else {
				success = true
				break
			}
		}

		finishTime := time.Now()

		// Publish the processed job status to the Job DB
		processedJob := job.Processed{
			Unprocessed:  desc,
			StartTime:    startTime,
			FinishTime:   finishTime,
			Successful:   success,
			ErrorMessage: errMsg,
		}
		if err := postJobStatus(jobPostURL, keys, &processedJob, conn); err != nil {
			log.Error.Println(
				errors.Wrap(err, "posting job status to DB failed"))
			j.Nack(false, false)
			continue
		}

		j.Ack(false)
		result := "failed"
		if success {
			result = "success"
		}
		log.Info.Printf(
			"Finished publishing job: %v, %v\n", desc.ID.String(), result)
	}

	return nil
}

func processTransaction(desc *job.Unprocessed, tempDir string) error {
	// Create target dir if needed
	targetDir := path.Join(
		"/cvmfs", desc.Repository, desc.RepositoryPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return errors.Wrap(err, "could not create target dir")
	}

	// Download and unpack the payload, if given
	log.Info.Println("Downloading payload:", desc.Payload)
	if err := getter.Get(targetDir, desc.Payload); err != nil {
		return errors.Wrap(err, "could not download payload")
	}

	// Run the transaction script, if specified
	if desc.Script != "" {
		needsUnpacking := desc.TransferScript
		log.Info.Printf(
			"Running transaction script: %v (needs unpacking: %v)\n",
			desc.Script, needsUnpacking)

		var scriptFile string
		if needsUnpacking {
			var err error
			scriptFile = path.Join(tempDir, "transaction.sh")
			err = job.UnpackScript(desc.Script, scriptFile)
			if err != nil {
				return errors.Wrap(err, "unpacking transaction script failed")
			}
		} else {
			scriptFile = desc.Script
		}

		err := runScript(
			scriptFile, desc.Repository, desc.RepositoryPath, desc.ScriptArgs)
		if err != nil {
			return errors.Wrap(err, "running transaction script failed")
		}
	}

	return nil
}

func runScript(script string, repo string, repoPath string, args string) error {
	cmd := exec.Command(script, repo, repoPath, args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path.Join("/cvmfs", repo)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func postJobStatus(url string, keys *auth.Keys, j *job.Processed, q *queue.Connection) error {
	buf, err := json.Marshal(j)
	if err != nil {
		return errors.Wrap(err, "JSON encoding of job status failed")
	}

	// Compute message HMAC
	keyID, present := keys.RepoKeys[j.Repository]
	if !present {
		return errors.New(
			fmt.Sprintf("Key not found for repository: %v", j.Repository))
	}
	key, present := keys.Secrets[keyID]
	if !present {
		return errors.New(
			fmt.Sprintf("Secret not found for keyID: %v", keyID))
	}
	hmac, err := auth.ComputeHMAC(buf, key)
	if err != nil {
		errors.Wrap(err, "could not compute HMAC")
	}
	hmacStr := base64.StdEncoding.EncodeToString(hmac)

	rdr := bytes.NewReader(buf)

	req, err := http.NewRequest("POST", url, rdr)
	if err != nil {
		errors.Wrap(err, "could not create POST request")
	}
	req.Header["Authorization"] = []string{fmt.Sprintf("%v %v", keyID, hmacStr)}
	req.Header["Content-Type"] = []string{"application/json"}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Posting job status to DB failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Post request failed: %v", resp.Status))
	}

	buf2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Reading reply body failed")
	}

	var pubStat jobdb.PutJobReply
	if err := json.Unmarshal(buf2, &pubStat); err != nil {
		return errors.Wrap(err, "JSON decoding of reply failed")
	}

	if pubStat.Status != "ok" {
		return errors.Wrap(
			err, fmt.Sprintf("Posting job status failed: %s", pubStat.Reason))
	}

	shortStatus := jobdb.ShortJobStatus{ID: j.ID, Successful: j.Successful}
	var routingKey string
	if j.Successful {
		routingKey = queue.SuccessKey
	} else {
		routingKey = queue.FailedKey
	}
	if err := q.Publish(
		queue.CompletedJobExchange, routingKey, shortStatus); err != nil {
		return errors.Wrap(err, "Posting job status to notification exchange failed")
	}

	return nil
}
