package consume

import (
	"fmt"
	"os"
	"strings"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/auth"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/jobdb"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/queue"
	"github.com/pkg/errors"
)

var mock bool

func init() {
	mock = false
	v := os.Getenv("CVMFS_MOCK_JOB_CONSUMER")
	if v == "true" || v == "yes" || v == "on" {
		mock = true
	}
}

// Run - runs the job consumer
func Run(qCfg *queue.Config, jCfg *jobdb.Config, tempDir string, maxJobRetries int) error {
	keys, err := auth.ReadKeys(jCfg.KeyDir)
	if err != nil {
		return errors.Wrap(err, "could not read API keys from file")
	}

	var prefix string
	if !strings.HasPrefix(jCfg.Host, "http://") {
		prefix = "http://"
	}
	jobDBURL := fmt.Sprintf("%s%s:%v/jobs", prefix, jCfg.Host, jCfg.Port)

	// Create temporary dir
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return errors.Wrap(err, "could not create temp dir")
	}
	defer os.RemoveAll(tempDir)

	consumer, err := createConsumer(qCfg, keys, jobDBURL, tempDir, maxJobRetries)
	if err != nil {
		return errors.Wrap(err, "could not create RabbitMQ message consumer")
	}
	defer consumer.close()

	log.Info.Println("Waiting for jobs")

	if err := consumer.loop(); err != nil {
		return errors.Wrap(err, "error in consumer loop")
	}

	return nil
}
