package consume

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/auth"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/jobdb"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type consumer struct {
	qCons         *queue.Client
	qPub          *queue.Client
	jobDBURL      string
	keys          *auth.Keys
	tempDir       string
	maxJobRetries int
}

func createConsumer(
	qCfg *queue.Config, keys *auth.Keys, jobDBURL string, tempDir string,
	maxJobRetries int) (*consumer, error) {

	cleanUp := false

	// Create clients for the queue system
	qCons, err := queue.NewClient(qCfg, queue.ConsumerConnection)
	if err != nil {
		return nil, errors.Wrap(
			err, "could not create job queue connection (consumer)")
	}
	defer func() {
		if cleanUp {
			qCons.Close()
		}
	}()
	qPub, err := queue.NewClient(qCfg, queue.PublisherConnection)
	if err != nil {
		cleanUp = true
		return nil, errors.Wrap(
			err, "could not create job queue connection (publisher)")
	}

	return &consumer{
		qCons, qPub, jobDBURL, keys, tempDir, maxJobRetries}, nil
}

func (c *consumer) close() {
	c.qCons.Close()
	c.qPub.Close()
}

func (c *consumer) loop() error {
	jobs, err := c.qCons.Chan.Consume(
		c.qCons.NewJobQueue.Name, queue.ClientName,
		false, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "could not start consuming jobs")
	}

	go func() {
		ch := c.qCons.Chan.NotifyClose(make(chan *amqp.Error))
		err := <-ch
		log.Error.Println(
			errors.Wrap(err, "connection to job queue closed"))
		os.Exit(1)
	}()

	for j := range jobs {
		c.handleMessage(&j)
	}

	return nil
}

func (c *consumer) handleMessage(msg *amqp.Delivery) {
	startTime := time.Now()

	var desc job.Unprocessed
	if err := json.Unmarshal(msg.Body, &desc); err != nil {
		log.Error.Println(
			errors.Wrap(err, "could not unmarshal job description"))
		msg.Nack(false, false)
		return
	}

	// Wait for job dependencies to finish
	depStatus, err := job.WaitForJobs(
		desc.Dependencies, c.qCons, c.jobDBURL)
	if err != nil {
		err := errors.Wrap(err, "waiting for job dependencies failed")
		log.Error.Println(err)
		if err := postJobStatus(
			&desc, startTime, time.Now(), false, err.Error(),
			c.jobDBURL, c.keys, c.qCons); err != nil {
			log.Error.Println(
				errors.Wrap(err, "posting job status to DB failed"))
			msg.Nack(false, true)
			return
		}
	}

	// In any of the dependencies failed, the current job should also
	// be listed as failed
	failed := []string{}
	for _, st := range depStatus {
		if st.Successful == false {
			failed = append(failed, st.ID.String())
		}
	}
	if len(failed) > 0 {
		err := errors.New(
			fmt.Sprintf("failed job dependencies: %v", failed))
		log.Error.Println(err)
		if err := postJobStatus(
			&desc, startTime, time.Now(), false, err.Error(),
			c.jobDBURL, c.keys, c.qCons); err != nil {
			log.Error.Println(
				errors.Wrap(err, "posting job status to DB failed"))
			msg.Nack(false, true)
			return
		}
	}

	log.Info.Println("Start publishing job:", desc.ID.String())

	task := func() error {
		return desc.Process(c.tempDir)
	}

	success := false
	errMsg := ""
	retry := 0
	for retry <= c.maxJobRetries {
		err := RunTransaction(desc, task)
		if err != nil {
			wrappedErr := errors.Wrap(err, "could not run CVMFS transaction")
			errMsg = wrappedErr.Error()
			log.Error.Println(wrappedErr)
			retry++
			log.Info.Printf("Transaction failed.")
			if retry <= c.maxJobRetries {
				log.Info.Printf(" Retrying: %v/%v\n", retry, c.maxJobRetries)
			}
		} else {
			success = true
			break
		}
	}

	finishTime := time.Now()

	// Publish the processed job status to the Job DB
	if err := postJobStatus(
		&desc, startTime, finishTime, success, errMsg,
		c.jobDBURL, c.keys, c.qPub); err != nil {
		log.Error.Println(
			errors.Wrap(err, "posting job status to DB failed"))
		msg.Nack(false, true)
		return
	}

	msg.Ack(false)
	result := "failed"
	if success {
		result = "success"
	}
	log.Info.Printf(
		"Finished publishing job: %v, %v\n", desc.ID.String(), result)
}

func postJobStatus(
	j *job.Unprocessed, t0 time.Time, t1 time.Time, success bool, errMsg string,
	url string, keys *auth.Keys, q *queue.Client) error {

	processed := job.Processed{
		Unprocessed:  *j,
		StartTime:    t0,
		FinishTime:   t1,
		Successful:   success,
		ErrorMessage: errMsg,
	}
	buf, err := json.Marshal(processed)
	if err != nil {
		return errors.Wrap(err, "JSON encoding of job status failed")
	}

	// Compute message HMAC
	keyID, present := keys.RepoKeys[processed.Repository]
	if !present {
		return errors.New(
			fmt.Sprintf("Key not found for repository: %v", processed.Repository))
	}
	key, present := keys.Secrets[keyID]
	if !present {
		return errors.New(
			fmt.Sprintf("Secret not found for keyID: %v", keyID))
	}
	hmac := base64.StdEncoding.EncodeToString(auth.ComputeHMAC(buf, key))

	rdr := bytes.NewReader(buf)

	// Post processed job status to the job DB
	req, err := http.NewRequest("POST", url, rdr)
	if err != nil {
		errors.Wrap(err, "could not create POST request")
	}
	req.Header["Authorization"] = []string{fmt.Sprintf("%v %v", keyID, hmac)}
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

	// Publish a notification to the completed job exchange
	status := job.Status{ID: processed.ID, Successful: processed.Successful}
	var routingKey string
	if processed.Successful {
		routingKey = queue.SuccessKey
	} else {
		routingKey = queue.FailedKey
	}
	if err := q.Publish(
		queue.CompletedJobExchange, routingKey, status); err != nil {
		return errors.Wrap(err, "Posting job status to notification exchange failed")
	}

	return nil
}
