package commands

import (
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run conveyor worker",
	Long:  "Run the conveyor worker daemon",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cvmfs.InitLogging(os.Stderr, *logTimestamps)

		cfg, err := cvmfs.ReadConfig(cvmfs.WorkerProfile)
		if err != nil {
			cvmfs.Log.Error().Err(err).Msg("config error")
			os.Exit(1)
		}
		if rootCmd.PersistentFlags().Changed("timeout") {
			cfg.JobWaitTimeout = jobWaitTimeout
		}

		// Create temporary dir
		tempDir := cfg.Worker.TempDir

		os.RemoveAll(tempDir)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			cvmfs.Log.Error().Err(err).Msg("could not create temp dir")
			os.Exit(1)
		}
		defer os.RemoveAll(tempDir)

		worker, err := cvmfs.NewWorker(cfg)
		if err != nil {
			cvmfs.Log.Error().Err(err).Msg("could not create queue consumer")
			os.Exit(1)
		}
		defer worker.Close()

		cvmfs.Log.Info().Msgf("Worker %v started", cfg.Worker.Name)

		if err := worker.Loop(); err != nil {
			cvmfs.Log.Error().Err(err).Msg("error in worker loop")
			os.Exit(1)
		}
	},
}
