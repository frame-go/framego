package appmgr

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/fullstorydev/grpcui/standalone"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	channelz "github.com/rantav/go-grpc-channelz"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/frame-go/framego/log"
)

type observableImpl struct {
	ObservableService

	ctx          context.Context
	config       *ObservableConfig
	services     map[string]Service
	httpEndpoint string
	ginEngine    *gin.Engine
	waitGroup    sync.WaitGroup
}

type moduleHandler func(*observableImpl)

var moduleMap = map[string]moduleHandler{
	"pprof":    pprofModule,
	"metrics":  metricsModule,
	"swagger":  swaggerModule,
	"channelz": channelzModule,
	"grpcui":   grpcuiModule,
}

func newObservable(ctx context.Context, mm *middlewareManager, config *ObservableConfig, services map[string]Service) ObservableService {
	o := &observableImpl{}
	o.ctx = ctx
	o.config = config
	o.services = services
	o.httpEndpoint = config.Endpoints.Http
	middlewares := mm.Apply(nil, []interface{}{"recovery"})
	o.ginEngine = newGinEngin(middlewares)
	return o
}

func (o *observableImpl) GetContext() context.Context {
	return o.ctx
}

func (o *observableImpl) Run() error {
	for _, moduleName := range o.config.Modules {
		module, ok := moduleMap[strings.ToLower(moduleName)]
		if ok {
			module(o)
		} else {
			log.Logger.Error().Str("module", moduleName).Msg("run_observable_unknown_module_name")
		}
	}

	o.waitGroup.Add(1)
	return serveHttpService(o.ctx, ".observable", o.ginEngine, o.httpEndpoint, &o.waitGroup)
}

func (o *observableImpl) Wait() {
	o.waitGroup.Wait()
}

func pprofModule(o *observableImpl) {
	pprof.Register(o.ginEngine, "/pprof")
}

func metricsModule(o *observableImpl) {
	getGinPrometheus().MetricsPath = "/metrics"
	getGinPrometheus().SetMetricsPath(o.ginEngine)
}

func swaggerModule(o *observableImpl) {
	o.ginEngine.GET("/swagger", func(c *gin.Context) { c.Redirect(301, "/swagger/index.html") })
	o.ginEngine.Any("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func channelzModule(o *observableImpl) {
	for _, service := range o.services {
		grpcEndpiont := service.GetGrpcEndpoint()
		if grpcEndpiont != "" {
			if strings.HasPrefix(grpcEndpiont, ":") {
				grpcEndpiont = "127.0.0.1" + grpcEndpiont
			}
			if strings.HasPrefix(grpcEndpiont, "0.0.0.0:") {
				grpcEndpiont = "127.0.0.1:" + grpcEndpiont[8:]
			}
			channelzHandler := channelz.CreateHandler("/", grpcEndpiont)
			o.ginEngine.Any("/channelz/*any", gin.WrapH(channelzHandler))
			return
		}
	}
}

func grpcuiModule(o *observableImpl) {
	for _, service := range o.services {
		if service.GetGrpcChannelClient() != nil {
			grpcHandler, err := standalone.HandlerViaReflection(context.Background(), service.GetGrpcChannelClient(), service.GetGrpcEndpoint())
			if err == nil {
				o.ginEngine.Any("/grpcui/*any", gin.WrapH(http.StripPrefix("/grpcui", grpcHandler)))
			} else {
				log.Logger.Error().Err(err).Str("service", service.GetName()).Msg("init_observable_grpcui_error")
			}
			return
		}
	}
}
