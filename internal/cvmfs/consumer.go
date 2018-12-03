package cvmfs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

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

// Consumer - a job consumer object
type Consumer struct {
	qCons         *QueueClient
	qPub          *QueueClient
	jobDBURL      string
	keys          *Keys
	tempDir       string
	maxJobRetries int
}

// NewConsumer - creates a new job consumer object
func NewConsumer(
	qCfg *QueueConfig, keys *Keys, jobDBURL string, tempDir string,
	maxJobRetries int) (*Consumer, error) {

	cleanUp := false

	// Create clients for the queue system
	qCons, err := NewQueueClient(qCfg, ConsumerConnection)
	if err != nil {
		return nil, errors.Wrap(
			err, "could not create job queue connection (consumer)")
	}
	defer func() {
		if cleanUp {
			qCons.Close()
		}
	}()
	qPub, err := NewQueueClient(qCfg, PublisherConnection)
	if err != nil {
		cleanUp = true
		return nil, errors.Wrap(
			err, "could not create job queue connection (publisher)")
	}

	return &Consumer{
		qCons, qPub, jobDBURL, keys, tempDir, maxJobRetries}, nil
}

// Close all the internal connections of the consumer
func (c *Consumer) Close() {
	c.qCons.Close()
	c.qPub.Close()
}

// Loop start the event loop for consuming job messages
func (c *Consumer) Loop() error {
	jobs, err := c.qCons.Chan.Consume(
		c.qCons.NewJobQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "could not start consuming jobs")
	}

	go func() {
		ch := c.qCons.Chan.NotifyClose(make(chan *amqp.Error))
		err := <-ch
		LogError.Println(
			errors.Wrap(err, "connection to job queue closed"))
		os.Exit(1)
	}()

	for j := range jobs {
		c.handleMessage(&j)
	}

	return nil
}

func (c *Consumer) handleMessage(msg *amqp.Delivery) {
	startTime := time.Now()

	var desc UnprocessedJob
	if err := json.Unmarshal(msg.Body, &desc); err != nil {
		LogError.Println(
			errors.Wrap(err, "could not unmarshal job description"))
		msg.Nack(false, false)
		return
	}

	if len(desc.Dependencies) > 0 {
		// Wait for job dependencies to finish
		depStatus, err := WaitForJobs(
			desc.Dependencies, c.qCons, c.jobDBURL)
		if err != nil {
			err := errors.Wrap(err, "waiting for job dependencies failed")
			LogError.Println(err)
			if err := postJobStatus(
				&desc, startTime, time.Now(), false, err.Error(),
				c.jobDBURL, c.keys, c.qCons); err != nil {
				LogError.Println(
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
			LogError.Println(err)
			if err := postJobStatus(
				&desc, startTime, time.Now(), false, err.Error(),
				c.jobDBURL, c.keys, c.qCons); err != nil {
				LogError.Println(
					errors.Wrap(err, "posting job status to DB failed"))
				msg.Nack(false, true)
				return
			}
		}
	}

	LogInfo.Println("Start publishing job:", desc.ID.String())

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
			LogError.Println(wrappedErr)
			retry++
			LogInfo.Printf("Transaction failed.")
			if retry <= c.maxJobRetries {
				LogInfo.Printf(" Retrying: %v/%v\n", retry, c.maxJobRetries)
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
		LogError.Println(
			errors.Wrap(err, "posting job status to DB failed"))
		msg.Nack(false, true)
		return
	}

	msg.Ack(false)
	result := "failed"
	if success {
		result = "success"
	}
	LogInfo.Printf(
		"Finished publishing job: %v, %v\n", desc.ID.String(), result)
}

func postJobStatus(
	j *UnprocessedJob, t0 time.Time, t1 time.Time, success bool, errMsg string,
	url string, keys *Keys, q *QueueClient) error {

	processed := ProcessedJob{
		UnprocessedJob: *j,
		StartTime:      t0,
		FinishTime:     t1,
		Successful:     success,
		ErrorMessage:   errMsg,
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
	hmac := base64.StdEncoding.EncodeToString(ComputeHMAC(buf, key))

	rdr := bytes.NewReader(buf)

	// Post processed job status to the job DB
	req, err := http.NewRequest("POST", url, rdr)
	if err != nil {
		errors.Wrap(err, "could not create POST request")
	}
	req.Header.Add("Authorization", fmt.Sprintf("%v %v", keyID, hmac))
	req.Header.Add("Content-Type", "application/json")

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

	var pubStat PutJobReply
	if err := json.Unmarshal(buf2, &pubStat); err != nil {
		return errors.Wrap(err, "JSON decoding of reply failed")
	}

	if pubStat.Status != "ok" {
		return errors.Wrap(
			err, fmt.Sprintf("Posting job status failed: %s", pubStat.Reason))
	}

	// Publish a notification to the completed job exchange
	status := JobStatus{ID: processed.ID, Successful: processed.Successful}
	var routingKey string
	if processed.Successful {
		routingKey = SuccessKey
	} else {
		routingKey = FailedKey
	}
	if err := q.Publish(
		CompletedJobExchange, routingKey, status); err != nil {
		return errors.Wrap(err, "Posting job status to notification exchange failed")
	}

	return nil
}
