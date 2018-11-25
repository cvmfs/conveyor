package job

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	getter "github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Specification - job submission Specification
type Specification struct {
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
	Specification
}

// Processed - a processed job
type Processed struct {
	Unprocessed
	StartTime    time.Time
	FinishTime   time.Time
	Successful   bool
	ErrorMessage string
}

// Status - a pair of job ID and the completion status
type Status struct {
	ID         uuid.UUID
	Successful bool
}

// GetJobReply - Return type of the GetJob query
type GetJobReply struct {
	Status string      // "ok" || "error"
	Reason string      `json:",omitempty"`
	IDs    []Status    `json:",omitempty"`
	Jobs   []Processed `json:",omitempty"`
}

// PutJobReply - Return type of the PutJob query
type PutJobReply struct {
	Status string // "ok" || "error"
	Reason string `json:",omitempty"`
}

// CreateJob - create a new job struct with validated field values
func CreateJob(params *Specification) (*Unprocessed, error) {
	id, err := uuid.NewV1()
	if err != nil {
		return nil, errors.Wrap(err, "could not generate UUID")
	}

	leasePath := params.RepositoryPath
	if leasePath[0] != '/' {
		leasePath = "/" + leasePath
	}

	job := &Unprocessed{ID: id, Specification: *params}

	if params.Script != "" {
		if params.TransferScript {
			s, err := packScript(params.Script)
			if err != nil {
				return nil, errors.Wrap(err, "could not load script")
			}
			job.Script = s
		}
	}

	return job, nil
}

// Process - process a job (download and unpack payload, run script etc.)
func (j *Unprocessed) Process(tempDir string) error {
	// Create target dir if needed
	targetDir := path.Join(
		"/cvmfs", j.Repository, j.RepositoryPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return errors.Wrap(err, "could not create target dir")
	}

	// Download and unpack the payload, if given
	log.Info.Println("Downloading payload:", j.Payload)
	if err := getter.Get(targetDir, j.Payload); err != nil {
		return errors.Wrap(err, "could not download payload")
	}

	// Run the transaction script, if specified
	if j.Script != "" {
		needsUnpacking := j.TransferScript
		log.Info.Printf(
			"Running transaction script: %v (needs unpacking: %v)\n",
			j.Script, needsUnpacking)

		var scriptFile string
		if needsUnpacking {
			var err error
			scriptFile = path.Join(tempDir, "transaction.sh")
			err = unpackScript(j.Script, scriptFile)
			if err != nil {
				return errors.Wrap(err, "unpacking transaction script failed")
			}
		} else {
			scriptFile = j.Script
		}

		err := runScript(
			scriptFile, j.Repository, j.RepositoryPath, j.ScriptArgs)
		if err != nil {
			return errors.Wrap(err, "running transaction script failed")
		}
	}

	return nil
}

// packScript - packs a script into a gzipped, base64 encoded buffer
func packScript(script string) (string, error) {
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

// unpackScript - unpacks a script from a gzipped, base64 encoded buffer
//                and saves it to disk at `dest`
func unpackScript(body string, dest string) error {
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

func runScript(script string, repo string, repoPath string, args string) error {
	cmd := exec.Command(script, repo, repoPath, args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path.Join("/cvmfs", repo)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
