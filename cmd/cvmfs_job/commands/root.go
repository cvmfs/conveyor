package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/cvmfs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:     "cvmfs_job",
	Short:   "CVMFS publishing tool",
	Long:    "A publishing tool for CVMFS repositories",
	Version: "0.9.0",
}

var cfgFile string
var logTimestamps *bool

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"/etc/cvmfs/publisher/config.toml",
		"config file (TOML or JSON)")
	logTimestamps = rootCmd.PersistentFlags().Bool(
		"log-timestamps",
		false,
		"include timestamps in logging output")
	rootCmd.AddCommand(consumerCmd)
	rootCmd.AddCommand(submitCmd)
	rootCmd.AddCommand(serverCmd)

	viper.BindPFlag("log-timestamps", rootCmd.PersistentFlags().Lookup("log-timestamps"))
}

func initConfig() {
	cvmfs.InitLogging(os.Stdout, os.Stderr, *logTimestamps)

	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		cvmfs.LogError.Println(errors.Wrap(err, "could not read config"))
		os.Exit(1)
	}
}

// Execute the root command of the application
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		cvmfs.LogError.Println(err)
		os.Exit(1)
	}
}
