package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/submit"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var repo string
var payload string
var path string
var script string
var scriptArgs string
var remoteScript *bool
var deps *[]string

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job",
	Long:  "Submit a publishing job to a queue",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var params queue.Parameters
		if err := viper.Sub("rabbitmq").Unmarshal(&params); err != nil {
			log.Error.Println("Could not read RabbitMQ creds")
			os.Exit(1)
		}
		submit.Run(repo, payload, path, script, scriptArgs, *remoteScript, *deps, params)
	},
}

func init() {
	submitCmd.Flags().StringVar(&repo, "repo", "", "target CVMFS repository")
	submitCmd.MarkFlagRequired("repo")
	submitCmd.Flags().StringVar(&payload, "payload", "", "payload URL")
	submitCmd.MarkFlagRequired("payload")
	submitCmd.Flags().StringVar(&path, "path", "/", "target path inside the repository")
	submitCmd.Flags().StringVar(
		&script, "script", "", "script to run at the end of CVMFS transaction")
	submitCmd.Flags().StringVar(
		&scriptArgs, "script-args", "", "arguments of the transaction script")
	remoteScript = submitCmd.Flags().Bool(
		"remote-script", false, "transaction script is a remote script")
	deps = submitCmd.Flags().StringSlice(
		"deps", []string{}, "comma-separate list of job dependency UUIDs")
}
