package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/cvmfs"
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
		if err := cvmfs.RunConsume(qCfg, jCfg, tempDir, *maxJobRetries); err != nil {
			cvmfs.LogError.Println(err)
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
