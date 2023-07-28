package database

import (
	"context"
	"errors"
	"gorm.io/gorm/utils"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type logger struct {
	SlowThreshold         time.Duration
	SourceField           string
	SkipErrRecordNotFound bool
	Logger                *zerolog.Logger
}

func NewGormLogger(l *zerolog.Logger) *logger {
	return &logger{
		Logger:                l,
		SkipErrRecordNotFound: true,
	}
}

func (l *logger) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *logger) Info(ctx context.Context, s string, args ...interface{}) {
	l.Logger.Info().Msgf(s, args...)
}

func (l *logger) Warn(ctx context.Context, s string, args ...interface{}) {
	l.Logger.Warn().Msgf(s, args...)
}

func (l *logger) Error(ctx context.Context, s string, args ...interface{}) {
	l.Logger.Error().Msgf(s, args...)
}

func (l *logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	var logger *zerolog.Event
	var msg string
	if err != nil && !(errors.Is(err, gorm.ErrRecordNotFound) && l.SkipErrRecordNotFound) {
		logger = l.Logger.Error().Err(err)
		msg = "[GORM] query error"
		return
	} else if l.SlowThreshold != 0 && elapsed > l.SlowThreshold {
		logger = l.Logger.Warn()
		msg = "[GORM] slow query"
	} else {
		logger = l.Logger.Debug()
		msg = "[GORM] query"
	}
	sql, rows := fc()
	logger.Str("sql", sql)
	if rows >= 0 {
		logger.Int64("rows", rows)
	}
	logger.Dur("duration", elapsed)
	if l.SourceField != "" {
		logger.Str(l.SourceField, utils.FileWithLineNum())
	}
	logger.Msg(msg)
}
