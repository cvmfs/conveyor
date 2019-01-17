package commands

import (
	"fmt"
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var jobName string
var repo string
var payload string
var path string
var script string
var scriptArgs string
var transferScript *bool
var deps *[]string
var wait *bool

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job",
	Long:  "Submit a publishing job to a queue",
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

		spec := &cvmfs.JobSpecification{
			JobName: jobName, Repository: repo, Payload: payload, RepositoryPath: path,
			Script: script, ScriptArgs: scriptArgs, TransferScript: *transferScript,
			Dependencies: *deps}

		if err := spec.Prepare(); err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not create job object"))
			os.Exit(1)
		}

		client, err := cvmfs.NewJobClient(cfg, keys)
		if err != nil {
			cvmfs.LogError.Println("could not start job client")
			os.Exit(1)
		}

		stat, err := client.PostNewJob(spec)
		if err != nil {
			cvmfs.LogError.Println(
				errors.Wrap(err, "could not post new job"))
			os.Exit(1)
		}

		if stat.Status != "ok" {
			fmt.Printf("{\"Status\": \"error\", \"Reason\": \"%v\"}\n", stat.Reason)
			os.Exit(1)
		}

		id := stat.ID

		// Optionally wait for completion of the job
		if *wait {
			stats, err := client.WaitForJobs([]string{id.String()}, spec.Repository)
			if err != nil {
				cvmfs.LogError.Println(
					errors.Wrap(err, "waiting for job completion failed"))
				os.Exit(1)
			}

			if !stats[0].Successful {
				fmt.Printf("{\"Status\": \"error\", \"ID\": \"%s\"}\n", id)
				os.Exit(1)
			}
		}

		fmt.Printf("{\"Status\": \"ok\", \"ID\": \"%s\"}\n", id)
	},
}

func init() {
	submitCmd.Flags().StringVar(&jobName, "job-name", "", "name of the job")
	submitCmd.Flags().StringVar(&repo, "repo", "", "target CVMFS repository")
	submitCmd.MarkFlagRequired("repo")
	submitCmd.Flags().StringVar(&payload, "payload", "", "payload URL")
	submitCmd.Flags().StringVar(&path, "path", "/", "target path inside the repository")
	submitCmd.Flags().StringVar(
		&script, "script", "", "script to run at the end of CVMFS transaction")
	submitCmd.Flags().StringVar(
		&scriptArgs, "script-args", "", "arguments of the transaction script")
	transferScript = submitCmd.Flags().Bool(
		"transfer-script", false, "transaction script is a local file which should be sent")
	deps = submitCmd.Flags().StringSlice(
		"deps", []string{}, "comma-separate list of job dependency UUIDs")
	wait = submitCmd.Flags().Bool("wait", false, "wait for completion of the submitted job")
}
