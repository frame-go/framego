package appmgr

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/fullstorydev/grpchan"
	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/gin-gonic/gin"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	channelzservice "google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/reflection"

	"github.com/frame-go/framego/health"
)

type serviceImpl struct {
	Service

	ctx           context.Context
	name          string
	app           App
	middlewares   *middlewareApplier
	grpcEndpoint  string
	grpcServer    *grpc.Server
	grpcRegistrar *grpchan.HandlerMap
	httpEndpoint  string
	ginEngine     *gin.Engine
	grpcChannel   *inprocgrpc.Channel
	grpcHttpMux   *runtime.ServeMux
	waitGroup     sync.WaitGroup
	healthServer  health.Server
	healthRunner  health.Runner
}

func newService(ctx context.Context, app App, mm *middlewareManager, config *ServiceConfig) (Service, error) {
	s := &serviceImpl{}
	s.ctx = ctx
	s.app = app
	s.name = config.Name
	s.middlewares = mm.Apply(s, config.Middlewares)
	if config.Endpoints.Grpc != "" {
		s.grpcEndpoint = config.Endpoints.Grpc
		grpcSecurityConfig := &config.Security.Grpc
		grpcSecurityConfig.Resolve()
		tlsConfig, err := newTLSConfig(grpcSecurityConfig.Cert, grpcSecurityConfig.Key, grpcSecurityConfig.Ca)
		if err != nil {
			return nil, err
		}
		s.grpcServer, s.grpcChannel = newGrpcServerWithChannel(tlsConfig, s.middlewares)
		s.grpcRegistrar = &grpchan.HandlerMap{}
	}
	if config.Endpoints.Http != "" {
		s.httpEndpoint = config.Endpoints.Http
		s.ginEngine = newGinEngin(s.middlewares)
		if config.Endpoints.Grpc != "" {
			s.grpcHttpMux = newGrpcHttpMux()
			s.ginEngine.NoRoute(func(c *gin.Context) {
				c.Status(http.StatusOK) // NoRoute handlers will be set to NotFound status by default, here reset to OK.
				s.grpcHttpMux.ServeHTTP(c.Writer, c.Request)
			})
		}
	}
	s.healthRunner = health.NewRunner(health.WithName(s.name))
	s.healthServer = health.NewServer(s.healthRunner)
	return s, nil
}

func (s *serviceImpl) GetContext() context.Context {
	return s.ctx
}

func (s *serviceImpl) GetName() string {
	return s.name
}

func (s *serviceImpl) GetApp() App {
	return s.app
}

func (s *serviceImpl) GetGrpcEndpoint() string {
	return s.grpcEndpoint
}

func (s *serviceImpl) GetHttpEndpoint() string {
	return s.httpEndpoint
}

func (s *serviceImpl) GetGrpcServiceRegistrar() grpc.ServiceRegistrar {
	return s
}

func (s *serviceImpl) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	if s.grpcRegistrar == nil {
		return
	}
	healthChecker, ok := impl.(health.Checker)
	if ok {
		s.healthRunner.AddCheck(desc.ServiceName, healthChecker.HealthCheck)
	}
	s.grpcRegistrar.RegisterService(desc, impl)
}

func (s *serviceImpl) GetGrpcChannelClient() grpc.ClientConnInterface {
	return s.grpcChannel
}

func (s *serviceImpl) GetServeMux() *runtime.ServeMux {
	return s.grpcHttpMux
}

func (s *serviceImpl) GetGinRouter() gin.IRoutes {
	return s.ginEngine
}

func (s *serviceImpl) AddHealthCheck(name string, check health.CheckFunc) {
	s.healthRunner.AddCheck(name, check)
}

func (s *serviceImpl) Run() (err error) {
	status := s.healthRunner.Start()
	if status.State != health.StateHealthy {
		return fmt.Errorf("init_health_check_failed(state=%v,error=%v,source=%s)",
			status.State, status.Error, status.Source)
	}

	if s.grpcServer != nil {
		health.RegisterServer(s.grpcRegistrar, s.healthServer)
		channelzservice.RegisterChannelzServiceToServer(s.grpcRegistrar)
		reflection.Register(s.grpcRegistrar)
		s.grpcRegistrar.ForEach(s.grpcServer.RegisterService)
		if s.grpcChannel != nil {
			s.grpcRegistrar.ForEach(s.grpcChannel.RegisterService)
		}
		if s.middlewares.IsMiddlewareEnabled("metrics") {
			// need to initialize metrics by service info in grpc server only.
			grpc_prometheus.Register(s.grpcServer)
		}
		s.waitGroup.Add(1)
		err = serveGrpcService(s.ctx, s.name, s.grpcServer, s.grpcEndpoint, &s.waitGroup)
		if err != nil {
			return
		}
	}

	if s.ginEngine != nil {
		if s.grpcServer != nil {
			_ = health.RegisterHandlerClient(s.ctx, s.grpcHttpMux, s.grpcChannel)
		}
		s.waitGroup.Add(1)
		err = serveHttpService(s.ctx, s.name, s.ginEngine, s.httpEndpoint, &s.waitGroup)
		if err != nil {
			return
		}
	}
	return
}

func (s *serviceImpl) Wait() {
	s.waitGroup.Wait()
}
