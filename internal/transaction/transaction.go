package transaction

import (
	"os"
	"os/exec"
	"path"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
)

var mock bool

func init() {
	mock = false
	v := os.Getenv("CVMFS_MOCKED_JOB_CONSUMER")
	if v == "true" || v == "yes" || v == "on" {
		mock = true
	}
}

// Run - run the CVMFS transaction according to the job description
func Run(desc job.Description, task func() error) error {
	fullPath := path.Join(desc.Repo, desc.Path)

	ok := true

	log.Info.Println("Opening CVMFS transaction for:", fullPath)

	if err := startTransaction(fullPath); err != nil {
		log.Error.Println("Error starting CVMFS transaction:", err)
		return err
	}

	defer func() {
		if ok {
			log.Info.Println("Publishing CVMFS transaction")
			if err := commitTransaction(desc.Repo); err != nil {
				log.Error.Println("Error committing CVMFS transaction:", err)
			}
		} else {
			log.Error.Println("Aborting CVMFS transaction")
			if err := abortTransaction(desc.Repo); err != nil {
				log.Error.Println("Error aborting CVMFS transaction:", err)
			}
		}
	}()

	if err := task(); err != nil {
		log.Error.Println("Error running task during transaction:", err)
		ok = false
		return err
	}

	return nil
}

func startTransaction(path string) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "transaction", path)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func commitTransaction(repo string) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "publish", repo)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func abortTransaction(repo string) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "abort", "-f", repo)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
