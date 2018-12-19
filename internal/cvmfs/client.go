package cvmfs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/pkg/errors"
)

// JobClient offers functionality for interacting with the job server
type JobClient struct {
	keys      *Keys
	endpoints HTTPEndpoints
	qcl       *QueueClient
}

// NewJobClient constructs a new JobClient object using a configuration object and a set
// of keys
func NewJobClient(cfg *Config, keys *Keys) (*JobClient, error) {
	q, err := NewQueueClient(&cfg.Queue, consumerConnection)
	if err != nil {
		return nil, errors.Wrap(err, "could not create queue connection")
	}
	return &JobClient{keys, cfg.HTTPEndpoints(), q}, nil
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
func (c *JobClient) WaitForJobs(ids []string, repo string) ([]JobStatus, error) {
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

	LogInfo.Println("Waiting for jobs")

L:
	for {
		select {
		case e := <-ch:
			if e != nil {
				return []JobStatus{}, errors.Wrap(e, "could not perform job server query")
			}
		case j := <-notifications:
			jobStatuses[j.ID] = j.Successful
			LogInfo.Println("(Notification) job finished:", j)
			if !j.Successful || len(ids) == len(jobStatuses) {
				break L
			}
		case j := <-queryResults:
			jobStatuses[j.ID] = j.Successful
			LogInfo.Println("(Query result) job finished:", j)
			if !j.Successful || len(ids) == len(jobStatuses) {
				break L
			}
		case <-time.After(MaxJobDuration * time.Second):
			return []JobStatus{}, errors.New("timeout")
		}
	}

	LogInfo.Println("All jobs complete. Continuing")

	st := []JobStatus{}
	for k, v := range jobStatuses {
		st = append(st, JobStatus{ID: k, Successful: v})
	}

	return st, nil
}

// GetJobStatus queries the status of a set of jobs from the server
func (c *JobClient) GetJobStatus(ids []string, repo string) (*GetJobStatusReply, error) {
	req, err := http.NewRequest("GET", c.endpoints.CompletedJobs(true), nil)
	if err != nil {
		errors.Wrap(err, "Could not create GET request")
	}
	q := req.URL.Query()
	q["id"] = ids
	q.Set("full", "false")
	req.URL.RawQuery = q.Encode()

	// Compute message HMAC
	keyID, secret, err := c.keys.getKeyForRepo(repo)
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve request signing key")
	}
	buf := []byte(req.URL.RawQuery)
	hmac := base64.StdEncoding.EncodeToString(computeHMAC(buf, secret))
	req.Header.Add("Authorization", fmt.Sprintf("%v %v", keyID, hmac))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Getting job status from server failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("GET request failed: %v", resp.Status))
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

	reply, err := c.postMsg(buf, job.Repository, c.endpoints.NewJobs(true))
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

	reply, err := c.postMsg(buf, job.Repository, c.endpoints.CompletedJobs(true))
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
func (c *JobClient) postMsg(msg []byte, repository string, url string) ([]byte, error) {
	// Compute message HMAC
	keyID, secret, err := c.keys.getKeyForRepo(repository)
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve request signing key")
	}
	hmac := base64.StdEncoding.EncodeToString(computeHMAC(msg, secret))

	rdr := bytes.NewReader(msg)

	// Post processed job status to the job server
	req, err := http.NewRequest("POST", url, rdr)
	if err != nil {
		errors.Wrap(err, "could not create POST request")
	}
	req.Header.Add("Authorization", fmt.Sprintf("%v %v", keyID, hmac))
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Posting job status to server failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Post request failed: %v", resp.Status))
	}

	buf2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Reading reply body failed")
	}

	return buf2, nil
}
