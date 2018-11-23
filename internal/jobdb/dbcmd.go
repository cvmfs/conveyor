package jobdb

import (
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/util"
	"github.com/pkg/errors"
)

// Config - configuration for the job  db service
type Config struct {
	Host    string
	Port    int
	Keys    []string
	Backend BackendConfig
}

// Run - run the job db service
func Run(cfg Config) error {
	log.Info.Println("CVMFS job database service starting")

	keys, err := util.ReadKeys(cfg.Keys)
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
