package job

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
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
		log.Error.Println("Could not generate UUID:", err)
		return nil, err
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
				log.Error.Println("Could not load script:", err)
				return nil, err
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
		log.Error.Println("Could not read script file:", err)
		return "", err
	}
	if _, err := gz.Write(data); err != nil {
		log.Error.Println("Could not compress script:", err)
		return "", err
	}
	if err := gz.Close(); err != nil {
		log.Error.Println("Could not close gzip compressor:", err)
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
