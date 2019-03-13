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

	"github.com/google/uuid"
	"github.com/streadway/amqp"

	"github.com/pkg/errors"
)

// Maximum number of retries for HTTP requests
const maxRequestRetries = 50

// Retry delay in seconds for the job status query
const queryRetryDelay = 30

// JobClient offers functionality for interacting with the job server
type JobClient struct {
	sharedKey string
	endpoints HTTPEndpoints
	qcl       *QueueClient
}

// NewJobClient constructs a new JobClient object using a configuration object and a set
// of keys
func NewJobClient(cfg *Config) (*JobClient, error) {
	q, err := NewQueueClient(&cfg.Queue, consumerConnection)
	if err != nil {
		return nil, errors.Wrap(err, "could not create queue connection")
	}
	return &JobClient{cfg.SharedKey, cfg.HTTPEndpoints(), q}, nil
}

// Close all the internal connections of the object
func (c *JobClient) Close() {
	c.qcl.Close()
}

// SubscribeNewJobs returns a channel with new job messages coming from the conveyor
// server
func (c *JobClient) SubscribeNewJobs(keyID string) (<-chan amqp.Delivery, error) {
	ch, err := c.qcl.Chan.Consume(
		c.qcl.NewJobQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not start consuming jobs")
	}

	return ch, nil
}

// WaitForJobs waits for the completion of a set of jobs referenced through theirs
// unique ids. The job status is obtained from the completed job notification channel
// of the job queue and from the job server
func (c *JobClient) WaitForJobs(
	ids []string, repo string, timeout int) ([]JobStatus, error) {
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = false
	}
	jobStatuses := map[uuid.UUID]bool{}

	// Channel used to send a quit signal to all goroutines spawned here
	quit := make(chan struct{})
	defer close(quit)

	// Subscribe to notifications from the completed job channel
	notifications := make(chan JobStatus)
	if err := listen(idMap, c.qcl, notifications, quit); err != nil {
		return []JobStatus{}, errors.Wrap(err, "could not subscribe to notifications")
	}

	// Query the job server for the status of completed jobs
	queryResults := make(chan JobStatus)
	ch := query(ids, c, repo, queryResults, quit)

	Log.Info().Msg("Waiting for jobs")

L:
	for {
		select {
		case e := <-ch:
			if e != nil {
				return []JobStatus{}, errors.Wrap(e, "could not perform job server query")
			}
		case j := <-notifications:
			jobStatuses[j.ID] = j.Successful
			Log.Info().
				Str("source", "notification").
				Str("job_id", j.ID.String()).
				Bool("success", j.Successful).
				Msg("job finished")
			if !j.Successful || len(ids) == len(jobStatuses) {
				break L
			}
		case j := <-queryResults:
			jobStatuses[j.ID] = j.Successful
			Log.Info().
				Str("source", "query").
				Str("job_id", j.ID.String()).
				Bool("success", j.Successful).
				Msgf("(Query result) job finished: %v", j)
			if !j.Successful || len(ids) == len(jobStatuses) {
				break L
			}
		case <-time.After(time.Duration(timeout) * time.Second):
			return []JobStatus{}, errors.New("timeout")
		}
	}

	Log.Info().Msg("All jobs complete. Continuing")

	st := []JobStatus{}
	for k, v := range jobStatuses {
		st = append(st, JobStatus{ID: k, Successful: v})
	}

	return st, nil
}

// GetJobStatus queries the status of a set of jobs from the server
func (c *JobClient) GetJobStatus(
	ids []string, repo string, full bool, quit <-chan struct{}) (*GetJobStatusReply, error) {

	req, err := http.NewRequest("GET", c.endpoints.CompletedJobs(true), nil)
	if err != nil {
		errors.Wrap(err, "Could not create GET request")
	}
	q := req.URL.Query()
	q["id"] = ids
	if full {
		q.Set("full", "true")
	} else {
		q.Set("full", "false")
	}
	req.URL.RawQuery = q.Encode()

	// Compute message HMAC
	buf := []byte(req.URL.RawQuery)
	hmac := base64.StdEncoding.EncodeToString(computeHMAC(buf, c.sharedKey))
	req.Header.Add("Authorization", fmt.Sprintf("%v", hmac))

	resp, err := makeRequest(req, quit)
	if err != nil {
		return nil, errors.Wrap(err, "Getting job status from server failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET request failed: %v", resp.Status)
	}

	buf2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Reading reply body failed")
	}

	var status GetJobStatusReply
	if err := json.Unmarshal(buf2, &status); err != nil {
		return nil, errors.Wrap(err, "JSON decoding of reply failed")
	}

	return &status, nil
}

