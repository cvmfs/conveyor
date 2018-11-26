package jobdb

import (
	"fmt"
	"strings"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/auth"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Config - configuration for the job  db service
type Config struct {
	Host    string
	Port    int
	KeyDir  string
	Backend BackendConfig
}

// ReadConfig - populate the Config object using the global viper object
//              and the config file
func ReadConfig() (*Config, error) {
	v := viper.Sub("jobdb")
	v.SetDefault("port", 8080)
	v.SetDefault("keydir", "/etc/cvmfs/keys")
	v.SetDefault("backend.port", 3306)
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "could not read job db configuration")
	}
	return &cfg, nil
}

// JobDBURL constructs the URL of the job DB service
func (c *Config) JobDBURL() string {
	var prefix string
	if !strings.HasPrefix(c.Host, "http://") {
		prefix = "http://"
	}
	return fmt.Sprintf("%s%s:%v/jobs", prefix, c.Host, c.Port)
}

// Run - run the job db service
func Run(cfg *Config) error {
	log.Info.Println("CVMFS job database service starting")

	keys, err := auth.ReadKeys(cfg.KeyDir)
	if err != nil {
		return errors.Wrap(err, "could not read API key from file")
	}

	backend, err := startBackEnd(cfg.Backend)
	if err != nil {
		return errors.Wrap(err, "could not start service back-end")
	}
	defer backend.Close()

	if err := startFrontEnd(cfg.Port, backend, keys); err != nil {
		return errors.Wrap(err, "could not start service front-end")
	}

	return nil
}
