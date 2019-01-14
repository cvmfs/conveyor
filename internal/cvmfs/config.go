package cvmfs

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// BackendConfig - database backend configuration for the conveyor job server DB backend
type BackendConfig struct {
	Type     string
	Database string
	Username string
	Password string
	Host     string
	Port     int
}

// QueueConfig - configuration of message queue (RabbitMQ)
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

// HTTPEndpoints holds the different HTTP end points of the conveyor job server
type HTTPEndpoints struct {
	base string
}

// NewHTTPEndpoints creates a new HTTPEndpoints object using a hostname and a port.
// Prepends "http://" to the hostname if neither "http://"" nor "https://" are given
func newHTTPEndpoints(host string, port int) HTTPEndpoints {
	var prefix string
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		prefix = "http://"
	}
	base := fmt.Sprintf("%s%s:%v", prefix, host, port)
	return HTTPEndpoints{base}
}

// NewJobs returns the endpoint for new jobs. If "withBase" is true, the base URL
// is prepended
func (o HTTPEndpoints) NewJobs(withBase bool) string {
	pt := "/jobs/new"
	if withBase {
		return o.base + pt
	}
	return pt
}

// CompletedJobs returns the endpoint for completed jobs.  If "withBase" is true, the
// base URL is prepended
func (o HTTPEndpoints) CompletedJobs(withBase bool) string {
	pt := "/jobs/complete"
	if withBase {
		return o.base + pt
	}
	return pt
}

// HTTPEndpoints constructs an HTTPEndpoints object
func (c *Config) HTTPEndpoints() HTTPEndpoints {
	return newHTTPEndpoints(c.Host, c.Port)
}

// ReadConfig - populate the config object using the global viper object
// and the config file
func ReadConfig() (*Config, error) {
	return readConfigFromViper(viper.GetViper())
}

func readConfigFromViper(v *viper.Viper) (*Config, error) {
	srv := v.Sub("server")
	if srv == nil {
		return nil, fmt.Errorf("Could not read config; missing server section")
	}
	srv.SetDefault("port", 8080)
	srv.SetDefault("keydir", "/etc/cvmfs/keys")
	var cfg Config
	if err := srv.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "could not read server configuration")
	}

	q := v.Sub("queue")
	if q != nil {
		q.SetDefault("port", 5672)
		q.SetDefault("vhost", "/cvmfs")
		if err := q.Unmarshal(&cfg.Queue); err != nil {
			return nil, errors.Wrap(err, "could not read queue configuration")
		}
	}

	db := v.Sub("db")
	if db != nil {
		db.SetDefault("port", 3306)
		if err := db.Unmarshal(&cfg.Backend); err != nil {
			return nil, errors.Wrap(err, "could not read db configuration")
		}
	}

	return &cfg, nil
}
