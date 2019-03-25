package cvmfs

import (
	"io"

	"github.com/rs/zerolog"
)

// Log is the application-wide logger
var Log zerolog.Logger

// InitLogging initializes the logger
func InitLogging(sink io.Writer) {
	Log = zerolog.New(sink)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// ConfigLogging updates the logging settings with
// values from a Config object (Meant to be called after ReadConfig)
func ConfigLogging(cfg *Config) {
	if cfg.LogTimestamps {
		Log = Log.With().Timestamp().Logger()
	}
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}
