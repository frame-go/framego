package grpcex

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const AllowedHeaderPrefix = "x-"

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
