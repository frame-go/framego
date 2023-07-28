package grpcex

import (
	"context"
	"google.golang.org/grpc"
	"strings"

	"google.golang.org/grpc/metadata"
)

const tokenAuthHeader = "authorization"
const tokenAuthPrefix = "Bearer "

type TokenAuth struct {
	token string
}

func (t TokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		tokenAuthHeader: tokenAuthPrefix + t.token,
	}, nil
}

func (TokenAuth) RequireTransportSecurity() bool {
	return false
}

func NewTokenAuth(token string) *TokenAuth {
	return &TokenAuth{
		token: token,
	}
}

func WithAuthToken(token string) grpc.CallOption {
	return grpc.PerRPCCredentials(NewTokenAuth(token))
}

func GetAuthToken(ctx context.Context) (token string) {
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
	authes := meta.Get(tokenAuthHeader)
	if len(authes) > 0 {
		auth := authes[0]
		if strings.HasPrefix(auth, tokenAuthPrefix) {
			token = auth[len(tokenAuthPrefix):]
		}
	}
	return
}
