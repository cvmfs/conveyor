package jobdb

import (
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/pkg/errors"
)

// BackendConfig - database backend configuration for the job db service
type BackendConfig struct {
	Type     string
	Username string
	Password string
	Host     string
	Port     int
}

// Config - configuration for the job  db service
type Config struct {
	Host    string
	Port    int
	Backend BackendConfig
}

// Run - run the job db service
func Run(cfg Config) error {
	log.Info.Println("CVMFS job database service starting")

	if err := startFrontEnd(cfg.Port); err != nil {
		return errors.Wrap(err, "could not start service front-end")
	}

	return nil
}
