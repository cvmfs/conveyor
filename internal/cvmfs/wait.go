package cvmfs

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

const maxQueryRetries = 50 // max number of job server query retries

// listen is a helper function for JobClient.WaitForJobs. Listens for completion
// status messages from the queue and publishes them to the notifications channel, if
// they correspond to any job in "ids"
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

// listen is a helper function for JobClient.WaitForJobs. Queries the conveyor server for
// the completion status of jobs identified by "ids", forwarding the messages on the
// results channel
func query(ids []string, client *JobClient, repo string, results chan<- JobStatus, quit <-chan struct{}) chan error {
	ch := make(chan error)

	go func() {
		w := DefaultWaiter()
		retry := 0

	L:
		for retry < maxQueryRetries {
			reply, err := client.GetJobStatus(ids, repo)
			if err != nil {
				ch <- errors.Wrap(err, "could not retrieve job status from server")
				return
			}

			if reply.Status != "ok" {
				ch <- errors.Wrap(
					err, fmt.Sprintf("Getting job status failed: %s", reply.Reason))
			}

			for _, j := range reply.IDs {
				results <- j
			}

			w.Wait()
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
