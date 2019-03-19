package commands

import (
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/spf13/cobra"
)

type workerCmdVars struct {
	name    string
	retries int
	tempDir string
}

var wrkvs workerCmdVars

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
		if cmd.Flags().Changed("job-wait-timeout") {
			cfg.JobWaitTimeout = jobWaitTimeout
		}
		if cmd.Flags().Changed("worker-name") {
			cfg.Worker.Name = wrkvs.name
		}
		if cmd.Flags().Changed("job-retries") {
			cfg.Worker.JobRetries = wrkvs.retries
		}
		if cmd.Flags().Changed("temp-dir") {
			cfg.Worker.TempDir = wrkvs.tempDir
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

func init() {
	workerCmd.Flags().StringVarP(&wrkvs.name, "worker-name", "n", "", "name of the worker daemon")
	workerCmd.Flags().IntVarP(&wrkvs.retries, "job-retries", "R", 0, "number of times the transaction script should be retried")
	workerCmd.Flags().StringVarP(&wrkvs.tempDir, "temp-dir", "T", "", "temporary directory used by the worker daemon")
}
