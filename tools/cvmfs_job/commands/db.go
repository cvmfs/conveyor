package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/jobdb"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Start job DB",
	Long:  "Start the job database service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := jobdb.ReadConfig()
		if err != nil {
			log.Error.Println(err)
			os.Exit(1)
		}
		if err := jobdb.Run(cfg); err != nil {
			log.Error.Println(err)
			os.Exit(1)
		}
	},
}
