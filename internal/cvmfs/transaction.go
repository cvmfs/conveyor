package cvmfs

import (
	"os"
	"os/exec"
	"path"

	"github.com/pkg/errors"
)

// RunTransaction - run the CVMFS transaction according to the job description
func RunTransaction(desc UnprocessedJob, task func() error) error {
	fullPath := path.Join(desc.Repository, desc.RepositoryPath)

	// Close any existing transactions
	abortTransaction(desc.Repository, false)

	LogInfo.Println("Opening CVMFS transaction for:", fullPath)

	abort := false
	defer func() {
		if abort {
			LogError.Println("Aborting CVMFS transaction")
			if err := abortTransaction(desc.Repository, true); err != nil {
				LogError.Println(
					errors.Wrap(err, "could not abort CVMFS transaction"))
			}
		}
	}()

	if err := startTransaction(fullPath, true); err != nil {
		abort = true
		return errors.Wrap(err, "could not start CVMFS transaction")
	}

	if !mock {
		if err := task(); err != nil {
			abort = true
			return errors.Wrap(err, "coult not run task during transaction")
		}
	}

	LogInfo.Println("Publishing CVMFS transaction")
	if err := commitTransaction(desc.Repository, true); err != nil {
		abort = true
		return errors.Wrap(err, "could not commit CVMFS transaction")
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
