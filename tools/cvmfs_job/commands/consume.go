package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/consume"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tempDir string

var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "Consume jobs",
	Long:  "Consume publishing jobs from the queue",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var params queue.Parameters
		if err := viper.Sub("rabbitmq").Unmarshal(&params); err != nil {
			log.Error.Println("Could not read RabbitMQ creds")
			os.Exit(1)
		}
		consume.Run(params, tempDir)
	},
}

func init() {
	consumeCmd.Flags().StringVar(
		&tempDir, "temp-dir", "/tmp/cvmfs-consumer", "temporary directory for use during CVMFS transaction")
}
