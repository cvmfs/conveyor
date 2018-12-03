package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/cvmfs"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Start job DB",
	Long:  "Start the job database service",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := cvmfs.ReadJobDbConfig()
		if err != nil {
			cvmfs.LogError.Println(err)
			os.Exit(1)
		}
		if err := cvmfs.RunJobDb(cfg); err != nil {
			cvmfs.LogError.Println(err)
			os.Exit(1)
		}
	},
}
