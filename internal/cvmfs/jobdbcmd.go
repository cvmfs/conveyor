package cvmfs

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// JobDbConfig - configuration for the job  db service
type JobDbConfig struct {
	Host    string
	Port    int
	KeyDir  string
	Backend BackendConfig
}

// ReadJobDbConfig - populate the Config object using the global viper object
//              and the config file
func ReadJobDbConfig() (*JobDbConfig, error) {
	v := viper.Sub("jobdb")
	v.SetDefault("port", 8080)
	v.SetDefault("keydir", "/etc/cvmfs/keys")
	v.SetDefault("backend.port", 3306)
	var cfg JobDbConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "could not read job db configuration")
	}
	return &cfg, nil
}

// JobDBURL constructs the URL of the job DB service
func (c *JobDbConfig) JobDBURL() string {
	var prefix string
	if !strings.HasPrefix(c.Host, "http://") {
		prefix = "http://"
	}
	return fmt.Sprintf("%s%s:%v/jobs", prefix, c.Host, c.Port)
}

// RunJobDb - run the job db service
func RunJobDb(cfg *JobDbConfig) error {
	LogInfo.Println("CVMFS job database service starting")

	keys, err := ReadKeys(cfg.KeyDir)
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
