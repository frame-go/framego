package ginex

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/frame-go/framego/log"
)

// ContextLoggerMiddleware add logger to Context
func ContextLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := log.Logger.With().
			Str("ip", c.ClientIP()).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Logger()
		log.SetGinContextLogger(c, &logger)
	}
}

func LogRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		logger := log.FromContext(c)

		// Handle the panic case and still write the error log
		defer func() {
			if v := recover(); v != nil {
				logger.Error().
					Str("query", c.Request.URL.RawQuery).
					Int64("request_size", c.Request.ContentLength).
					Dur("latency", time.Since(start)).
					Interface("panic", v).
					Msg("http_request_panic")

				// re-raise error
				panic(v)
			}
		}()

		// Process request
		c.Next()

		// Write request log
		logger.Info().
			Str("query", c.Request.URL.RawQuery).
			Int64("request_size", c.Request.ContentLength).
			Int("status", c.Writer.Status()).
			Int("response_size", c.Writer.Size()).
			Dur("latency", time.Since(start)).
			Msg("http_request")
	}
}
