package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:     "cvmfs_job",
	Short:   "CVMFS publishing tool",
	Long:    "A tool for working with publishing jobs to CVMFS repositories",
	Version: "0.9.0",
}

var cfgFile = "/etc/cvmfs/publisher/config.json"

var logTimestamps *bool

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"/etc/cvmfs/publisher/config.json",
		"config file")
	logTimestamps = rootCmd.PersistentFlags().Bool(
		"log-timestamps",
		false,
		"include timestamps in logging output")
	rootCmd.AddCommand(consumeCmd)
	rootCmd.AddCommand(submitCmd)
	rootCmd.AddCommand(dbCmd)

	viper.BindPFlag("log-timestamps", rootCmd.PersistentFlags().Lookup("log-timestamps"))
}

func initConfig() {
	log.InitLogging(os.Stdout, os.Stderr, *logTimestamps)

	viper.SetDefault("rabbitmq.port", 5672)
	viper.SetDefault("rabbitmq.vhost", "/cvmfs")
	viper.SetDefault("jobdb.port", 8080)
	viper.SetDefault("jobdb.backend.port", 5432)

	viper.SetConfigFile(cfgFile)

	if err := viper.ReadInConfig(); err != nil {
		log.Error.Println("Could not read config:", err)
		os.Exit(1)
	}
}

// Execute the root command of the application
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error.Println(err)
		os.Exit(1)
	}
}
