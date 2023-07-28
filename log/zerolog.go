package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var Logger *zerolog.Logger

// Init initializes the zerolog global logger with default config
func Init(level string, debug bool, beauty bool) {
	logLevel := zerolog.InfoLevel
	if debug {
		logLevel = zerolog.DebugLevel
	}
	if level != "" {
		parsedLevel, err := zerolog.ParseLevel(level)
		if err == nil {
			logLevel = parsedLevel
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "[Init Logger Error] Unknown Log Level (%s): %v\n", level, err)
		}
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldUnit = time.Nanosecond
	zerolog.DurationFieldInteger = true
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	var output io.Writer
	if beauty {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339Nano}
	} else {
		output = os.Stdout
	}
	zlog.Logger = zerolog.New(output).With().Timestamp().Logger()
	Logger = &zlog.Logger
}