// PostNewJob posts a new unprocessed job to the server
func (c *JobClient) PostNewJob(job *JobSpecification) (*PostNewJobReply, error) {
	buf, err := json.Marshal(job)
	if err != nil {
		return nil, errors.Wrap(err, "JSON encoding of job status failed")
	}

	quit := make(chan struct{})
	reply, err := c.postMsg(buf, job.Repository, c.endpoints.NewJobs(true), quit)
	if err != nil {
		return nil, errors.Wrap(err, "POST request failed")
	}

	var stat PostNewJobReply
	if err := json.Unmarshal(reply, &stat); err != nil {
		return nil, errors.Wrap(err, "JSON decoding of reply failed")
	}

	return &stat, nil
}

// PostJobStatus posts the status of a completed job to the server
func (c *JobClient) PostJobStatus(job *ProcessedJob) (*PostJobStatusReply, error) {
	buf, err := json.Marshal(job)
	if err != nil {
		return nil, errors.Wrap(err, "JSON encoding of job status failed")
	}

	quit := make(chan struct{})
	reply, err := c.postMsg(buf, job.Repository, c.endpoints.CompletedJobs(true), quit)
	if err != nil {
		return nil, errors.Wrap(err, "POST request failed")
	}

	var stat PostJobStatusReply
	if err := json.Unmarshal(reply, &stat); err != nil {
		return nil, errors.Wrap(err, "JSON decoding of reply failed")
	}

	return &stat, nil
}

// postMsg makes a POST request to the conveyor server located at "url" with the body
// provided in the "msg" slice. The message is signed with the key corresponding to
// "repository"
func (c *JobClient) postMsg(
	msg []byte, repository, url string, quit <-chan struct{}) ([]byte, error) {

	// Compute message HMAC
	hmac := base64.StdEncoding.EncodeToString(computeHMAC(msg, c.sharedKey))

	rdr := bytes.NewReader(msg)

	// Post processed job status to the job server
	req, err := http.NewRequest("POST", url, rdr)
	if err != nil {
		errors.Wrap(err, "could not create POST request")
	}
	req.Header.Add("Authorization", fmt.Sprintf("%v", hmac))
	req.Header.Add("Content-Type", "application/json")

	resp, err := makeRequest(req, quit)
	if err != nil {
		return nil, errors.Wrap(err, "Posting job status to server failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Post request failed: %v", resp.Status)
	}

	buf2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Reading reply body failed")
	}

	return buf2, nil
}

// RequestCancelled is an error value that signals a cancelled HTTP request
type RequestCancelled struct{}

func (c RequestCancelled) Error() string {
	return "Request cancelled"
}

// Helper function to perform an HTTP request with retries, backoff and cancellation
// On return, "err" is of type "RequestCancelled" if the request was cancelled
func makeRequest(req *http.Request, quit <-chan struct{}) (*http.Response, error) {
	w := DefaultWaiter()
	var resp *http.Response
	var err error
L:
	for retry := 0; retry < maxRequestRetries; retry++ {
		select {
		case <-quit:
			err = RequestCancelled{}
			break L
		default:
		}
		resp, err = http.DefaultClient.Do(req)
		if err == nil {
			break L
		}
		if retry < maxRequestRetries-1 {
			w.Wait()
		}
	}
	return resp, err
}

// listen is a helper function for WaitForJobs. Listens for completion status messages
// from the queue and publishes them to the notifications channel, if they correspond
// to any job in "ids"
func listen(
	ids map[string]bool,
	q *QueueClient,
	notifications chan<- JobStatus,
	quit <-chan struct{}) error {

	jobs, err := q.Chan.Consume(
		q.CompletedJobQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "could not start consuming jobs")
	}

	go func() {
	L:
		for {
			select {
			case j := <-jobs:
				var stat JobStatus
				if err := json.Unmarshal(j.Body, &stat); err != nil {
					Log.Error().Err(err).Msg("job status deserialization error")
					j.Nack(false, false)
					os.Exit(1) // Is there a better way to handle this than restarting?
				}
				id := stat.ID.String()
				_, pres := ids[id]
				if pres {
					notifications <- stat
				}
				j.Ack(false)
			case <-quit:
				break L
			}
		}
	}()

	return nil
}

// listen is a helper function for WaitForJobs. Repeatedly queries the conveyor
// server for the completion status of jobs identified by "ids", forwarding the
// messages on the results channel
func query(
	ids []string,
	client *JobClient,
	repo string,
	results chan<- JobStatus,
	quit <-chan struct{}) chan error {

	ch := make(chan error)

	go func() {
	L:
		for {
			select {
			case <-quit:
				break L
			default:
			}
			reply, err := client.GetJobStatus(ids, repo, false, quit)
			if err != nil {
				// Only forward the error value if there wasn't any cancellation
				if _, cancellation := err.(*RequestCancelled); !cancellation {
					ch <- errors.Wrap(err, "could not retrieve job status from server")
				}
				return
			}

			if reply.Status != "ok" {
				ch <- errors.Wrap(
					err, fmt.Sprintf("Getting job status failed: %s", reply.Reason))
			}

			for _, j := range reply.IDs {
				results <- j
			}

			time.Sleep(queryRetryDelay * time.Second)
		}
		close(ch)
	}()

	return ch
}
