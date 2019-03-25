package cvmfs

import (
	"io"

	"github.com/rs/zerolog"
)

// Log is the application-wide logger
var Log zerolog.Logger

// InitLogging initializes the logger
func InitLogging(sink io.Writer, logTimestamps, debug bool) {
	l := zerolog.New(sink)
	if logTimestamps {
		Log = l.With().Timestamp().Logger()
	} else {
		Log = l
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}
