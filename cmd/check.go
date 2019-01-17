package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
)

type checkCmdVars struct {
	ids      *[]string
	repo     string
	wait     *bool
	extended *bool
}

var chkvs checkCmdVars

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "check job status",
	Long:  "check the status of a submitted job",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := cvmfs.ReadConfig()
		if err != nil {
			cvmfs.LogError.Println(err)
			os.Exit(1)
		}

		keys, err := cvmfs.ReadKeys(cfg.KeyDir)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not read API keys from file"))
			os.Exit(1)
		}

		client, err := cvmfs.NewJobClient(cfg, keys)
		if err != nil {
			cvmfs.LogError.Println("could not start job client")
			os.Exit(1)
		}

		// Optionally wait for completion of the jobs
		if *chkvs.wait {
			_, err := client.WaitForJobs(*chkvs.ids, chkvs.repo)
			if err != nil {
				cvmfs.LogError.Println(
					errors.Wrap(err, "waiting for job completion failed"))
				os.Exit(1)
			}
		}

		quit := make(chan struct{})
		stats, err := client.GetJobStatus(*chkvs.ids, chkvs.repo, *chkvs.extended, quit)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "error checking job status"))
			os.Exit(1)
		}

		if stats.Status != "ok" {
			fmt.Printf("{\"Status\": \"error\", \"Reason\": \"%v\"}\n", stats.Reason)
			os.Exit(1)
		}

		cvmfs.LogInfo.Println("Completed jobs:")
		if *chkvs.extended {
			for _, j := range stats.Jobs {
				printStatus(j.ID, j)
			}
		} else {
			for _, j := range stats.IDs {
				printStatus(j.ID, j)
			}
		}
	},
}

func printStatus(id uuid.UUID, st interface{}) {
	buf, err := json.Marshal(&st)
	if err != nil {
		cvmfs.LogError.Printf(
			"could not serialize status of job %v to JSON", id)
	} else {
		fmt.Println(string(buf))
	}
}

func init() {
	chkvs.ids = checkCmd.Flags().StringSlice(
		"ids", []string{}, "comma-separate list of job UUIDs to query")
	checkCmd.MarkFlagRequired("ids")
	checkCmd.Flags().StringVar(&chkvs.repo, "repo", "", "target CVMFS repository of the jobs ")
	checkCmd.MarkFlagRequired("repo")
	chkvs.wait = checkCmd.Flags().Bool("wait", false, "wait for completion of the queried jobs")
	chkvs.extended = checkCmd.Flags().Bool("extended-status", false, "return the extended status of the job")
}
