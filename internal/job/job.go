package job

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"strings"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	uuid "github.com/satori/go.uuid"
)

// Description - parameters of a job
type Description struct {
	ID           uuid.UUID
	Repo         string
	Payload      string
	Path         string
	Script       string   `json:",omitempty"`
	ScriptArgs   string   `json:",omitempty"`
	RemoteScript *bool    `json:",omitempty"`
	Deps         []string `json:",omitempty"`
}

// CreateJob - create a new job struct with validated field values
func CreateJob(repo string, payload string, path string,
	script string, scriptArgs string, remoteScript bool,
	deps string) (*Description, error) {
	id, err := uuid.NewV1()
	if err != nil {
		log.Error.Println("Could not generate UUID:", err)
		return nil, err
	}

	leasePath := path
	if leasePath[0] != '/' {
		leasePath = "/" + leasePath
	}

	dependencies := []string{}
	if deps != "" {
		dependencies = strings.Split(deps, ",")
	}
	job := &Description{
		id,
		repo,
		payload,
		leasePath,
		"", "", nil, dependencies}

	if script != "" {
		job.RemoteScript = &remoteScript
		job.ScriptArgs = scriptArgs
		if remoteScript {
			job.Script = script
		} else {
			s, err := loadScript(script)
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
