package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/jobdb"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Start job DB",
	Long:  "Start the job database service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var jobDbCfg jobdb.Config
		if err := viper.Sub("jobdb").Unmarshal(&jobDbCfg); err != nil {
			log.Error.Println(
				errors.Wrap(err, "could not read job db configuration"))
			os.Exit(1)
		}
		if err := jobdb.Run(jobDbCfg); err != nil {
			log.Error.Println(err)
			os.Exit(1)
		}
	},
}
