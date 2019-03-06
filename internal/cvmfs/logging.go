package cvmfs

import (
	"io"

	"github.com/Sirupsen/logrus"
)

// Log is the application-wide logger
var Log *logrus.Logger

// InitLogging initializes the logger
func InitLogging(hd io.Writer, logTimestamps bool) {
	Log = logrus.New()
	formatter := new(logrus.JSONFormatter)
	formatter.DisableTimestamp = !logTimestamps
	Log.SetFormatter(formatter)
	Log.SetOutput(hd)
}
