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
	v := os.Getenv("CONVEYOR_MOCK_WORKER")
	if v == "true" || v == "yes" || v == "on" {
		mock = true
	}
}

// Worker encapsulates the loop where job descriptions received from the conveyor server
// are downloaded and processed
type Worker struct {
	name          string
	maxJobRetries int
	tempDir       string
	client        *JobClient
	sharedKey     string
	endpoints     HTTPEndpoints
	timeout       int
}

// NewWorker creates a new Worker object using a config object
func NewWorker(cfg *Config) (*Worker, error) {

	client, err := NewJobClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not create a queue client")
	}

	return &Worker{
		cfg.Worker.Name, cfg.Worker.JobRetries, cfg.Worker.TempDir,
		client, cfg.SharedKey, cfg.HTTPEndpoints(), cfg.JobWaitTimeout}, nil
}

// Close all the internal connections of the Worker object
func (w *Worker) Close() {
	w.client.Close()
}

// Loop subscribes to the new job messages from the conveyor server and processes them
// one by one
func (w *Worker) Loop() error {
	// Select the lowest alphabetical keyID to be used for signing the subscription request
	// This is an arbitrary choice which has no impact on the content of the messages.
	ch, err := w.client.SubscribeNewJobs(w.sharedKey)
	if err != nil {
		return errors.Wrap(err, "could not start job subscription")
	}

	for msg := range ch {
		if err := w.handle(&msg); err != nil {
			Log.Error().Err(err).Msg("Error in job handler")
		}
	}

	return nil
}

// handle a job message received from the conveyor server; involves deserializing the job
// description and processing the job
func (w *Worker) handle(msg *amqp.Delivery) error {

	var job UnprocessedJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		return errors.Wrap(err, "could not unmarshal queue message")
	}

	if len(job.Dependencies) > 0 {
		// Wait for job dependencies to finish
		depStatus, err := w.client.WaitForJobs(job.Dependencies, job.Repository, w.timeout)
		if err != nil {
			t := time.Now()
			if err := w.postJobStatus(
				&job, w.name, t, t, false, err.Error()); err != nil {
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
			err := fmt.Errorf("failed job dependencies: %v", failed)
			Log.Error().Err(err).Msg("Error in job handler")
			t := time.Now()
			if err := w.postJobStatus(
				&job, w.name, t, t, false, err.Error()); err != nil {
				msg.Nack(false, true)
				return errors.Wrap(err, "posting job status to server failed")
			}
		}
	}

	Log.Info().Str("job_id", job.ID.String()).Msg("Start publishing job")
	startTime := time.Now()

	task := func() error {
		return job.process(w.tempDir)
	}

	success := false
	errMsg := ""
	retry := 0
	for retry <= w.maxJobRetries {
		err := runTransaction(job.Repository, job.LeasePath, task)
		if err != nil {
			errMsg = err.Error()
			Log.Error().Err(err).Msg("Error in job handler")
			retry++
			Log.Info().Msg("Transaction failed.")
			if retry <= w.maxJobRetries {
				Log.Info().Msgf(" Retrying: %v/%v\n", retry, w.maxJobRetries)
			}
		} else {
			success = true
			break
		}
	}

	finishTime := time.Now()

	// Publish the processed job status to the job server
	if err := w.postJobStatus(
		&job, w.name, startTime, finishTime, success, errMsg); err != nil {
		msg.Nack(false, true)
		return errors.Wrap(err, "posting job status to server failed")
	}

	msg.Ack(false)
	Log.Info().
		Str("job_id", job.ID.String()).
		Bool("success", success).
		Msg("Finished publishing job")

	return nil
}

func (w *Worker) postJobStatus(
	j *UnprocessedJob, workerName string, t0 time.Time, t1 time.Time, success bool, errMsg string) error {

	processed := ProcessedJob{
		UnprocessedJob: *j,
		WorkerName:     workerName,
		StartTime:      t0,
		FinishTime:     t1,
		Successful:     success,
		ErrorMessage:   errMsg,
	}

	// Post job status to the job server
	pubStat, err := w.client.PostJobStatus(&processed)
	if err != nil {
		return errors.Wrap(err, "Could not post job status")
	}

	if pubStat.Status != "ok" {
		return fmt.Errorf("Posting job status request failed: %s", pubStat.Reason)
	}

	return nil
}
