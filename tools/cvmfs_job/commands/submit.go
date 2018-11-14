package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job",
	Long:  "Submit a publishing job to a queue",
	Args:  cobra.NoArgs,
	Run:   runSubmit,
}

var repo string
var payload string
var path string
var script string
var scriptArgs string
var remoteScript *bool
var deps string

func init() {
	submitCmd.Flags().StringVar(&repo, "repo", "", "target CVMFS repository")
	submitCmd.MarkFlagRequired("repo")
	submitCmd.Flags().StringVar(&payload, "payload", "", "payload URL")
	submitCmd.MarkFlagRequired("payload")
	submitCmd.Flags().StringVar(&path, "path", "/", "target path inside the repository")
	submitCmd.Flags().StringVar(&script, "script", "", "script to run at the end of CVMFS transaction")
	submitCmd.Flags().StringVar(&scriptArgs, "script-args", "", "arguments of the transaction script")
	remoteScript = submitCmd.Flags().Bool("remote-script", false, "transaction script is a remote script")
	submitCmd.Flags().StringVar(&deps, "deps", "", "comma-separate list of job dependency UUIDs")
}

func runSubmit(cmd *cobra.Command, args []string) {
	conn, err := queue.NewConnection(
		viper.GetString("rabbitmq.username"),
		viper.GetString("rabbitmq.password"),
		viper.GetString("rabbitmq.url"),
		viper.GetString("rabbitmq.vhost"),
		viper.GetInt("rabbitmq.port"))
	if err != nil {
		log.Error.Println("Could not create job queue connection:", err)
		os.Exit(1)
	}
	defer conn.Close()

	err = conn.Chan.ExchangeDeclare(queue.NewJobExchange, "direct", true, false, false, false, nil)
	if err != nil {
		log.Error.Println("Could not create exchange:", err)
		os.Exit(1)
	}

	job, err := job.CreateJob(repo, payload, path, script, scriptArgs, *remoteScript, deps)
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
		queue.NewJobExchange, queue.RoutingKey, false, false, msg)
	if err != nil {
		log.Error.Println("Could not publish job:", err)
		os.Exit(1)
	}

	fmt.Printf("{\"Status\": \"ok\", \"ID\": \"%s\"}\n", job.ID)
}
