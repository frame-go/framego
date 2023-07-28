package appmgr

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"math"
	"os"
	"time"

	"github.com/fullstorydev/grpchan/inprocgrpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/grpcex"
	"github.com/frame-go/framego/log"
)

const ServerConnMaxIdle = time.Duration(math.MaxInt64)
const ServerConnMaxAge = time.Duration(math.MaxInt64)
const ServerConnMaxAgeGrace = time.Duration(math.MaxInt64)
const KeepaliveMinTime = 10 * time.Second
const KeepaliveTime = 1 * time.Minute
const KeepaliveTimeout = 20 * time.Second

func initGrpc() {
	grpcex.SetZeroLogger()
	grpcex.RegisterJsonCodec()
}

func newGrpcServerWithChannel(tlsConfig *tls.Config, middlewares *middlewareApplier) (*grpc.Server, *inprocgrpc.Channel) {
	chainUnaryInterceptor, chainStreamInterceptor := middlewares.GrpcServerInterceptor()
	serverOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(chainUnaryInterceptor),
		grpc.StreamInterceptor(chainStreamInterceptor),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     ServerConnMaxIdle,
			MaxConnectionAge:      ServerConnMaxAge,
			MaxConnectionAgeGrace: ServerConnMaxAgeGrace,
			Time:                  KeepaliveTime,
			Timeout:               KeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             KeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	}
	if tlsConfig != nil {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		creds := credentials.NewTLS(tlsConfig)
		serverOptions = append(serverOptions, grpc.Creds(creds))
	}
	server := grpc.NewServer(serverOptions...)
	grpcChannel := &inprocgrpc.Channel{}
	grpcChannel.WithServerUnaryInterceptor(chainUnaryInterceptor)
	grpcChannel.WithServerStreamInterceptor(chainStreamInterceptor)
	return server, grpcChannel
}

func newGrpcClient(ctx context.Context, name string, endpoint string, tlsConfig *tls.Config, middlewares *middlewareApplier) grpc.ClientConnInterface {
	var creds credentials.TransportCredentials
	if tlsConfig == nil {
		creds = insecure.NewCredentials()
	} else {
		tlsConfig.ServerName = name
		creds = credentials.NewTLS(tlsConfig)
	}
	chainUnaryInterceptor, chainStreamInterceptor := middlewares.GrpcClientInterceptor()
	conn, err := grpc.Dial(
		endpoint,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithUnaryInterceptor(chainUnaryInterceptor),
		grpc.WithStreamInterceptor(chainStreamInterceptor),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                KeepaliveTime,
			Timeout:             KeepaliveTimeout,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		log.Logger.Error().Err(err).Str("endpoint", endpoint).Msg("new_grpc_client_dial_error")
		return nil
	}
	return conn
}

func newGrpcHttpMux() *runtime.ServeMux {
	return runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				UseEnumNumbers:  true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
		runtime.WithIncomingHeaderMatcher(grpcex.DefaultHeaderMatcher),
	)
}

func newTLSConfig(certFile string, keyFile string, caFile string) (*tls.Config, error) {
	if certFile == "" && keyFile == "" && caFile == "" {
		return nil, nil
	}
	tlsConfig := &tls.Config{}
	if certFile != "" && keyFile != "" {
		certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, errors.Wrap(err, "load_cert_key_pair_error").
				With("cert_file", certFile).With("key_file", keyFile)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}

	}
	if caFile != "" {
		certPool := x509.NewCertPool()
		ca, err := os.ReadFile(caFile)
		if err != nil {
			return nil, errors.Wrap(err, "read_ca_file_error").With("ca_file", caFile)
		}
		if !certPool.AppendCertsFromPEM(ca) {
			return nil, errors.New("add_ca_cert_error").With("ca_file", caFile)
		}
		tlsConfig.RootCAs = certPool
		tlsConfig.ClientCAs = certPool
	}
	return tlsConfig, nil
}
