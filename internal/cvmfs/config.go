package cvmfs

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// BackendConfig - database backend configuration for the job server DB backend
type BackendConfig struct {
	Type     string
	Database string
	Username string
	Password string
	Host     string
	Port     int
}

// QueueConfig - configuration of message queue
type QueueConfig struct {
	Username string
	Password string
	Host     string
	VHost    string
	Port     int
}

// Config - main configuration object
type Config struct {
	Host    string
	Port    int
	KeyDir  string
	Queue   QueueConfig
	Backend BackendConfig
}

// JobServerURL constructs the URL of the job DB service
func (c *Config) JobServerURL() string {
	var prefix string
	if !strings.HasPrefix(c.Host, "http://") {
		prefix = "http://"
	}
	return fmt.Sprintf("%s%s:%v/jobs", prefix, c.Host, c.Port)
}

// ReadConfig - populate the config object using the global viper object
// and the config file
func ReadConfig() (*Config, error) {
	srv := viper.Sub("server")
	srv.SetDefault("port", 8080)
	srv.SetDefault("keydir", "/etc/cvmfs/keys")
	var cfg Config
	if err := srv.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "could not read server configuration")
	}

	q := viper.Sub("queue")
	q.SetDefault("port", 5672)
	q.SetDefault("vhost", "/cvmfs")
	if err := q.Unmarshal(&cfg.Queue); err != nil {
		return nil, errors.Wrap(err, "could not read queue configuration")
	}

	db := viper.Sub("db")
	db.SetDefault("db.port", 3306)
	if err := db.Unmarshal(&cfg.Backend); err != nil {
		return nil, errors.Wrap(err, "could not read db configuration")
	}

	return &cfg, nil
}
