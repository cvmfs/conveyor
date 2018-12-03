package cvmfs

import (
	"os"

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

// RunConsume - runs the job consumer
func RunConsume(qCfg *QueueConfig, jCfg *JobDbConfig, tempDir string, maxJobRetries int) error {
	keys, err := ReadKeys(jCfg.KeyDir)
	if err != nil {
		return errors.Wrap(err, "could not read API keys from file")
	}

	// Create temporary dir
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return errors.Wrap(err, "could not create temp dir")
	}
	defer os.RemoveAll(tempDir)

	consumer, err := createConsumer(qCfg, keys, jCfg.JobDBURL(), tempDir, maxJobRetries)
	if err != nil {
		return errors.Wrap(err, "could not create RabbitMQ message consumer")
	}
	defer consumer.close()

	LogInfo.Println("Entering consumer loop")

	if err := consumer.loop(); err != nil {
		return errors.Wrap(err, "error in consumer loop")
	}

	return nil
}
