package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/cvmfs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var maxJobRetries *int
var tempDir string

var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "Consume jobs",
	Long:  "Consume publishing jobs from the queue",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		qCfg, err := cvmfs.ReadQueueConfig()
		if err != nil {
			cvmfs.LogError.Println(err)
			os.Exit(1)
		}
		jCfg, err := cvmfs.ReadJobDbConfig()
		if err != nil {
			cvmfs.LogError.Println(err)
			os.Exit(1)
		}

		keys, err := cvmfs.ReadKeys(jCfg.KeyDir)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not read API keys from file"))
			os.Exit(1)
		}

		// Create temporary dir
		os.RemoveAll(tempDir)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not create temp dir"))
			os.Exit(1)
		}
		defer os.RemoveAll(tempDir)

		consumer, err := cvmfs.NewConsumer(
			qCfg, keys, jCfg.JobDBURL(), tempDir, *maxJobRetries)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not create RabbitMQ message consumer"))
			os.Exit(1)
		}
		defer consumer.Close()

		cvmfs.LogInfo.Println("Entering consumer loop")

		if err := consumer.Loop(); err != nil {
			cvmfs.LogInfo.Println(errors.Wrap(err, "error in consumer loop"))
			os.Exit(1)
		}
	},
}

func init() {
	maxJobRetries = consumeCmd.Flags().Int(
		"max-job-retries", 3, "maximum number of retries for processing a job before "+
			"giving up and recording it as a failed job")
	consumeCmd.Flags().StringVar(
		&tempDir, "temp-dir", "/tmp/cvmfs-consumer", "temporary directory for use during CVMFS transaction")
}
