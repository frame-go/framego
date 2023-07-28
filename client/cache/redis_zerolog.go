package cache

import (
	"context"

	"github.com/rs/zerolog"
)

type Logging interface {
	Printf(ctx context.Context, format string, v ...interface{})
}

// zerologWrapper implements go-redis Logging interface based on underlying zerolog.Logger
type zerologWrapper struct {
	l *zerolog.Logger
}

// NewLoggerWithZerolog creates a new logger which wraps the given zerolog.Logger
func NewLoggerWithZerolog(logger *zerolog.Logger) Logging {
	return &zerologWrapper{
		l: logger,
	}
}

func (w *zerologWrapper) Printf(ctx context.Context, format string, v ...interface{}) {
	w.l.Error().Msgf(format, v...)
}
