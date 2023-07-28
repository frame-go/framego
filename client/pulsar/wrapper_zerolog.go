package pulsar

import (
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/rs/zerolog"
)

// zerologWrapper implements Logger interface based on underlying zerolog.Logger
type zerologWrapper struct {
	l *zerolog.Logger
}

// NewLoggerWithZerolog creates a new logger which wraps the given zerolog.Logger
func NewLoggerWithZerolog(logger *zerolog.Logger) log.Logger {
	return &zerologWrapper{
		l: logger,
	}
}

func (l *zerologWrapper) SubLogger(fs log.Fields) log.Logger {
	c := l.l.With()
	for k, v := range fs {
		c = c.Interface(k, v)
	}
	sl := c.Logger()
	return &zerologWrapper{
		l: &sl,
	}
}

func (l *zerologWrapper) WithFields(fs log.Fields) log.Entry {
	return l.SubLogger(fs)
}

func (l *zerologWrapper) WithField(name string, value interface{}) log.Entry {
	sl := l.l.With().Interface(name, value).Logger()
	return &zerologWrapper{
		l: &sl,
	}
}

func (l *zerologWrapper) WithError(err error) log.Entry {
	sl := l.l.With().Err(err).Logger()
	return &zerologWrapper{
		l: &sl,
	}
}

func (l *zerologWrapper) Debug(args ...interface{}) {
	// Disable debug log
	//l.l.Debug().Msg(fmt.Sprint(args...))
}

func (l *zerologWrapper) Info(args ...interface{}) {
	l.l.Info().Msg(fmt.Sprint(args...))
}

func (l *zerologWrapper) Warn(args ...interface{}) {
	l.l.Warn().Msg(fmt.Sprint(args...))
}

func (l *zerologWrapper) Error(args ...interface{}) {
	l.l.Error().Msg(fmt.Sprint(args...))
}

func (l *zerologWrapper) Debugf(format string, args ...interface{}) {
	// Disable debug log
	//l.l.Debug().Msg(fmt.Sprintf(format, args...))
}

func (l *zerologWrapper) Infof(format string, args ...interface{}) {
	l.l.Info().Msg(fmt.Sprintf(format, args...))
}

func (l *zerologWrapper) Warnf(format string, args ...interface{}) {
	l.l.Warn().Msg(fmt.Sprintf(format, args...))
}

func (l *zerologWrapper) Errorf(format string, args ...interface{}) {
	l.l.Error().Msg(fmt.Sprintf(format, args...))
}
