package log

import (
	"io"
	"log"
	"os"
)

var (
	// Info is the logging level for normal messages
	Info *log.Logger
	// Error is reserved for error messages
	Error *log.Logger
)

// InitLogging initializes the Info and Error loggers
func InitLogging(
	infoHandle io.Writer,
	errorHandle io.Writer) {

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime)
}

func init() {
	InitLogging(os.Stdout, os.Stderr)
}
