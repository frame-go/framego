package grpcex

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestDefaultOutgoingHeaderMatcher(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string
	}{
		{name: "set-cookie passes through", key: "set-cookie", want: "Set-Cookie"},
		{name: "set-cookie in any case", key: "Set-Cookie", want: "Set-Cookie"},
		{name: "other metadata keeps prefix", key: "x-request-id", want: "Grpc-Metadata-x-request-id"},
		{name: "standard header keeps prefix", key: "content-type", want: "Grpc-Metadata-content-type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := DefaultOutgoingHeaderMatcher(tc.key)
			assert.True(t, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetCookie(t *testing.T) {
	cases := []struct {
		name  string
		md    metadata.MD
		want  string
		found bool
	}{
		{name: "gateway-forwarded header", md: metadata.Pairs("grpcgateway-cookie", "x=0; session=abc"), want: "abc", found: true},
		{name: "native metadata", md: metadata.Pairs("cookie", "session=abc"), want: "abc", found: true},
		{name: "absent", md: metadata.Pairs("grpcgateway-cookie", "x=0"), found: false},
		{name: "no metadata", md: nil, found: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.md != nil {
				ctx = metadata.NewIncomingContext(ctx, tc.md)
			}
			cookie, err := GetCookie(ctx, "session")
			if !tc.found {
				assert.ErrorIs(t, err, http.ErrNoCookie)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, cookie.Value)
		})
	}
}

type headerRecorder struct {
	header metadata.MD
}

func (r *headerRecorder) Method() string { return "/test.Service/Method" }

func (r *headerRecorder) SetHeader(md metadata.MD) error {
	r.header = metadata.Join(r.header, md)
	return nil
}

func (r *headerRecorder) SendHeader(md metadata.MD) error { return r.SetHeader(md) }

func (r *headerRecorder) SetTrailer(metadata.MD) error { return nil }

func TestSetCookie(t *testing.T) {
	recorder := &headerRecorder{}
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), recorder)

	require.NoError(t, SetCookie(ctx, &http.Cookie{Name: "a", Value: "1", Path: "/", HttpOnly: true}))
	require.NoError(t, SetCookie(ctx, &http.Cookie{Name: "b", Value: "2"}))
	assert.Equal(t, []string{"a=1; Path=/; HttpOnly", "b=2"}, recorder.header.Get(setCookieMetadataKey))

	assert.Error(t, SetCookie(ctx, &http.Cookie{Name: "bad name", Value: "1"}))
}
