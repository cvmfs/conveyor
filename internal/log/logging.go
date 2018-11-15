package log

import (
	"io"
	"log"
)

var (
	// Info is the logging level for normal messages
	Info *log.Logger
	// Error is reserved for error messages
	Error *log.Logger
)

// InitLogging initializes the Info and Error loggers
func InitLogging(infoHandle io.Writer, errorHandle io.Writer, logTimestamps bool) {
	flags := 0
	if logTimestamps {
		flags = log.Ldate | log.Ltime
	}

	Info = log.New(infoHandle, "INFO: ", flags)
	Error = log.New(errorHandle, "ERROR: ", flags)
}
