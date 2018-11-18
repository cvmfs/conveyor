package submit

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// Run - runs the new job submission process
func Run(jparams job.Parameters, qcfg queue.Config) error {
	conn, err := queue.NewConnection(qcfg)
	if err != nil {
		return errors.Wrap(err, "could not create job queue connection")
	}
	defer conn.Close()

	job, err := job.CreateJob(jparams)
	if err != nil {
		return errors.Wrap(err, "could not create job object")
	}

	log.Info.Printf("Job description:\n%+v\n", job)

	body, err := json.Marshal(job)
	if err != nil {
		return errors.Wrap(err, "could not marshal job into JSON")
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/json",
		Body:         []byte(body),
	}

	err = conn.Chan.Publish(
		queue.NewJobExchange, queue.RoutingKey, true, false, msg)
	if err != nil {
		return errors.Wrap(err, "could not publish job")
	}

	fmt.Printf("{\"Status\": \"ok\", \"ID\": \"%s\"}\n", job.ID)

	return nil
}
