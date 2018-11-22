package consume

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
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
func Run(qcfg queue.Config, tempDir string) error {
	// Create temporary dir
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return errors.Wrap(err, "could not create temp dir")
	}
	defer os.RemoveAll(tempDir)

	conn, err := queue.NewConnection(qcfg)
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
	}()

	log.Info.Println("Waiting for jobs")

	var desc job.Unprocessed
	for j := range jobs {
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

		if err := RunTransaction(desc, task); err != nil {
			log.Error.Println(
				errors.Wrap(err, "could not run CVMFS transaction"))
			j.Nack(false, true)
			continue
		}

		j.Ack(false)
		log.Info.Println("Finished publishing job:", desc.ID.String())
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
			scriptFile, err = unpackScript(desc.Script, tempDir)
			if err != nil {
				return errors.Wrap(err, "unpacking transaction script failed")
			}
		} else {
			scriptFile = desc.Script
		}

		err := runScript(scriptFile, desc.Repository, desc.RepositoryPath, desc.ScriptArgs)
		if err != nil {
			return errors.Wrap(err, "running transaction script failed")
		}
	}

	return nil
}

func unpackScript(script string, tempDir string) (string, error) {
	buf, err := base64.StdEncoding.DecodeString(script)
	if err != nil {
		return "", errors.Wrap(err, "base64 decoding failed")
	}
	rd := bytes.NewReader(buf)
	gz, err := gzip.NewReader(rd)
	if err != nil {
		return "", errors.Wrap(err, "gzip reader construction failed")
	}
	rawbuf, err := ioutil.ReadAll(gz)
	if err != nil {
		return "", errors.Wrap(err, "decompression failed")
	}
	scriptFileName := path.Join(tempDir, "transaction.sh")
	if err := ioutil.WriteFile(scriptFileName, rawbuf, 0755); err != nil {
		return "", errors.Wrap(err, "writing to disk failed")
	}

	return scriptFileName, nil
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
