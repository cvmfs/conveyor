package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/jobdb"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/submit"
	"github.com/spf13/cobra"
)

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
		qCfg, err := queue.ReadConfig()
		if err != nil {
			log.Error.Println(err)
			os.Exit(1)
		}
		jCfg, err := jobdb.ReadConfig()
		if err != nil {
			log.Error.Println(err)
			os.Exit(1)
		}
		jparams := &job.Specification{
			Repository: repo, Payload: payload, RepositoryPath: path,
			Script: script, ScriptArgs: scriptArgs, TransferScript: *transferScript,
			Dependencies: *deps}
		if err := submit.Run(jparams, qCfg, jCfg, *wait); err != nil {
			log.Error.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
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
