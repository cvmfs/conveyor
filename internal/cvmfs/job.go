package cvmfs

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"

	getter "github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// MaxJobDuration is the number of seconds that a job is allowed to take
const MaxJobDuration = 2 * 3600

// JobSpecification contains all the parameters of a new job which is to be submitted
type JobSpecification struct {
	Repository     string
	Payload        string
	RepositoryPath string
	Script         string
	ScriptArgs     string
	TransferScript bool
	Dependencies   []string
}

// UnprocessedJob describes a job which has been submitted, having been assigned
// a unique ID
type UnprocessedJob struct {
	ID uuid.UUID
	JobSpecification
}

// ProcessedJob describes a completed job. Additional fields with respect to an
// unprocessed job are related to the execution time of the job and its completion status
type ProcessedJob struct {
	UnprocessedJob
	StartTime    time.Time
	FinishTime   time.Time
	Successful   bool
	ErrorMessage string
}

// JobStatus holds a job ID and its completion status
type JobStatus struct {
	ID         uuid.UUID
	Successful bool
}

// BasicReply is a status message and optional error cause
type BasicReply struct {
	Status string // "ok" || "error"
	Reason string `json:",omitempty"`
}

// GetJobStatusReply is the return type of the GetJob query
type GetJobStatusReply struct {
	BasicReply
	IDs  []JobStatus    `json:",omitempty"`
	Jobs []ProcessedJob `json:",omitempty"`
}

// PostNewJobReply is the return type of the PostNewJob action
type PostNewJobReply struct {
	BasicReply
	ID uuid.UUID
}

// PostJobStatusReply is the return value of the PutJobStatus action
type PostJobStatusReply struct {
	BasicReply
}

// Prepare a job specification for submission: normalizes the repository path and embeds
// the transaction script in the job description, if the script is a local file
func (spec *JobSpecification) Prepare() error {
	if spec.RepositoryPath[0] != '/' {
		spec.RepositoryPath = "/" + spec.RepositoryPath
	}

	if spec.Script != "" {
		if spec.TransferScript {
			s, err := packScript(spec.Script)
			if err != nil {
				return errors.Wrap(err, "could not load script")
			}
			spec.Script = s
		}
	}

	return nil
}

// Process a job (download and unpack payload, run script etc.)
func (j *UnprocessedJob) process(tempDir string) error {
	// Create target dir if needed
	targetDir := path.Join(
		"/cvmfs", j.Repository, j.RepositoryPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return errors.Wrap(err, "could not create target dir")
	}

	// Download and unpack the payload, if given
	if j.Payload != "" {
		LogInfo.Println("Downloading payload:", j.Payload)
		if err := getter.Get(targetDir, j.Payload); err != nil {
			return errors.Wrap(err, "could not download payload")
		}
	}

	// Run the transaction script, if specified
	if j.Script != "" {
		needsUnpacking := j.TransferScript
		LogInfo.Printf(
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

// packScript into a gzipped, base64 encoded buffer
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

// unpackScript from a gzipped, base64 encoded buffer and saves it to disk at `dest`
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
