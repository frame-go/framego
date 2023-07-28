package appmgr

import (
	"context"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/linxGnu/mssqlx"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"github.com/frame-go/framego/client/cache"
	"github.com/frame-go/framego/health"
)

type AppInfo struct {
	Name        string
	Version     string
	Description string
}

type App interface {
	// Init initializes application by config
	// Any error will cause process exit with error status
	Init()

	// Run runs application
	Run() error

	// RunOrExit runs application and exits with error status if got any error
	RunOrExit()

	// RegisterMiddleware registers a middleware with name in application
	// The middlewares need to be registered before calling App.Init
	RegisterMiddleware(string, Middleware)

	// AddJob adds job func with name into application
	AddJob(string, func(ctx context.Context) error)

	// GetContext gets context of application
	GetContext() context.Context

	// GetService gets service object by name
	GetService(string) Service

	// GetGrpcClientConn gets grpc client conn interface by name
	GetGrpcClientConn(string) grpc.ClientConnInterface

	// GetGrpcClientConns gets grpc client conn interfaces by name for different endpoints
	GetGrpcClientConns(string) []grpc.ClientConnInterface

	// GetDatabaseClient gets database gorm client by name
	GetDatabaseClient(string) *gorm.DB

	// GetDatabaseSqlxClient gets database mssqlx client by name
	GetDatabaseSqlxClient(string) *mssqlx.DBs

	// GetCacheClient gets cache client by name
	GetCacheClient(string) cache.Client

	// GetPulsarClient gets Pulsar client by name
	GetPulsarClient(string) pulsar.Client
}

type Service interface {
	GetContext() context.Context
	GetName() string
	GetApp() App
	GetGrpcEndpoint() string
	GetHttpEndpoint() string
	GetGrpcServiceRegistrar() grpc.ServiceRegistrar
	GetGrpcChannelClient() grpc.ClientConnInterface
	GetServeMux() *runtime.ServeMux
	GetGinRouter() gin.IRoutes
	AddHealthCheck(name string, check health.CheckFunc)
	Run() error
	Wait()
}

type ObservableService interface {
	GetContext() context.Context
	Run() error
	Wait()
}

type ClientManager interface {
	GetGrpcClientConn(string) grpc.ClientConnInterface
	GetGrpcClientConns(string) []grpc.ClientConnInterface
}

type DatabaseManager interface {
	GetDatabaseClient(string) *gorm.DB
	GetDatabaseSqlxClient(string) *mssqlx.DBs
}

type Middleware interface {
	GinHandler(map[string]interface{}) gin.HandlerFunc
	GrpcServerInterceptor(map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor)
	GrpcClientInterceptor(map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor)
}
