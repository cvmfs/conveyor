package jobdb

import (
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
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
func Run(cfg Config) {
	log.Info.Println("CVMFS job database service starting")

	startFrontEnd(cfg.Port)
}
