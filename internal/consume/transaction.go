package consume

import (
	"os"
	"os/exec"
	"path"

	"github.com/cvmfs/cvmfs-publisher-tools/internal/job"
	"github.com/cvmfs/cvmfs-publisher-tools/internal/log"
	"github.com/pkg/errors"
)

// RunTransaction - run the CVMFS transaction according to the job description
func RunTransaction(desc job.Unprocessed, task func() error) error {
	fullPath := path.Join(desc.Repository, desc.RepositoryPath)

	// Close any existing transactions
	abortTransaction(desc.Repository, false)

	ok := true

	log.Info.Println("Opening CVMFS transaction for:", fullPath)

	if err := startTransaction(fullPath, true); err != nil {
		return errors.Wrap(err, "could not start CVMFS transaction")
	}

	defer func() {
		if ok {
			log.Info.Println("Publishing CVMFS transaction")
			if err := commitTransaction(desc.Repository, true); err != nil {
				log.Error.Println(
					errors.Wrap(err, "could not commit CVMFS transaction"))
			}
		} else {
			log.Error.Println("Aborting CVMFS transaction")
			if err := abortTransaction(desc.Repository, true); err != nil {
				log.Error.Println(
					errors.Wrap(err, "could not abort CVMFS transaction"))
			}
		}
	}()

	if !mock {
		if err := task(); err != nil {
			ok = false
			return errors.Wrap(err, "coult not run task during transaction")
		}
	}

	return nil
}

func startTransaction(path string, verbose bool) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "transaction", "-r", path)
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func commitTransaction(repo string, verbose bool) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "publish", repo)
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func abortTransaction(repo string, verbose bool) error {
	if !mock {
		cmd := exec.Command("cvmfs_server", "abort", "-f", repo)
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
