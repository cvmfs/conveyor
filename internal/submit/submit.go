package submit

import (
	"fmt"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/pkg/errors"
)

// Run - runs the new job submission process
func Run(jparams *job.Parameters, qcfg *queue.Config) error {
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

	if err := conn.Publish(queue.NewJobExchange, queue.RoutingKey, job); err != nil {
		return errors.Wrap(err, "job description publishing failed")
	}

	fmt.Printf("{\"Status\": \"ok\", \"ID\": \"%s\"}\n", job.ID)

	return nil
}
