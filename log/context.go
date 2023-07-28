package log

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type IContextGetter interface {
	Value(key interface{}) interface{}
}

const contextLoggerKey = "_logger"

func FromContext(ctx IContextGetter) *zerolog.Logger {
	v := ctx.Value(contextLoggerKey)
	if v == nil {
		return Logger
	}
	ctxLogger, ok := v.(*zerolog.Logger)
	if !ok || ctxLogger == nil {
		return Logger
	}
	return ctxLogger
}

func SetContextLogger(ctx context.Context, logger *zerolog.Logger) context.Context {
	return context.WithValue(ctx, contextLoggerKey, logger)
}

func SetGinContextLogger(ctx *gin.Context, logger *zerolog.Logger) {
	ctx.Set(contextLoggerKey, logger)
}
