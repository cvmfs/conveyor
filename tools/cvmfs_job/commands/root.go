package commands

import (
	"os"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:     "cvmfs-job",
	Short:   "CVMFS publishing tool",
	Long:    "A tool for working with publishing jobs to CVMFS repositories",
	Version: "0.9.0",
}

var cfgFile = "/etc/cvmfs/publisher/config.json"

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"/etc/cvmfs/publisher/config.json",
		"config file")
	rootCmd.AddCommand(submitCmd)
}

func initConfig() {
	viper.SetDefault("rabbitmq.port", 5672)
	viper.SetDefault("rabbitmq.vhost", "/cvmfs")
	viper.SetConfigFile(cfgFile)

	if err := viper.ReadInConfig(); err != nil {
		log.Error.Println("Can't read config:", err)
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
