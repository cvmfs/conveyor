package cvmfs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

const clientConfig = `
shared_key = "TESTKEY" # Default key dir

# Job server configuration is used by conveyor {submit, consumer, server}
[server]
host = "job.service.host.name"
port = 1111

# Queue configuration is used by conveyor server
[queue]
username = "quser"
password = "qpass"
host = "queue.host.name"
port = 2222
vhost = "/cvmfs"
new_job_exchange = "nje"
new_job_queue = "njq"
completed_job_exchange = "cje"
`

const serverConfig = `
shared_key = "TESTKEY" # Default key dir

# Job server configuration is used by conveyor {submit, consumer, server}
[server]
host = "job.service.host.name"
port = 1111

# Queue configuration is used by conveyor server
[queue]
username = "quser"
password = "qpass"
host = "queue.host.name"
port = 2222
vhost = "/cvmfs"
new_job_exchange = "nje"
new_job_queue = "njq"
completed_job_exchange = "cje"

# Job server backend configuration is only used by conveyor server
[db]
type = "mysql"
database = "testdb"
username = "dbuser"
password = "dbpass"
host = "db.host.name"
post = 3333
`

const workerConfig = `
shared_key = "TESTKEY" # Default key dir

# Job server configuration is used by conveyor {submit, consumer, server}
[server]
host = "job.service.host.name"
port = 1111

# Queue configuration is used by conveyor server
[queue]
username = "quser"
password = "qpass"
host = "queue.host.name"
port = 2222
vhost = "/cvmfs"
new_job_exchange = "nje"
new_job_queue = "njq"
completed_job_exchange = "cje"

# Worker configuration
[worker]
name = "jeff"
job_retries = 11
temp_dir = "/tmp/dir"
`

const partialConfig = `
shared_key = "TESTKEY" # Default key dir

# Job server configuration is used by conveyor {submit, consumer, server}
[server]
host = "job.service.host.name"
port = 1111
`

const incompleteConfig = `
# Queue configuration is used by conveyor server
[queue]
username = "quser"
password = "qpass"
host = "queue.host.name"
port = 2222
vhost = "/cvmfs"
`

func PrepareViperHelper(t *testing.T, cfg string) (*viper.Viper, error) {
	t.Helper()
	rd := strings.NewReader(cfg)
	v := viper.New()
	v.SetConfigType("toml")
	if err := v.ReadConfig(rd); err != nil {
		return nil, fmt.Errorf("Could not read config file body")
	}
	return v, nil
}

func TestReadClientConfig(t *testing.T) {
	v, err := PrepareViperHelper(t, clientConfig)
	if err != nil {
		t.Errorf(err.Error())
	}
	cfg, err := readConfigFromViper(v, nil, ClientProfile)
	if err != nil {
		t.Errorf("Could not read config from Viper object")
	}

	if cfg.SharedKey != "TESTKEY" {
		t.Errorf("Invalid key dir: %v\n", cfg.SharedKey)
	}

	if cfg.Server.Host != "job.service.host.name" {
		t.Errorf("Invalid hostname: %v\n", cfg.Server.Host)
	}
	if cfg.Server.Port != 1111 {
		t.Errorf("Invalid port: %v\n", cfg.Server.Port)
	}
}

func TestReadServerConfig(t *testing.T) {
	v, err := PrepareViperHelper(t, serverConfig)
	if err != nil {
		t.Errorf(err.Error())
	}
	cfg, err := readConfigFromViper(v, nil, ServerProfile)
	if err != nil {
		t.Errorf("Could not read config from Viper object")
	}

	if cfg.Queue.NewJobExchange != "nje" {
		t.Errorf(
			"Invalid name of new job exchange: %v\n",
			cfg.Queue.NewJobExchange)
	}
	if cfg.Queue.NewJobQueue != "njq" {
		t.Errorf(
			"Invalid name of new job queue: %v\n",
			cfg.Queue.NewJobQueue)
	}
	if cfg.Queue.CompletedJobExchange != "cje" {
		t.Errorf(
			"Invalid name of completed job exchange: %v\n",
			cfg.Queue.CompletedJobExchange)
	}
}

func TestReadWorkerConfig(t *testing.T) {
	v, err := PrepareViperHelper(t, workerConfig)
	if err != nil {
		t.Errorf(err.Error())
	}
	cfg, err := readConfigFromViper(v, nil, WorkerProfile)
	if err != nil {
		t.Errorf("Could not read config from Viper object")
	}

	if cfg.Worker.Name != "jeff" {
		t.Errorf("Invalid worker name: %v\n", cfg.Worker.Name)
	}

	if cfg.Worker.TempDir != "/tmp/dir" {
		t.Errorf("Invalid temp dir: %v\n", cfg.Worker.TempDir)
	}

	if cfg.Worker.JobRetries != 11 {
		t.Errorf("Invalid max job retries: %v\n", cfg.Worker.JobRetries)
	}
}

func TestHTTPEndpoints(t *testing.T) {
	host1 := "http://base.host.name1"
	port1 := 111
	epts1 := newHTTPEndpoints(host1, port1)
	if epts1.NewJobs(true) != host1+fmt.Sprintf(":%v", port1)+"/jobs/new" {
		t.Errorf("Invalid new job endpoint: %v\n", epts1.NewJobs(true))
	}

	// "http://" should be prepended
	host2 := "base.host.name2"
	port2 := 222
	epts2 := newHTTPEndpoints(host2, port2)
	if epts2.CompletedJobs(true) != "http://"+host2+fmt.Sprintf(":%v", port2)+"/jobs/complete" {
		t.Errorf("Invalid HTTP endpoint base: %v\n", epts2.base)
	}
}
