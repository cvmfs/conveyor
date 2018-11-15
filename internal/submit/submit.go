package submit

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/streadway/amqp"
)

// Run - runs the new job submission process
func Run(jparams job.Parameters, qcfg queue.Config) {
	conn, err := queue.NewConnection(qcfg)
	if err != nil {
		log.Error.Println("Could not create job queue connection:", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := conn.SetupTopology(); err != nil {
		log.Error.Println("Could not set up RabbitMQ topology:", err)
		os.Exit(1)
	}

	job, err := job.CreateJob(jparams)
	if err != nil {
		log.Error.Println("Could not create job object:", err)
		os.Exit(1)
	}

	log.Info.Printf("Job description:\n%+v\n", job)

	body, err := json.Marshal(job)
	if err != nil {
		log.Error.Println("Could not marshal job into JSON:", err)
		os.Exit(1)
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
		log.Error.Println("Could not publish job:", err)
		os.Exit(1)
	}

	fmt.Printf("{\"Status\": \"ok\", \"ID\": \"%s\"}\n", job.ID)
}
