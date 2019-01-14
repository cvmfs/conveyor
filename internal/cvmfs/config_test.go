package cvmfs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

const fullConfig = `
# Job server configuration is used by conveyor {submit, consumer, server}
[server]
host = "job.service.host.name"
port = 1111
keydir = "/test/key/dir" # Default key dir

# Queue configuration is used by conveyor server
[queue]
username = "quser"
password = "qpass"
host = "queue.host.name"
port = 2222
vhost = "/cvmfs"

# Job server backend configuration is only used by conveyor server
[db]
type = "mysql"
database = "testdb"
username = "dbuser"
password = "dbpass"
host = "db.host.name"
post = 3333
`

const partialConfig = `
# Job server configuration is used by conveyor {submit, consumer, server}
[server]
host = "job.service.host.name"
port = 1111
keydir = "/test/key/dir" # Default key dir
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

func TestReadFullConfig(t *testing.T) {
	v, err := PrepareViperHelper(t, fullConfig)
	if err != nil {
		t.Errorf(err.Error())
	}
	cfg, err := readConfigFromViper(v)
	if err != nil {
		t.Errorf("Could not read config from Viper object")
	}

	if cfg.Host != "job.service.host.name" {
		t.Errorf("Invalid hostname in config object")
	}
	if cfg.Port != 1111 {
		t.Errorf("Invalid port in config object")
	}
	if cfg.KeyDir != "/test/key/dir" {
		t.Errorf("Invalid key dir in config object")
	}
}

func TestReadPartialConfig(t *testing.T) {
	v, err := PrepareViperHelper(t, partialConfig)
	if err != nil {
		t.Errorf(err.Error())
	}
	if _, err := readConfigFromViper(v); err != nil {
		t.Errorf("Could not read config from Viper object")
	}
}

func TestReadIncompleteConfig(t *testing.T) {
	v, err := PrepareViperHelper(t, incompleteConfig)
	if err != nil {
		t.Errorf(err.Error())
	}
	if _, err := readConfigFromViper(v); err == nil {
		t.Errorf("Reading an incomplete configuration should result in an error")
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
