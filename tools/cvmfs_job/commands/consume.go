package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/consume"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/pkg/errors"
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
		var qcfg queue.Config
		if err := viper.Sub("rabbitmq").Unmarshal(&qcfg); err != nil {
			log.Error.Println(
				errors.Wrap(err, "could not read RabbitMQ creds"))
			os.Exit(1)
		}
		if err := consume.Run(qcfg, tempDir); err != nil {
			log.Error.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	consumeCmd.Flags().StringVar(
		&tempDir, "temp-dir", "/tmp/cvmfs-consumer", "temporary directory for use during CVMFS transaction")
}
