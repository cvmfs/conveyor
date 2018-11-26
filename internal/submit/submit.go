package submit

import (
	"fmt"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/jobdb"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/pkg/errors"
)

// Run - runs the new job submission process
func Run(
	spec *job.Specification,
	qCfg *queue.Config,
	jCfg *jobdb.Config,
	wait bool) error {

	pub, err := queue.NewClient(qCfg, queue.PublisherConnection)
	if err != nil {
		return errors.Wrap(err, "could not create publisher connection")
	}
	defer pub.Close()

	newJob, err := job.CreateJob(spec)
	if err != nil {
		return errors.Wrap(err, "could not create job object")
	}

	log.Info.Printf("Job description:\n%+v\n", newJob)

	if err := pub.Publish(queue.NewJobExchange, "", newJob); err != nil {
		return errors.Wrap(err, "job description publishing failed")
	}

	// Optionally wait for completion of the job
	if wait {
		consumer, err := queue.NewClient(qCfg, queue.ConsumerConnection)
		if err != nil {
			return errors.Wrap(err, "could not create consumer connection")
		}
		stats, err := job.WaitForJobs(
			[]string{newJob.ID.String()}, consumer, jCfg.JobDBURL())
		if err != nil {
			return errors.Wrap(err, "waiting for job completion failed")
		}

		if !stats[0].Successful {
			fmt.Printf("{\"Status\": \"error\", \"ID\": \"%s\"}\n", newJob.ID)
			return nil
		}
	}

	fmt.Printf("{\"Status\": \"ok\", \"ID\": \"%s\"}\n", newJob.ID)

	return nil
}
