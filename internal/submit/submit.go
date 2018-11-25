package submit

import (
	"fmt"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/pkg/errors"
)

// Run - runs the new job submission process
func Run(spec *job.Specification, qcfg *queue.Config) error {
	client, err := queue.NewClient(qcfg, queue.PublisherConnection)
	if err != nil {
		return errors.Wrap(err, "could not create job queue connection")
	}
	defer client.Close()

	job, err := job.CreateJob(spec)
	if err != nil {
		return errors.Wrap(err, "could not create job object")
	}

	log.Info.Printf("Job description:\n%+v\n", job)

	if err := client.Publish(queue.NewJobExchange, "", job); err != nil {
		return errors.Wrap(err, "job description publishing failed")
	}

	fmt.Printf("{\"Status\": \"ok\", \"ID\": \"%s\"}\n", job.ID)

	return nil
}
