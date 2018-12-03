package cvmfs

import (
	"fmt"

	"github.com/pkg/errors"
)

// RunSubmit - runs the new job submission process
func RunSubmit(
	spec *JobSpecification,
	qCfg *QueueConfig,
	jCfg *JobDbConfig,
	wait bool) error {

	pub, err := NewQueueClient(qCfg, PublisherConnection)
	if err != nil {
		return errors.Wrap(err, "could not create publisher connection")
	}
	defer pub.Close()

	newJob, err := CreateJob(spec)
	if err != nil {
		return errors.Wrap(err, "could not create job object")
	}

	LogInfo.Printf("Job description:\n%+v\n", newJob)

	if err := pub.Publish(NewJobExchange, "", newJob); err != nil {
		return errors.Wrap(err, "job description publishing failed")
	}

	// Optionally wait for completion of the job
	if wait {
		consumer, err := NewQueueClient(qCfg, ConsumerConnection)
		if err != nil {
			return errors.Wrap(err, "could not create consumer connection")
		}
		stats, err := WaitForJobs(
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
