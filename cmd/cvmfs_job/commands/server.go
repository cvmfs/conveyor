package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/cvmfs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the job server",
	Long:  "Start the job server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := cvmfs.ReadConfig()
		if err != nil {
			cvmfs.LogError.Println(err)
			os.Exit(1)
		}
		cvmfs.LogInfo.Println("CVMFS job server starting")

		keys, err := cvmfs.ReadKeys(cfg.KeyDir)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not read API key from file"))
			os.Exit(1)
		}

		backend, err := cvmfs.StartBackEnd(cfg.Backend)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not start service back-end"))
			os.Exit(1)
		}
		defer backend.Close()

		if err := cvmfs.StartFrontEnd(cfg.Port, backend, keys); err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not start service front-end"))
			os.Exit(1)
		}
	},
}
