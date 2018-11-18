package consume

import (
	"encoding/json"
	"os"
	"path"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	getter "github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// Mock - enable mocking the CVMFS transaction
var Mock bool

func init() {
	Mock = false
	v := os.Getenv("CVMFS_MOCKED_JOB_CONSUMER")
	if v == "true" || v == "yes" || v == "on" {
		Mock = true
	}
}

// Run - runs the job consumer
func Run(qcfg queue.Config, tempDir string) error {
	// Create temporary dir
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

	var desc job.Description
	for j := range jobs {
		if err := json.Unmarshal(j.Body, &desc); err != nil {
			log.Error.Println(
				errors.Wrap(err, "could not unmarshal job description"))
			j.Nack(false, false)
			continue
		}

		log.Info.Println("Start publishing job:", desc.ID.String())

		task := func() error {
			targetDir := path.Join("/cvmfs", desc.Repo, desc.Path)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return errors.Wrap(err, "could not create target dir")
			}
			log.Info.Println("Downloading payload:", desc.Payload)
			if err := getter.Get(targetDir, desc.Payload); err != nil {
				return errors.Wrap(err, "could not download payload")
			}
			return nil
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
