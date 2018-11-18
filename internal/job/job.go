package job

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Parameters - job submission parameters
type Parameters struct {
	Repo         string
	Payload      string
	Path         string
	Script       string
	ScriptArgs   string
	RemoteScript bool
	Deps         []string
}

// Description - parameters of a job
type Description struct {
	ID uuid.UUID
	Parameters
}

// CreateJob - create a new job struct with validated field values
func CreateJob(params Parameters) (*Description, error) {
	id, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate UUID")
	}

	leasePath := params.Path
	if leasePath[0] != '/' {
		leasePath = "/" + leasePath
	}

	job := &Description{id, params}

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
