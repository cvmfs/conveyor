package job

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Parameters - job submission parameters
type Parameters struct {
	Repository     string
	Payload        string
	RepositoryPath string
	Script         string
	ScriptArgs     string
	RemoteScript   bool
	Dependencies   []string
}

// Unprocessed - a job submission that has been assigned and ID
type Unprocessed struct {
	ID uuid.UUID
	Parameters
}

// Processed - a processed job
type Processed struct {
	Unprocessed
	StartTime    time.Time
	FinishTime   time.Time
	Successful   bool
	ErrorMessage string
}

// CreateJob - create a new job struct with validated field values
func CreateJob(params Parameters) (*Unprocessed, error) {
	id, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate UUID")
	}

	leasePath := params.RepositoryPath
	if leasePath[0] != '/' {
		leasePath = "/" + leasePath
	}

	job := &Unprocessed{ID: id, Parameters: params}

	if params.Script != "" {
		if !params.RemoteScript {
			s, err := loadScript(params.Script)
			if err != nil {
				return nil, errors.Wrap(err, "could not load script")
			}
			job.Script = s
		}
	}

	return job, nil
}

func loadScript(s string) (string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	data, err := ioutil.ReadFile(s)
	if err != nil {
		return "", errors.Wrap(err, "could not read script file")
	}
	if _, err := gz.Write(data); err != nil {
		return "", errors.Wrap(err, "could not compress script")
	}
	if err := gz.Close(); err != nil {
		return "", errors.Wrap(err, "could not close gzip compressor")
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
