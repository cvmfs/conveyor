package commands

import (
	"errors"
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/spf13/cobra"
)

type submitCmdVars struct {
	jobName   string
	repo      string
	payload   string
	leasePath string
	deps      *[]string
	wait      *bool
}

var subvs submitCmdVars

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job",
	Long:  "Submit a publishing job to a queue",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cvmfs.InitLogging(os.Stdout, *logTimestamps)

		cfg, err := cvmfs.ReadConfig(cvmfs.ClientProfile)
		if err != nil {
			cvmfs.Log.Error().Err(err).Msg("config error")
			os.Exit(1)
		}
		if rootCmd.PersistentFlags().Changed("timeout") {
			cfg.JobWaitTimeout = jobWaitTimeout
		}

		spec := &cvmfs.JobSpecification{
			JobName: subvs.jobName, Repository: subvs.repo, Payload: subvs.payload,
			LeasePath: subvs.leasePath, Dependencies: *subvs.deps}

		spec.Prepare()

		client, err := cvmfs.NewJobClient(cfg)
		if err != nil {
			cvmfs.Log.Error().Err(err).Msg("could not start job client")
			os.Exit(1)
		}

		stat, err := client.PostNewJob(spec)
		if err != nil {
			cvmfs.Log.Error().Err(err).Msg("could not post new job")
			os.Exit(1)
		}

		if stat.Status != "ok" {
			cvmfs.Log.Error().
				Err(errors.New(stat.Reason)).
				Msg("job failed")
			os.Exit(1)
		}

		id := stat.ID

		cvmfs.Log.Info().Str("job_id", id.String()).Msg("job submitted successfully")

		// Optionally wait for completion of the job
		if *subvs.wait {
			stats, err := client.WaitForJobs([]string{id.String()}, jobWaitTimeout)
			if err != nil {
				cvmfs.Log.Error().
					Err(err).
					Msg("waiting for job completion failed")
				os.Exit(1)
			}

			if stats[0].Successful {
				cvmfs.Log.Info().
					Str("job_id", id.String()).
					Bool("success", stats[0].Successful).
					Msg("job finished")
			} else {
				quit := make(chan struct{})
				st, err := client.GetJobStatus([]string{id.String()}, true, quit)
				if err != nil {
					cvmfs.Log.Error().Err(err).Msg("job status check failed")
					os.Exit(1)
				}
				job := st.Jobs[0]
				cvmfs.Log.Error().
					Str("job_id", id.String()).
					Bool("success", job.Successful).
					Str("error", job.ErrorMessage).
					Msg("job finished")
				os.Exit(1)
			}
		}
	},
}

func init() {
	submitCmd.Flags().StringVarP(&subvs.jobName, "job-name", "j", "", "name of the job")
	submitCmd.Flags().StringVarP(&subvs.repo, "repo", "r", "", "target CVMFS repository")
	submitCmd.MarkFlagRequired("repo")
	submitCmd.Flags().StringVarP(&subvs.payload, "payload", "p", "", "payload URL")
	submitCmd.Flags().StringVarP(&subvs.leasePath, "lease-path", "l", "/", "leased path inside the repository")
	subvs.deps = submitCmd.Flags().StringSliceP(
		"deps", "d", []string{}, "comma-separate list of job dependency UUIDs")
	subvs.wait = submitCmd.Flags().BoolP("wait", "w", false, "wait for completion of the submitted job")
}
