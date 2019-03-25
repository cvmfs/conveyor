package cvmfs

import (
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
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
func (j *UnprocessedJob) process(tempDir string) error {
	if j.Payload != "" {
		// Parse the payload string
		tokens := strings.Split(j.Payload, "|")
		if tokens[0] != "script" || len(tokens) < 2 {
			return errors.New("invalid payload string")
		}
		scriptURL := tokens[1]
		var scriptArg string
		if len(tokens) > 2 {
			scriptArg = tokens[2]
		}

		u, err := url.Parse(scriptURL)
		if err != nil {
			return errors.New("could not parse payload script URL")
		}
		scriptFile := path.Join(tempDir, u.Path)

		// Download the script into the temp directory
		Log.Debug().Str("url", scriptURL).Msg("downloading transaction script")
		if err := downloadFile(tempDir, scriptURL, downloadTimeout); err != nil {
			return errors.Wrap(err, "could not download payload")
		}

		// Make downloaded script file executable
		if err := os.Chmod(scriptFile, 0755); err != nil {
			return errors.Wrap(err, "could not make transaction script executable")
		}

		// Run the script from the root of the repository; the repository name,
		// the lease path, and the optional argument from the payload strin are
		// passed as arguments to the string
		if err := runScript(scriptFile, j.Repository, j.LeasePath, scriptArg); err != nil {
			return errors.Wrap(err, "running transaction script failed")
		}
	}

	return nil
}

func runScript(script string, repo string, leasePath string, arg string) error {
	cmd := exec.Command(script, repo, leasePath, arg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path.Join("/cvmfs", repo)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
