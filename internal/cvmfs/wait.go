package cvmfs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

const maxWait = 2 * 3600 // max 2h to wait for a job dependency to finish

const maxQueryRetries = 50 // max number of job DB query retries

// WaitForJobs wait for the completion of a set of jobs referenced through theirs
// unique ids. The job status is obtained from the completed job notification channel
// of the job queue and from the job DB service
func WaitForJobs(ids []string, q *QueueClient, jobDBURL string) ([]JobStatus, error) {
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = false
	}
	jobStatuses := map[uuid.UUID]bool{}

	// Subscribe to notifications from the completed job channel
	notifications := make(chan JobStatus)
	notifQuit := make(chan bool)
	if err := listen(idMap, q, notifications, notifQuit); err != nil {
		return []JobStatus{}, errors.Wrap(err, "could not subscribe to notifications")
	}

	// Query the job DB for the JobStatus of completed jobs
	queryResults := make(chan JobStatus)
	queryQuit := make(chan bool)
	ch := query(ids, jobDBURL, queryResults, queryQuit)

	LogInfo.Println("Waiting for jobs")

	stop := false
	for !stop {
		select {
		case e := <-ch:
			if e != nil {
				close(notifQuit)
				close(queryQuit)
				return []JobStatus{}, errors.Wrap(e, "could not perform job DB query")
			}
		case j := <-notifications:
			jobStatuses[j.ID] = j.Successful
			LogInfo.Println("(Notification) job finished:", j)
			if !j.Successful || len(ids) == len(jobStatuses) {
				close(notifQuit)
				close(queryQuit)
				stop = true
			}
		case j := <-queryResults:
			jobStatuses[j.ID] = j.Successful
			LogInfo.Println("(Query result) job finished:", j)
			if !j.Successful || len(ids) == len(jobStatuses) {
				close(notifQuit)
				close(queryQuit)
				stop = true
			}
		case <-time.After(maxWait * time.Second):
			close(notifQuit)
			close(queryQuit)
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

func listen(
	ids map[string]bool,
	q *QueueClient,
	notifications chan<- JobStatus,
	quit <-chan bool) error {

	jobs, err := q.Chan.Consume(
		q.CompletedJobQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "could not start consuming jobs")
	}

	go func() {
		ch := q.Chan.NotifyClose(make(chan *amqp.Error))
		err := <-ch
		LogError.Println(
			errors.Wrap(err, "connection to job queue closed"))
		os.Exit(1)
	}()

	go func() {
	L:
		for {
			select {
			case j := <-jobs:
				var stat JobStatus
				if err := json.Unmarshal(j.Body, &stat); err != nil {
					LogError.Println(err)
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

func query(ids []string, jobDBURL string, results chan<- JobStatus, quit <-chan bool) chan error {
	ch := make(chan error)

	go func() {
		w := defaultWaiter()
		retry := 0

	L:
		for retry < maxQueryRetries {
			req, err := http.NewRequest("GET", jobDBURL, nil)
			if err != nil {
				errors.Wrap(err, "could not create GET request")
			}
			q := req.URL.Query()
			q["id"] = ids
			q.Set("full", "false")
			req.URL.RawQuery = q.Encode()

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				ch <- errors.Wrap(err, "Getting job status from DB failed")
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				ch <- errors.New(fmt.Sprintf("GET request failed: %v", resp.Status))
			}

			buf2, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				ch <- errors.Wrap(err, "Reading reply body failed")
			}

			var reply GetJobReply
			if err := json.Unmarshal(buf2, &reply); err != nil {
				ch <- errors.Wrap(err, "JSON decoding of reply failed")
			}

			if reply.Status != "ok" {
				ch <- errors.Wrap(
					err, fmt.Sprintf("Getting job status failed: %s", reply.Reason))
			}

			for _, j := range reply.IDs {
				results <- j
			}

			w.wait()
			select {
			case <-quit:
				break L
			default:
			}
		}

		ch <- nil
	}()

	return ch
}

const initRetryDelay = 5   // seconds
const maxRetryDelay = 1800 // seconds

type waiter struct {
	currentDelay int
	initDelay    int
	maxDelay     int
}

func defaultWaiter() waiter {
	return waiter{
		currentDelay: initRetryDelay,
		initDelay:    initRetryDelay,
		maxDelay:     maxRetryDelay,
	}
}

func newWaiter(initDelay, maxDelay int) waiter {
	return waiter{initDelay, initDelay, maxDelay}
}

func (w *waiter) wait() {
	time.Sleep(time.Duration(w.currentDelay) * time.Second)
	w.currentDelay *= 2
	if w.currentDelay > w.maxDelay {
		w.currentDelay = w.maxDelay
	}
}

func (w *waiter) reset() {
	w.currentDelay = w.initDelay
}
