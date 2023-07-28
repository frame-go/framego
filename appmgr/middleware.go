package appmgr

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/frame-go/framego/config"
	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/ginex"
	"github.com/frame-go/framego/grpcex"
	"github.com/frame-go/framego/log"
)

type middlewareManager struct {
	middlewares map[string]Middleware
}

func newMiddlewareManager() *middlewareManager {
	return &middlewareManager{
		middlewares: make(map[string]Middleware),
	}
}

func newDefaultMiddlewareManager() *middlewareManager {
	m := newMiddlewareManager()
	m.RegisterMiddleware("context_logger", NewContextLoggerMiddleware())
	m.RegisterMiddleware("log_request", NewLogRequestMiddleware())
	m.RegisterMiddleware("metrics", NewMetricsMiddleware())
	m.RegisterMiddleware("open_tracing", NewOpenTracingMiddleware())
	m.RegisterMiddleware("recovery", NewRecoveryMiddleware())
	m.RegisterMiddleware("request_validation", NewRequestValidationMiddleware())
	m.RegisterMiddleware("cors", NewCorsMiddleware())
	m.RegisterMiddleware("compress", NewCompressMiddleware())
	m.RegisterMiddleware("access_control", NewAccessControlMiddleware())
	return m
}

func (m *middlewareManager) RegisterMiddleware(name string, middleware Middleware) {
	m.middlewares[strings.ToLower(name)] = middleware
}

func (m *middlewareManager) GetMiddleware(name string) Middleware {
	return m.middlewares[strings.ToLower(name)]
}

func (m *middlewareManager) Apply(service Service, configs []interface{}) *middlewareApplier {
	ma := newMiddlewareApplier()
	ma.AddMiddleware("", NewContextTagsMiddleware(), nil)
	if service != nil {
		ma.AddMiddleware("", NewServiceContextMiddleware(service), nil)
	}
	for i, middlewareConfig := range configs {
		err := ma.AddMiddlewareByConfig(m, middlewareConfig)
		if err != nil {
			errors.LogError(log.Logger.Error(), err).Str("service", service.GetName()).
				Int("index", i).Msg("create_middleware_error")
		}
	}
	return ma
}

type middlewareConstructor struct {
	name    string
	m       Middleware
	options map[string]interface{}
}

type middlewareApplier struct {
	mcs []*middlewareConstructor
}

func newMiddlewareApplier() *middlewareApplier {
	a := &middlewareApplier{
		mcs: make([]*middlewareConstructor, 0),
	}
	return a
}

func (a *middlewareApplier) AddMiddleware(name string, middleware Middleware, options map[string]interface{}) {
	mc := &middlewareConstructor{name: name, m: middleware, options: options}
	a.mcs = append(a.mcs, mc)
}

func (a *middlewareApplier) AddMiddlewareByConfig(mm *middlewareManager, config interface{}) error {
	name, options, err := parseMiddlewareConfig(config)
	if err != nil {
		return err
	}
	middleware := mm.GetMiddleware(name)
	if middleware == nil {
		return errors.New("unknown_middleware_name").With("middleware", name)
	}
	a.AddMiddleware(strings.ToLower(name), middleware, options)
	return nil
}

func (a *middlewareApplier) IsMiddlewareEnabled(name string) bool {
	if name == "" {
		return false
	}
	name = strings.ToLower(name)
	for _, mc := range a.mcs {
		if mc.name == name {
			return true
		}
	}
	return false
}

func (a *middlewareApplier) ApplyGin(routes gin.IRoutes) {
	for _, mc := range a.mcs {
		if mc.m != nil {
			handler := mc.m.GinHandler(mc.options)
			if handler != nil {
				routes.Use(handler)
			}
		}
	}
}

func (a *middlewareApplier) GrpcServerInterceptor() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	unaryServerInterceptors := make([]grpc.UnaryServerInterceptor, 0)
	streamServerInterceptors := make([]grpc.StreamServerInterceptor, 0)
	for _, mc := range a.mcs {
		if mc.m != nil {
			unaryInterceptor, streamInterceptor := mc.m.GrpcServerInterceptor(mc.options)
			if unaryInterceptor != nil {
				unaryServerInterceptors = append(unaryServerInterceptors, unaryInterceptor)
			}
			if streamInterceptor != nil {
				streamServerInterceptors = append(streamServerInterceptors, streamInterceptor)
			}
		}
	}
	chainUnaryServerInterceptor := grpc_middleware.ChainUnaryServer(unaryServerInterceptors...)
	chainStreamServerInterceptor := grpc_middleware.ChainStreamServer(streamServerInterceptors...)
	return chainUnaryServerInterceptor, chainStreamServerInterceptor
}

func (a *middlewareApplier) GrpcClientInterceptor() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	unaryClientInterceptors := make([]grpc.UnaryClientInterceptor, 0)
	streamClientInterceptors := make([]grpc.StreamClientInterceptor, 0)
	for _, mc := range a.mcs {
		if mc.m != nil {
			unaryInterceptor, streamInterceptor := mc.m.GrpcClientInterceptor(mc.options)
			if unaryInterceptor != nil {
				unaryClientInterceptors = append(unaryClientInterceptors, unaryInterceptor)
			}
			if streamInterceptor != nil {
				streamClientInterceptors = append(streamClientInterceptors, streamInterceptor)
			}
		}
	}
	chainUnaryClientInterceptor := grpc_middleware.ChainUnaryClient(unaryClientInterceptors...)
	chainStreamClientInterceptor := grpc_middleware.ChainStreamClient(streamClientInterceptors...)
	return chainUnaryClientInterceptor, chainStreamClientInterceptor
}

type contextTagsMiddleware struct {
	Middleware
}

func NewContextTagsMiddleware() Middleware {
	return &contextTagsMiddleware{}
}

