package commands

import (
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the job server",
	Long:  "Start the job server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := cvmfs.ReadConfig(cvmfs.ServerProfile)
		if err != nil {
			cvmfs.Log.Error().Err(err).Msg("config error")
			os.Exit(1)
		}
		if rootCmd.PersistentFlags().Changed("timeout") {
			cfg.JobWaitTimeout = jobWaitTimeout
		}
		cvmfs.Log.Info().Msg("CVMFS job server starting")

		if err := cvmfs.StartServer(cfg); err != nil {
			cvmfs.Log.Error().Err(err).Msg("could not start conveyor server")
			os.Exit(1)
		}
	},
}
