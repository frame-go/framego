package grpcex

// TODO: replace implemenation by go-grpc-middleware v2

import (
	"context"
	"time"

	"github.com/google/uuid"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpczerolog "github.com/philip-bui/grpc-zerolog"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	ferrors "github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/log"
)

func SetZeroLogger() {
	grpclog.SetLoggerV2(grpczerolog.NewGrpcZeroLogger(*log.Logger))
}

func ContextLoggerUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		ctxLogger := getContextLogger(ctx, info.FullMethod)
		ctx = log.SetContextLogger(ctx, ctxLogger)
		resp, err := handler(ctx, req)
		return resp, err
	}
}

func ContextLoggerStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		ctxLogger := getContextLogger(ctx, info.FullMethod)
		ctx = log.SetContextLogger(ctx, ctxLogger)
		wrappedStream := grpc_middleware.WrapServerStream(stream)
		wrappedStream.WrappedContext = ctx
		return handler(srv, wrappedStream)
	}
}

func LogRequestUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		logger := log.FromContext(ctx)
		startTime := time.Now()

		defer logPanic(ctx, startTime)

		resp, err := handler(ctx, req)
		latency := time.Since(startTime)
		if err == nil {
			logger.Info().Dur("latency", latency).Msg("grpc_request")
		} else {
			ferrors.LogError(logger.Warn(), err).Dur("latency", latency).Msg("grpc_request_with_error")
		}
		return resp, err
	}
}

func LogRequestStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		logger := log.FromContext(ctx)
		startTime := time.Now()

		defer logPanic(ctx, startTime)

		logger.Info().Msg("grpc_stream_start")
		err := handler(srv, stream)
		latency := time.Since(startTime)
		if err == nil {
			logger.Info().Dur("latency", latency).Msg("grpc_stream_end")
		} else {
			ferrors.LogError(logger.Warn(), err).Dur("latency", latency).Msg("grpc_stream_end_with_error")
		}
		return err
	}
}

func getRequestIdFromContext(ctx context.Context) string {
	// TODO: use request ID in context injected by open tracing middleware
	return uuid.New().String()
}

func getContextLogger(ctx context.Context, method string) *zerolog.Logger {
	l := log.Logger.With().
		Str("request_id", getRequestIdFromContext(ctx)).
		Str("method", method).
		Str("ip", GetClientIP(ctx)).
		Logger()
	return &l
}

// logPanic handles the panic case and still write the error log
func logPanic(ctx context.Context, startTime time.Time) {
	logger := log.FromContext(ctx)
	if v := recover(); v != nil {
		// TODO: find a way to skip 2 frames in stack trace.
		var panicErr error
		rawErr, ok := v.(error)
		if ok {
			panicErr = errors.Wrap(rawErr, "panic")
		} else {
			panicErr = errors.Errorf("panic: %v", v)
		}

		logger.Error().
			Stack().
			Err(panicErr).
			Dur("latency", time.Since(startTime)).
			Interface("panic", v).
			Msg("grpc_request_panic")

		// re-raise error
		panic(v)
	}
}
