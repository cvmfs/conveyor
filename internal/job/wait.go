package job

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

const maxWait = 12 * 3600 // max 12h to wait for job dependencies to finish

// WaitForJobs wait for the completion of a set of jobs referenced through theirs
// unique ids. The job status is obtained from the completed job notification channel
// of the job queue and from the job DB service
func WaitForJobs(ids []string, q *queue.Client, jobDBURL string) ([]Status, error) {
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = false
	}
	jobStatuses := map[uuid.UUID]bool{}

	// Subscribe to notifications from the completed job channel
	notifications, err := listen(idMap, q)
	if err != nil {
		return []Status{}, errors.Wrap(err, "could not subscribe to notifications")
	}

	// Query the job DB for the status of completed jobs
	queryResults, err := query(ids, jobDBURL)
	if err != nil {
		return []Status{}, errors.Wrap(err, "could not perform job DB query")
	}

	stop := false
	for !stop {
		select {
		case j := <-notifications:
			log.Info.Println("(Notification) job completed:", j)
			jobStatuses[j.ID] = j.Successful
			if !j.Successful || len(ids) == len(jobStatuses) {
				stop = true
			}
		case j := <-queryResults:
			log.Info.Println("(Query) job completed:", j)
			jobStatuses[j.ID] = j.Successful
			if !j.Successful || len(ids) == len(jobStatuses) {
				stop = true
			}
		case <-time.After(maxWait * time.Second):
			return []Status{}, errors.New("timeout")
		}
	}

	st := []Status{}
	for k, v := range jobStatuses {
		st = append(st, Status{ID: k, Successful: v})
	}

	return st, nil
}

func listen(ids map[string]bool, q *queue.Client) (chan *Status, error) {
	jobs, err := q.Chan.Consume(
		q.CompletedJobQueue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not start consuming jobs")
	}

	go func() {
		ch := q.Chan.NotifyClose(make(chan *amqp.Error))
		err := <-ch
		log.Error.Println(
			errors.Wrap(err, "connection to job queue closed"))
		os.Exit(1)
	}()

	ch := make(chan *Status)

	go func() {
		for j := range jobs {
			var stat Status
			if err := json.Unmarshal(j.Body, &stat); err != nil {
				log.Error.Println(err)
				j.Nack(false, false)
				ch <- nil
			}
			id := stat.ID.String()
			_, pres := ids[id]
			if pres {
				ch <- &stat
			}
		}
	}()

	return ch, nil
}

func query(ids []string, jobDBURL string) (chan *Status, error) {
	req, err := http.NewRequest("GET", jobDBURL, nil)
	if err != nil {
		errors.Wrap(err, "could not create GET request")
	}
	q := req.URL.Query()
	q.Set("ids", strings.Join(ids, ","))
	q.Set("full", "false")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Getting job status from DB failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("GET request failed: %v", resp.Status))
	}

	buf2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Reading reply body failed")
	}

	var reply GetJobReply
	if err := json.Unmarshal(buf2, &reply); err != nil {
		return nil, errors.Wrap(err, "JSON decoding of reply failed")
	}

	if reply.Status != "ok" {
		return nil, errors.Wrap(
			err, fmt.Sprintf("Getting job status failed: %s", reply.Reason))
	}

	ch := make(chan *Status)

	go func() {
		for _, j := range reply.IDs {
			ch <- &j
		}
	}()

	return nil, nil
}
