package cvmfs

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	// ClientProfile is used by the submit and check commands
	ClientProfile = iota
	// ServerProfile is used by the server command
	ServerProfile
	// WorkerProfile is used by the worker command
	WorkerProfile
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
	Username             string
	Password             string
	Host                 string
	VHost                string
	Port                 int
	NewJobExchange       string `mapstructure:"new_job_exchange"`
	NewJobQueue          string `mapstructure:"new_job_queue"`
	CompletedJobExchange string `mapstructure:"completed_job_exchange"`
}

// WorkerConfig - configuration of the Conveyor worker daemon
type WorkerConfig struct {
	Name       string
	JobRetries int    `mapstructure:"job_retries"`
	TempDir    string `mapstructure:"temp_dir"`
}

// ServerConfig - configuration of the Conveyor jov server
type ServerConfig struct {
	Host string
	Port int
}

// Config - main configuration object
type Config struct {
	SharedKey      string `mapstructure:"shared_key"`
	JobWaitTimeout int    `mapstructure:"job_wait_timeout"`
	Server         ServerConfig
	Queue          QueueConfig
	Backend        BackendConfig
	Worker         WorkerConfig
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
	return newHTTPEndpoints(c.Server.Host, c.Server.Port)
}

// ReadConfig - populate the config object using the global viper object
// and the config file, based on profile (client, worker, or server).
// The different sections may not needed in all profiles
func ReadConfig(profile int) (*Config, error) {
	return readConfigFromViper(viper.GetViper(), profile)
}

func readConfigFromViper(v *viper.Viper, profile int) (*Config, error) {
	// Create new config object with default values
	cfg, err := newConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not create default configuration")
	}

	// Read configuration file (Viper)
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "could not read server configuration")
	}

	srv := v.Sub("server")
	if srv == nil {
		return nil, fmt.Errorf("Could not read config; missing server section")
	}
	if err := srv.Unmarshal(&cfg.Server); err != nil {
		return nil, errors.Wrap(err, "could not read server configuration")
	}

	q := v.Sub("queue")
	if q != nil {
		if err := q.Unmarshal(&cfg.Queue); err != nil {
			return nil, errors.Wrap(err, "could not read queue configuration")
		}
	}

	if profile == ServerProfile {
		db := v.Sub("db")
		if db != nil {
			if err := db.Unmarshal(&cfg.Backend); err != nil {
				return nil, errors.Wrap(err, "could not read db configuration")
			}
		}
	}

	if profile == WorkerProfile {
		worker := v.Sub("worker")
		if worker != nil {
			if err := worker.Unmarshal(&cfg.Worker); err != nil {
				return nil, errors.Wrap(err, "could not read worker configuration")
			}
		}
	}

	// Apply overrides from environment variables (for credentials)
	overrideWithEnvVars(cfg, profile)

	// Check that all mandatory parameters are set
	if err := validateConfig(cfg, profile); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	return cfg, nil
}

func defaultName() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", errors.Wrap(err, "could not retrieve hostname")
	}

	return name, nil
}

func newConfig() (*Config, error) {
	var cfg Config

	cfg.SharedKey = "UNSET"
	cfg.JobWaitTimeout = 7200

	cfg.Server.Port = 8080

	cfg.Queue.Port = 5672
	cfg.Queue.VHost = "/cvmfs/"
	cfg.Queue.NewJobExchange = "jobs.new"
	cfg.Queue.NewJobQueue = "jobs.new"
	cfg.Queue.CompletedJobExchange = "jobs.done"

	cfg.Backend.Port = 5432

	// worker name defaults to the hostname
	name, err := defaultName()
	if err != nil {
		return nil, err
	}
	cfg.Worker.Name = name

	// default temporary dir used for handling job artifacts
	cfg.Worker.TempDir = "/tmp/conveyor-worker"

	// maximum number of retries for processing a job before giving up
	// and recording it as a failed job
	cfg.Worker.JobRetries = 3

	return &cfg, nil
}

func overrideWithEnvVars(cfg *Config, profile int) {
	setFromEnvVar(&cfg.SharedKey, "CONVEYOR_SHARED_KEY")
	setFromEnvVar(&cfg.Queue.Username, "CONVEYOR_QUEUE_USER")
	setFromEnvVar(&cfg.Queue.Password, "CONVEYOR_QUEUE_PASS")
	if profile == ServerProfile {
		setFromEnvVar(&cfg.Backend.Username, "CONVEYOR_DB_USER")
		setFromEnvVar(&cfg.Backend.Password, "CONVEYOR_DB_PASS")
	}
}

func validateConfig(cfg *Config, profile int) error {
	if isUnset(cfg.SharedKey) {
		return errors.New("shared API key is unset")
	}

	if isUnset(cfg.Queue.Username) {
		return errors.New("RabbitMQ username is unset")
	}
	if isUnset(cfg.Queue.Password) {
		return errors.New("RabbitMQ password is unset")
	}
	if isUnset(cfg.Queue.Host) {
		return errors.New("RabbitMQ hostname is unset")
	}

	if profile == ServerProfile {
		if isUnset(cfg.Backend.Type) {
			return errors.New("Database type is unset")
		}
		if isUnset(cfg.Backend.Database) {
			return errors.New("Database name is unset")
		}
		if isUnset(cfg.Backend.Username) {
			return errors.New("Database username is unset")
		}
		if isUnset(cfg.Backend.Password) {
			return errors.New("Database password is unset")
		}
		if isUnset(cfg.Backend.Host) {
			return errors.New("Database hostname is unset")
		}
	}

	return nil
}

func isUnset(v string) bool {
	return (v == "UNSET" || v == "")
}

// Set the value of p from the environment variable if it is set.
// If envVar is empty, the value of p is unchanged
func setFromEnvVar(p *string, envVar string) {
	t := os.Getenv(envVar)
	if t != "" {
		*p = t
	}
}