func (m *contextTagsMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return nil
}

func (m *contextTagsMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return grpc_ctxtags.UnaryServerInterceptor(), grpc_ctxtags.StreamServerInterceptor()
}

func (m *contextTagsMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}

type serviceContextMiddleware struct {
	Middleware
	service Service
}

func NewServiceContextMiddleware(service Service) Middleware {
	return &serviceContextMiddleware{service: service}
}

func (m *serviceContextMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		SetGinContextService(c, m.service)
	}
}

func (m *serviceContextMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return serviceContextUnaryServerInterceptor(m.service), serviceContextStreamServerInterceptor(m.service)
}

func serviceContextUnaryServerInterceptor(service Service) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		ctx = SetContextService(ctx, service)
		resp, err := handler(ctx, req)
		return resp, err
	}
}

func serviceContextStreamServerInterceptor(service Service) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		ctx = SetContextService(ctx, service)
		wrappedStream := grpc_middleware.WrapServerStream(stream)
		wrappedStream.WrappedContext = ctx
		return handler(srv, wrappedStream)
	}
}

func (m *serviceContextMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}

type contextLoggerMiddleware struct {
	Middleware
}

func NewContextLoggerMiddleware() Middleware {
	return &contextLoggerMiddleware{}
}

func (m *contextLoggerMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return ginex.ContextLoggerMiddleware()
}

func (m *contextLoggerMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return grpcex.ContextLoggerUnaryServerInterceptor(), grpcex.ContextLoggerStreamServerInterceptor()
}

func (m *contextLoggerMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}

type logRequestMiddleware struct {
	Middleware
}

func NewLogRequestMiddleware() Middleware {
	return &logRequestMiddleware{}
}

func (m *logRequestMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return ginex.LogRequestMiddleware()
}

func (m *logRequestMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return grpcex.LogRequestUnaryServerInterceptor(), grpcex.LogRequestStreamServerInterceptor()
}

func (m *logRequestMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}

type metricsMiddleware struct {
	Middleware
}

func NewMetricsMiddleware() Middleware {
	grpc_prometheus.EnableHandlingTimeHistogram()
	return &metricsMiddleware{}
}

func (m *metricsMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return getGinPrometheus().HandlerFunc()
}

func (m *metricsMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return grpc_prometheus.UnaryServerInterceptor, grpc_prometheus.StreamServerInterceptor
}

func (m *metricsMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return grpc_prometheus.UnaryClientInterceptor, grpc_prometheus.StreamClientInterceptor
}

type openTracingMiddleware struct {
	Middleware
}

func NewOpenTracingMiddleware() Middleware {
	return &openTracingMiddleware{}
}

func (m *openTracingMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	// TODO: implementation
	// https://github.com/opentracing-contrib/go-gin
	// https://github.com/Bose/go-gin-opentracing
	return nil
}

func (m *openTracingMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// TODO: implementation
	// Integrate wiht https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/tracing/opentracing
	return nil, nil
}

func (m *openTracingMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	// TODO: implementation
	// Integrate wiht https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/tracing/opentracing
	return nil, nil
}

type recoveryMiddleware struct {
	Middleware
}

func NewRecoveryMiddleware() Middleware {
	return &recoveryMiddleware{}
}

func (m *recoveryMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return gin.Recovery()
}

func (m *recoveryMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandlerContext(func(ctx context.Context, p interface{}) (err error) {
			var panicErr error
			rawErr, ok := p.(error)
			if ok {
				panicErr = errors.Wrap(rawErr, "panic").WithGRPCCode(codes.Internal)
			} else {
				panicErr = errors.New("panic").With("cause", p).WithGRPCCode(codes.Internal)
			}
			log.Logger.Error().Stack().Err(panicErr).Msg("grpc_handler_panic_recovery")
			return panicErr
		}),
	}
	return grpc_recovery.UnaryServerInterceptor(opts...), grpc_recovery.StreamServerInterceptor(opts...)
}

func (m *recoveryMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}

type requestValidationMiddleware struct {
	Middleware
}

func NewRequestValidationMiddleware() Middleware {
	return &requestValidationMiddleware{}
}

func (m *requestValidationMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return nil
}

func (m *requestValidationMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return grpcex.ValidatorUnaryServerInterceptor(), grpcex.ValidatorStreamServerInterceptor()
}

func (m *requestValidationMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return grpcex.ValidatorUnaryClientInterceptor(), nil
}

type corsMiddleware struct {
	Middleware
}

func NewCorsMiddleware() Middleware {
	return &corsMiddleware{}
}

func (m *corsMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return ginex.CorsMiddleware(options)
}

func (m *corsMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return nil, nil
}

func (m *corsMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}

type compressMiddleware struct {
	Middleware
}

func NewCompressMiddleware() Middleware {
	return &compressMiddleware{}
}

func (m *compressMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return ginex.CompressMiddleware(options)
}

func (m *compressMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return nil, nil
}

func (m *compressMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}

type accessControlMiddleware struct {
	Middleware
}

func NewAccessControlMiddleware() Middleware {
	return &accessControlMiddleware{}
}

func (m *accessControlMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return nil
}

func (m *accessControlMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	c := config.StringMap(options)
	model, _ := c.GetString("model")
	if model != "" {
		model = resolvePathInConfig(model)
	}
	policy, _ := c.GetString("policy")
	if policy == "" {
		log.Logger.Fatal().Interface("options", options).Msg("grpc_access_middleware_without_policy")
	} else {
		policy = resolvePathInConfig(policy)
	}
	accessController, err := grpcex.NewAccessController(model, policy)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("grpc_access_middleware_init_failed")
	}
	return accessController.UnaryServerInterceptor(), accessController.StreamServerInterceptor()
}

func (m *accessControlMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}
