package jobdb

import (
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/pkg/errors"
)

// Config - configuration for the job  db service
type Config struct {
	Host    string
	Port    int
	Backend BackendConfig
}

// Run - run the job db service
func Run(cfg Config) error {
	log.Info.Println("CVMFS job database service starting")

	backend, err := startBackEnd(cfg.Backend)
	if err != nil {
		return errors.Wrap(err, "could not start service back-end")
	}
	defer backend.Close()

	if err := startFrontEnd(cfg.Port, backend); err != nil {
		return errors.Wrap(err, "could not start service front-end")
	}

	return nil
}
