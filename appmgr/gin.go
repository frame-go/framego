package appmgr

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	ginprometheus "github.com/zsais/go-gin-prometheus"

	"github.com/frame-go/framego/log"
)

func regularizeGinMsg(p []byte) string {
	s := string(p)
	if strings.HasSuffix(s, "\n") {
		s = s[:len(s)-1]
	}
	return s
}

type infoLogger struct{}

func (l *infoLogger) Write(p []byte) (n int, err error) {
	log.Logger.Info().Str("text", regularizeGinMsg(p)).Msg("gin_debug")
	return len(p), nil
}

type errorLogger struct{}

func (l *errorLogger) Write(p []byte) (n int, err error) {
	log.Logger.Error().Str("text", regularizeGinMsg(p)).Msg("gin_error")
	return len(p), nil
}

var prometheus *ginprometheus.Prometheus

func getGinPrometheus() *ginprometheus.Prometheus {
	return prometheus
}

func initGin(app string) {
	if viper.GetBool("beautify_log") {
		gin.ForceConsoleColor()
	} else {
		gin.DisableConsoleColor()
	}
	if viper.GetBool("debug") {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DefaultWriter = &infoLogger{}
	gin.DefaultErrorWriter = &errorLogger{}

	prometheus = ginprometheus.NewPrometheus(app)
}

func newGinEngin(middlewares *middlewareApplier) *gin.Engine {
	e := gin.New()
	middlewares.ApplyGin(e)
	return e
}
