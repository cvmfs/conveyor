package commands

import (
	"os"

	"github.com/cvmfs/conveyor/internal/cvmfs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:     "conveyor",
	Short:   "CernVM-FS Conveyor",
	Long:    "CernVM-FS Conveyor - Higher-level publishing tools for CVMFS repositories",
	Version: "0.9.0",
}

var cfgFile string
var logTimestamps *bool

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"/etc/cvmfs/conveyor/config.toml",
		"config file (TOML or JSON)")
	logTimestamps = rootCmd.PersistentFlags().Bool(
		"log-timestamps",
		false,
		"include timestamps in logging output")
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(submitCmd)
	rootCmd.AddCommand(workerCmd)

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