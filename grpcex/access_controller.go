package grpcex

import (
	"context"
	"path"
	"strings"

	casbin "github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/frame-go/framego/errors"
	"github.com/frame-go/framego/log"
)

const (
	reflectionServiceName = "/grpc.reflection.v1alpha.ServerReflection/"
	channelzServiceName   = "/grpc.channelz.v1.Channelz/"
	healthServiceName     = "/grpc.health.v1.Health/"
	anonymousServiceName  = "<anonymous>"
	defaultCasbinModel    = `
[request_definition]
r = sub, obj

[policy_definition]
p = sub, obj

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && (g(r.obj, p.obj) || globMatch(r.obj, p.obj))
`
)

type AccessController struct {
	enforcer *casbin.Enforcer
}

func NewAccessController(modelFile string, policyFile string) (*AccessController, error) {
	var err error
	var m model.Model
	if modelFile == "" {
		m, err = model.NewModelFromString(defaultCasbinModel)
		if err != nil {
			return nil, errors.Wrap(err, "new_default_casbin_model_error").With("model", defaultCasbinModel)
		}
	} else {
		m, err = model.NewModelFromFile(modelFile)
		if err != nil {
			return nil, errors.Wrap(err, "new_casbin_model_from_file_error").With("model_file", modelFile)
		}
	}
	a := fileadapter.NewAdapter(policyFile)
	enforcer, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return nil, errors.Wrap(err, "new_casbin_enforcer_error").
			With("model", m.ToText()).With("policy_file", policyFile)
	}
	enforcer.AddNamedMatchingFunc("g", "", globMatch)
	c := &AccessController{
		enforcer: enforcer,
	}
	return c, nil
}

func (c *AccessController) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		err := c.checkAccess(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (c *AccessController) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := c.checkAccess(stream.Context(), info.FullMethod)
		if err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

func (c *AccessController) checkAccess(ctx context.Context, method string) error {
	if strings.HasPrefix(method, reflectionServiceName) ||
		strings.HasPrefix(method, channelzServiceName) ||
		strings.HasPrefix(method, healthServiceName) {
		return nil
	}
	delimiter := strings.LastIndex(method, "/")
	if delimiter >= 0 {
		method = method[delimiter+1:]
	}
	client, ok := peer.FromContext(ctx)
	if !ok {
		return errors.New("peer_not_found_in_context").WithGRPCCode(codes.Unauthenticated)
	}
	authType := client.AuthInfo.AuthType()
	switch authType {
	case "inproc":
		if c.checkServiceAccess(anonymousServiceName, method) {
			return nil
		}
		return errors.New("access_denied").With("client_service", anonymousServiceName).
			WithGRPCCode(codes.Unauthenticated)
	case "tls":
		if authType == "inproc" {
		} else if authType != "tls" {
		}
		tlsInfo, ok := client.AuthInfo.(credentials.TLSInfo)
		if !ok {
			return errors.New("auth_tls_info_not_found").WithGRPCCode(codes.Unauthenticated)
		}
		if len(tlsInfo.State.PeerCertificates) == 0 {
			return errors.New("peer_cert_not_found").WithGRPCCode(codes.Unauthenticated)
		}
		clientCert := tlsInfo.State.PeerCertificates[0]
		serviceName := clientCert.Subject.CommonName
		if serviceName != "" {
			if c.checkServiceAccess(serviceName, method) {
				return nil
			}
		}
		for _, serviceName = range clientCert.DNSNames {
			if c.checkServiceAccess(serviceName, method) {
				return nil
			}
		}
		return errors.New("access_denied").With("client", serviceName).
			With("method", method).WithGRPCCode(codes.Unauthenticated)
	default:
		return errors.New("auth_type_not_supported").With("auth_type", authType).
			WithGRPCCode(codes.Unauthenticated)
	}
}

func (c *AccessController) checkServiceAccess(service string, method string) bool {
	ok, err := c.enforcer.Enforce(service, method)
	if err != nil {
		log.Logger.Error().Err(err).Str("client", service).Str("method", method).Msg("enforce_check_error")
		return false
	}
	return ok
}

func globMatch(key1 string, key2 string) bool {
	ok, _ := path.Match(key2, key1)
	return ok
}
