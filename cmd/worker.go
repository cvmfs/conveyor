package commands

import (
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run conveyor worker",
	Long:  "Run the conveyor worker daemon",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := cvmfs.ReadConfig()
		if err != nil {
			cvmfs.LogError.Println(err)
			os.Exit(1)
		}

		keys, err := cvmfs.LoadKeys(cfg.KeyDir)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not read API keys from file"))
			os.Exit(1)
		}

		// Create temporary dir
		tempDir := cfg.Worker.TempDir

		os.RemoveAll(tempDir)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not create temp dir"))
			os.Exit(1)
		}
		defer os.RemoveAll(tempDir)

		worker, err := cvmfs.NewWorker(cfg, keys)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not create queue consumer"))
			os.Exit(1)
		}
		defer worker.Close()

		cvmfs.LogInfo.Printf("Worker %v started.\n", cfg.Worker.Name)

		if err := worker.Loop(); err != nil {
			cvmfs.LogInfo.Println(errors.Wrap(err, "error in worker loop"))
			os.Exit(1)
		}
	},
}
