package cvmfs

import (
	"io"
	"log"
)

var (
	// LogInfo is the logging sink for normal messages
	LogInfo *log.Logger
	// LogError is the logging sink reserved for error messages
	LogError *log.Logger
)

// InitLogging initializes the Info and Error loggers
func InitLogging(infoHandle io.Writer, errorHandle io.Writer, logTimestamps bool) {
	flags := 0
	if logTimestamps {
		flags = log.Ldate | log.Ltime
	}

	LogInfo = log.New(infoHandle, "INFO: ", flags)
	LogError = log.New(errorHandle, "ERROR: ", flags)
}
