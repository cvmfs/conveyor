package transaction

import (
	"os/exec"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
)

// Run - run the CVMFS transaction according to the job description
func Run(desc job.Description, task func() error, mock bool) error {
	fullPath := desc.Repo
	if desc.Path != "/" {
		fullPath += desc.Path
	}

	ok := true

	log.Info.Println("Running CVMFS transaction for job:", desc.ID.String())

	if err := startTransaction(fullPath, mock); err != nil {
		log.Error.Println("Error starting CVMFS transaction:", err)
		return err
	}

	defer func() {
		if ok {
			log.Info.Println("Publishing CVMFS transaction for job:", desc.ID.String())
			if err := commitTransaction(desc.Repo, mock); err != nil {
				log.Error.Println("Error committing CVMFS transaction:", err)
			}
		} else {
			log.Error.Println("Aborting CVMFS transaction for job:", desc.ID.String())
			if err := abortTransaction(desc.Repo, mock); err != nil {
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

func startTransaction(path string, mock bool) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "transaction", path)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func commitTransaction(repo string, mock bool) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "publish", repo)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func abortTransaction(repo string, mock bool) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "abort", "-f", repo)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
