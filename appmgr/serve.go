package appmgr

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/frame-go/framego/log"
)

const shutdownTimeout = 10

func serveHttpService(ctx context.Context, name string, httpHandler http.Handler, endpoint string, wg *sync.WaitGroup) error {
	ctxLogger := log.Logger.With().Str("service", name).Str("type", "http").Str("endpoint", endpoint).Logger()
	ctxLogger.Info().Msg("start_serving_http_server")

	srv := &http.Server{
		Addr:    endpoint,
		Handler: httpHandler,
	}

	lis, err := net.Listen("tcp", endpoint)
	if err != nil {
		ctxLogger.Error().Err(err).Msg("http_server_listen_error")
		return err
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		err := srv.Serve(lis)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			ctxLogger.Error().Err(err).Msg("run_http_server_error")
			panic(err)
		} else {
			ctxLogger.Warn().Msg("closed_http_server")
		}
	}()

	// Wait for stop singel and gracefully exit server
	go func() {
		defer wg.Done()

		<-ctx.Done()
		ctxLogger.Warn().Msg("stopping_http_server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
		defer cancel()

		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			ctxLogger.Error().Err(err).Msg("stopped_http_server_with_error")
		} else {
			ctxLogger.Info().Msg("stopped_serving_http_server")
		}
	}()

	return nil
}

func serveGrpcService(ctx context.Context, name string, grpcServer *grpc.Server, endpoint string, wg *sync.WaitGroup) error {
	ctxLogger := log.Logger.With().Str("service", name).Str("type", "grpc").Str("endpoint", endpoint).Logger()
	ctxLogger.Info().Msg("start_serving_grpc_server")

	ln, err := net.Listen("tcp", endpoint)
	if err != nil {
		ctxLogger.Error().Err(err).Msg("grpc_server_listen_error")
		return err
	}

	go func() {
		defer wg.Done()
		err := grpcServer.Serve(ln)
		if err != nil {
			ctxLogger.Error().Err(err).Msg("run_groc_server_error")
			panic(err)
		} else {
			ctxLogger.Warn().Err(err).Msg("closed_grpc_server")
		}
	}()

	go func() {
		<-ctx.Done()
		ctxLogger.Warn().Msg("stopping_grpc_server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(shutdownTimeout)*time.Second)
		defer cancel()

		ok := make(chan struct{})
		go func() {
			// NOTE: GracefulStop will return instantly when Stop it called, preventing this goroutine from leaking.
			grpcServer.GracefulStop()
			close(ok)
		}()

		select {
		case <-ok:
			ctxLogger.Warn().Msg("gracefully_stopped_grpc_server")
		case <-shutdownCtx.Done():
			ctxLogger.Error().Msg("gracefully_stop_grpc_server_timeout")
			grpcServer.Stop()
			ctxLogger.Warn().Msg("force_stopped_grpc_server")
		}
		ctxLogger.Info().Msg("stopped_serving_grpc_server")
	}()

	return nil
}
