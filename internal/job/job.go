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
	TransferScript bool
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
func CreateJob(params *Parameters) (*Unprocessed, error) {
	id, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate UUID")
	}

	leasePath := params.RepositoryPath
	if leasePath[0] != '/' {
		leasePath = "/" + leasePath
	}

	job := &Unprocessed{ID: id, Parameters: *params}

	if params.Script != "" {
		if params.TransferScript {
			s, err := PackScript(params.Script)
			if err != nil {
				return nil, errors.Wrap(err, "could not load script")
			}
			job.Script = s
		}
	}

	return job, nil
}

// PackScript - packs a script into a gzipped, base64 encoded buffer
func PackScript(script string) (string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	data, err := ioutil.ReadFile(script)
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

// UnpackScript - unpacks a script from a gzipped, base64 encoded buffer
//                and saves it to disk at `dest`
func UnpackScript(body string, dest string) error {
	buf, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return errors.Wrap(err, "base64 decoding failed")
	}
	rd := bytes.NewReader(buf)
	gz, err := gzip.NewReader(rd)
	if err != nil {
		return errors.Wrap(err, "gzip reader construction failed")
	}
	rawbuf, err := ioutil.ReadAll(gz)
	if err != nil {
		return errors.Wrap(err, "decompression failed")
	}
	if err := ioutil.WriteFile(dest, rawbuf, 0755); err != nil {
		return errors.Wrap(err, "writing to disk failed")
	}

	return nil
}
