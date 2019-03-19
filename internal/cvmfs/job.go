package cvmfs

import (
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	// JobSuccess signals that a job was successfully processed
	JobSuccess = iota
	// JobRetry signals that a job that failed but should be retried
	JobRetry
	// JobFailure signals that a job failed and should not be retried
	JobFailure
)

// JobSpecification contains all the parameters of a new job which is to be submitted
type JobSpecification struct {
	JobName      string
	Repository   string
	Payload      string
	LeasePath    string
	Dependencies []string
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
	WorkerName   string
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

// Prepare a job specification for submission: normalizes the lease path and embeds
// the transaction script in the job description, if the script is a local file
func (spec *JobSpecification) Prepare() {
	if spec.LeasePath[0] != '/' {
		spec.LeasePath = "/" + spec.LeasePath
	}
}

// Process a job (download and unpack payload, run script etc.)
func (j *UnprocessedJob) process(tempDir string) (int, error) {
	var ret = JobSuccess
	if j.Payload != "" {
		// Parse the payload string
		tokens := strings.Split(j.Payload, "|")
		if tokens[0] != "script" || len(tokens) < 2 {
			return JobFailure, errors.New("invalid payload string")
		}
		scriptURL := tokens[1]
		var scriptArg string
		if len(tokens) > 2 {
			scriptArg = tokens[2]
		}

		u, err := url.Parse(scriptURL)
		if err != nil {
			return JobFailure, errors.New("could not parse payload script URL")
		}
		scriptFile := path.Join(tempDir, u.Path)

		// Download the script into the temp directory
		Log.Info().Str("url", scriptURL).Msg("downloading transaction script")
		if err := downloadFile(tempDir, scriptURL, downloadTimeout); err != nil {
			return JobRetry, errors.Wrap(err, "could not download payload")
		}

		// Make downloaded script file executable
		if err := os.Chmod(scriptFile, 0755); err != nil {
			return JobRetry, errors.Wrap(err, "could not make transaction script executable")
		}

		// Run the script from the root of the repository; the repository name,
		// the lease path, and the optional argument from the payload strin are
		// passed as arguments to the string
		ret, err = runScript(scriptFile, j.Repository, j.LeasePath, scriptArg)
		if err != nil {
			return ret, errors.Wrap(err, "running transaction script failed")
		}
	}

	return ret, nil
}

func runScript(script string, repo string, leasePath string, arg string) (int, error) {
	cmd := exec.Command(script, repo, leasePath, arg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path.Join("/cvmfs", repo)
	if err := cmd.Run(); err != nil {
		if exitCode(err) < 0 {
			return JobRetry, err
		}
		return JobFailure, err
	}

	return JobSuccess, nil
}

func exitCode(err error) int {
	exitCode := 1
	if exitError, ok := err.(*exec.ExitError); ok {
		ws := exitError.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	return exitCode
}
