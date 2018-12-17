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

// HTTPEndpoints holds the different HTTP end points of the job server
type HTTPEndpoints struct {
	base   string
	basews string
}

// NewJobs returns the endpoint for new jobs. Set withBase = true to prepend the base URL
func (o HTTPEndpoints) NewJobs(withBase bool) string {
	pt := "/jobs/new"
	if withBase {
		return o.base + pt
	}
	return pt
}

// CompletedJobs returns the endpoint for completed jobs. Set withBase = true to prepend the base URL
func (o HTTPEndpoints) CompletedJobs(withBase bool) string {
	pt := "/jobs/complete"
	if withBase {
		return o.base + pt
	}
	return pt
}

// HTTPEndpoints constructs an HTTPEndpoints object
func (c *Config) HTTPEndpoints() HTTPEndpoints {
	var prefix string
	if !strings.HasPrefix(c.Host, "http://") {
		prefix = "http://"
	}
	base := fmt.Sprintf("%s%s:%v", prefix, c.Host, c.Port)
	basews := "ws:" + strings.TrimPrefix(base, "http:")
	return HTTPEndpoints{base, basews}
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
	if q != nil {
		q.SetDefault("port", 5672)
		q.SetDefault("vhost", "/cvmfs")
		if err := q.Unmarshal(&cfg.Queue); err != nil {
			return nil, errors.Wrap(err, "could not read queue configuration")
		}
	}

	db := viper.Sub("db")
	if db != nil {
		db.SetDefault("port", 3306)
		if err := db.Unmarshal(&cfg.Backend); err != nil {
			return nil, errors.Wrap(err, "could not read db configuration")
		}
	}

	return &cfg, nil
}
