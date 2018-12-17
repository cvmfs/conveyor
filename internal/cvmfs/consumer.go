package cvmfs

import (
	"encoding/json"
	"fmt"
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
	client        *JobClient
	keys          *Keys
	endpoints     HTTPEndpoints
	tempDir       string
	maxJobRetries int
}

// NewConsumer - creates a new job consumer object
func NewConsumer(
	keys *Keys, cfg *Config, tempDir string,
	maxJobRetries int) (*Consumer, error) {

	client, err := NewJobClient(keys, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not create a queue client")
	}

	return &Consumer{
		client, keys, cfg.HTTPEndpoints(), tempDir, maxJobRetries}, nil
}

// Close all the internal connections of the consumer
func (c *Consumer) Close() {}

// Loop start the event loop for consuming job messages
func (c *Consumer) Loop() error {
	// Select the lowest alphabetical keyID to be used for signing the subscription request
	// This is an arbitrary choice which has no impact on the content of the messages.
	ch, err := c.client.SubscribeNewJobs(c.keys.firstKeyID())
	if err != nil {
		return errors.Wrap(err, "could not start job subscription")
	}

	for msg := range ch {
		if err := c.handle(&msg); err != nil {
			LogError.Println(errors.Wrap(err, "Error in job handler"))
		}
	}

	return nil
}

func (c *Consumer) handle(msg *amqp.Delivery) error {
	startTime := time.Now()

	var job UnprocessedJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		return errors.Wrap(err, "could not unmarshal queue message")
	}

	if len(job.Dependencies) > 0 {
		// Wait for job dependencies to finish
		depStatus, err := c.client.WaitForJobs(job.Dependencies, job.Repository)
		if err != nil {
			if err := c.postJobStatus(
				&job, startTime, time.Now(), false, err.Error()); err != nil {
				msg.Nack(false, true)
				return errors.Wrap(err, "posting job status to server failed")
			}
			msg.Nack(false, true)
			return errors.Wrap(err, "waiting for job dependencies failed")
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
			if err := c.postJobStatus(
				&job, startTime, time.Now(), false, err.Error()); err != nil {
				msg.Nack(false, true)
				return errors.Wrap(err, "posting job status to server failed")
			}
		}
	}

	LogInfo.Println("Start publishing job:", job.ID.String())

	task := func() error {
		return job.process(c.tempDir)
	}

	success := false
	errMsg := ""
	retry := 0
	for retry <= c.maxJobRetries {
		err := runTransaction(&job, task)
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

	// Publish the processed job status to the job server
	if err := c.postJobStatus(
		&job, startTime, finishTime, success, errMsg); err != nil {
		msg.Nack(false, true)
		return errors.Wrap(err, "posting job status to server failed")
	}

	msg.Ack(false)
	result := "failed"
	if success {
		result = "success"
	}
	LogInfo.Printf(
		"Finished publishing job: %v, %v\n", job.ID.String(), result)

	return nil
}

func (c *Consumer) postJobStatus(
	j *UnprocessedJob, t0 time.Time, t1 time.Time, success bool, errMsg string) error {

	processed := ProcessedJob{
		UnprocessedJob: *j,
		StartTime:      t0,
		FinishTime:     t1,
		Successful:     success,
		ErrorMessage:   errMsg,
	}

	// Post job status to the job server
	pubStat, err := c.client.PostJobStatus(&processed)
	if err != nil {
		return errors.Wrap(err, "Could not post job status")
	}

	if pubStat.Status != "ok" {
		return errors.New(
			fmt.Sprintf("Posting job status request failed: %s", pubStat.Reason))
	}

	return nil
}
