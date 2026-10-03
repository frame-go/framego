package grpcex

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/frame-go/framego/errors"
)

const AllowedHeaderPrefix = "x-"

const setCookieMetadataKey = "set-cookie"

// DefaultHeaderMatcher allows header with "x-" prefix
func DefaultHeaderMatcher(key string) (string, bool) {
	newKey, allowed := runtime.DefaultHeaderMatcher(key)
	if allowed {
		return newKey, allowed
	}
	key = strings.ToLower(key)
	if strings.HasPrefix(key, AllowedHeaderPrefix) {
		return runtime.MetadataPrefix + key, true
	}
	return "", false
}

// DefaultOutgoingHeaderMatcher passes "set-cookie" response metadata through as the Set-Cookie header,
// and keeps the "Grpc-Metadata-" prefix on all other keys so metadata cannot overwrite HTTP headers
func DefaultOutgoingHeaderMatcher(key string) (string, bool) {
	if strings.EqualFold(key, setCookieMetadataKey) {
		return "Set-Cookie", true
	}
	return runtime.MetadataHeaderPrefix + key, true
}

// GetHeader gets header from GRPC request header or HTTP converted header
func GetHeader(ctx context.Context, key string) string {
	var value string
	key = strings.ToLower(key)
	meta, _ := metadata.FromIncomingContext(ctx)
	values, ok := meta[key]
	if ok {
		if len(values) > 0 {
			value = values[0]
		}
		return value
	}
	key = runtime.MetadataPrefix + key
	values, ok = meta[key]
	if ok {
		if len(values) > 0 {
			value = values[0]
		}
	}
	return value
}

// GetClientIP gets client IP from HTTP forward header or GRPC context
func GetClientIP(ctx context.Context) string {
	ip := ""
	meta, ok := metadata.FromIncomingContext(ctx)
	if ok {
		ips := meta.Get("x-forwarded-for")
		if len(ips) > 0 {
			ip = ips[0]
		}
		comma := strings.IndexRune(ip, ',')
		if comma > 0 {
			ip = ip[:comma]
		}
	}
	if ip == "" {
		p, ok := peer.FromContext(ctx)
		if ok {
			ip = p.Addr.String()
		}
	}
	return ip
}

// GetCookie gets the named request cookie from the "cookie" gRPC metadata or the Cookie header forwarded by the gateway,
// returning http.ErrNoCookie when it is absent
func GetCookie(ctx context.Context, name string) (*http.Cookie, error) {
	meta, _ := metadata.FromIncomingContext(ctx)
	header := http.Header{"Cookie": slices.Concat(meta.Get("cookie"), meta.Get(runtime.MetadataPrefix+"cookie"))}
	req := http.Request{Header: header}
	return req.Cookie(name)
}

// SetCookie adds a Set-Cookie header to the gateway response; each call adds one header
func SetCookie(ctx context.Context, cookie *http.Cookie) error {
	value := cookie.String()
	if value == "" {
		return errors.New("invalid_cookie").With("name", cookie.Name)
	}
	return grpc.SetHeader(ctx, metadata.Pairs(setCookieMetadataKey, value))
}
