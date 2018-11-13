package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/constants"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
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
var deps string

func init() {
	submitCmd.Flags().StringVar(&repo, "repo", "", "target CVMFS repository")
	submitCmd.MarkFlagRequired("repo")
	submitCmd.Flags().StringVar(&payload, "payload", "", "payload URL")
	submitCmd.MarkFlagRequired("payload")
	submitCmd.Flags().StringVar(&path, "path", "/", "target path inside the repository")
	submitCmd.Flags().StringVar(&script, "script", "", "script to run at the end of CVMFS transaction")
	submitCmd.Flags().StringVar(&scriptArgs, "script-args", "", "arguments of the transaction script")
	submitCmd.Flags().Bool("remote-script", false, "transaction script is a remote script")
	submitCmd.Flags().StringVar(&deps, "deps", "", "comma-separate list of job dependency UUIDs")
}

func runSubmit(cmd *cobra.Command, args []string) {
	connection, err := amqp.Dial("amqp://" + viper.GetString("rabbitmq.url"))
	if err != nil {
		log.Error.Println("Could not open AMQP connection:", err)
		os.Exit(1)
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		log.Error.Println("Could not open AMQP channel:", err)
		os.Exit(1)
	}

	err = channel.ExchangeDeclare(constants.NewJobExchange, "direct", true, false, false, false, nil)
	if err != nil {
		log.Error.Println("Could not create exchange:", err)
		os.Exit(1)
	}
}
