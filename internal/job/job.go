package job

import (
	"strings"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	uuid "github.com/satori/go.uuid"
)

// Job - parameters of a job
type Job struct {
	ID           uuid.UUID
	Repo         string
	Payload      string
	Path         string
	Script       *string  `json:",omitempty"`
	ScriptArgs   *string  `json:",omitempty"`
	RemoteScript *bool    `json:",omitempty"`
	Deps         []string `json:",omitempty"`
}

// CreateJob - create a new job struct with validated field values
func CreateJob(repo string, payload string, path string,
	script *string, scriptArgs *string, remoteScript bool,
	deps string) (*Job, error) {
	id, err := uuid.NewV1()
	if err != nil {
		log.Error.Println("Could not generate UUID:", err)
		return nil, err
	}

	leasePath := path
	if leasePath[0] != '/' {
		leasePath = "/" + leasePath
	}

	job := &Job{
		id,
		repo,
		payload,
		leasePath,
		script,
		scriptArgs,
		&remoteScript,
		strings.Split(deps, ",")}

	return job, nil
}
