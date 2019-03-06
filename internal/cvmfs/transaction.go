package cvmfs

import (
	"os"
	"os/exec"
	"path"

	"github.com/pkg/errors"
)

// runTransaction runs a CVMFS transaction on the specified repository, locking the
// provided subpath. The body of the transaction is encoded in the "task" function
func runTransaction(repository, subpath string, task func() error) error {
	fullPath := path.Join(repository, subpath)

	// Close any existing transactions
	abortTransaction(repository, false)

	Log.Infoln("Opening CVMFS transaction for:", fullPath)

	abort := false
	defer func() {
		if abort {
			Log.Errorln("Aborting CVMFS transaction")
			if err := abortTransaction(repository, true); err != nil {
				Log.Errorln(
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

	Log.Infoln("Publishing CVMFS transaction")
	if err := commitTransaction(repository, true); err != nil {
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
