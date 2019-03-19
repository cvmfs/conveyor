package commands

import (
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:     "conveyor",
	Short:   "CernVM-FS Conveyor",
	Long:    "CernVM-FS Conveyor - Higher-level publishing tools for CVMFS repositories",
	Version: "0.1.0",
}

var cfgFile string
var logTimestamps *bool
var jobWaitTimeout int

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(
		&cfgFile,
		"config",
		"c",
		"/etc/cvmfs/conveyor/config.toml",
		"config file (TOML or JSON)")
	logTimestamps = rootCmd.PersistentFlags().BoolP(
		"log-timestamps",
		"s",
		false,
		"include timestamps in logging output")
	rootCmd.PersistentFlags().IntVarP(
		&jobWaitTimeout,
		"job-wait-timeout",
		"t",
		7200,
		"timeout (in seconds) when waiting for jobs")
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(submitCmd)
	rootCmd.AddCommand(workerCmd)

	viper.BindPFlag("log-timestamps", rootCmd.PersistentFlags().Lookup("log-timestamps"))
}

func initConfig() {
	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		cvmfs.Log.Error().Err(err).Msg("could not read config")
		os.Exit(1)
	}
}

// Execute the root command of the application
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		cvmfs.Log.Error().Err(err).Msg("could not run main command")
		os.Exit(1)
	}
}
