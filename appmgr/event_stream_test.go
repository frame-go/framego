package appmgr

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/frame-go/framego/grpcex"
	"github.com/frame-go/framego/health"
)

const testHeartbeatInterval = 20 * time.Millisecond

var (
	serving    = &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}
	notServing = &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}
)

type testHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	watch func(grpc_health_v1.Health_WatchServer) error
}

func (s *testHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return serving, nil
}

func (s *testHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return s.watch(stream)
}

type testStreamInterceptorMiddleware struct {
	interceptor grpc.StreamServerInterceptor
}

func (m *testStreamInterceptorMiddleware) GinHandler(options map[string]interface{}) gin.HandlerFunc {
	return nil
}

func (m *testStreamInterceptorMiddleware) GrpcServerInterceptor(options map[string]interface{}) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return nil, m.interceptor
}

func (m *testStreamInterceptorMiddleware) GrpcClientInterceptor(options map[string]interface{}) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	return nil, nil
}

func newEventStreamTestServer(t *testing.T, watch func(grpc_health_v1.Health_WatchServer) error, interceptors ...grpc.StreamServerInterceptor) *httptest.Server {
	mm := newMiddlewareManager()
	configs := make([]interface{}, 0, len(interceptors))
	for i, interceptor := range interceptors {
		name := fmt.Sprintf("test_interceptor_%d", i)
		mm.RegisterMiddleware(name, &testStreamInterceptorMiddleware{interceptor: interceptor})
		configs = append(configs, name)
	}
	return newEventStreamTestServerWithMiddlewares(t, watch, mm, configs)
}

func newEventStreamTestServerWithMiddlewares(t *testing.T, watch func(grpc_health_v1.Health_WatchServer) error, mm *middlewareManager, configs []interface{}) *httptest.Server {
	gin.SetMode(gin.TestMode)
	middlewares := mm.Apply(&serviceImpl{name: "test"}, configs)
	_, channel := newGrpcServerWithChannel(nil, middlewares)
	channel.RegisterService(&grpc_health_v1.Health_ServiceDesc, &testHealthServer{watch: watch})
	mux := newGrpcHttpMux(testHeartbeatInterval)
	require.NoError(t, health.RegisterHandlerClient(context.Background(), mux, channel))
	registerWatchHandler(t, mux, grpc_health_v1.NewHealthClient(channel))
	engine := newGinEngin(middlewares)
	engine.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusOK)
		mux.ServeHTTP(c.Writer, c.Request)
	})
	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)
	return server
}

// registerWatchHandler mirrors the handler protoc-gen-grpc-gateway generates for a server-streaming RPC.
func registerWatchHandler(t *testing.T, mux *runtime.ServeMux, client grpc_health_v1.HealthClient) {
	err := mux.HandlePath(http.MethodGet, "/health/v1/watch", func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		_, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		annotatedContext, err := runtime.AnnotateContext(ctx, mux, req, "/grpc.health.v1.Health/Watch", runtime.WithHTTPPathPattern("/health/v1/watch"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		var md runtime.ServerMetadata
		stream, err := client.Watch(annotatedContext, &grpc_health_v1.HealthCheckRequest{})
		if err == nil {
			md.HeaderMD, err = stream.Header()
		}
		annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
		runtime.ForwardResponseStream(annotatedContext, mux, outboundMarshaler, w, req, func() (proto.Message, error) {
			return stream.Recv()
		}, mux.GetForwardResponseOptions()...)
	})
	require.NoError(t, err)
}

func get(t *testing.T, ctx context.Context, url string, accepts ...string) *http.Response {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	for _, accept := range accepts {
		req.Header.Add("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func parseEvents(t *testing.T, body string) (data []string, pings int) {
	require.True(t, strings.HasSuffix(body, "\n\n"), "body: %q", body)
	for _, event := range strings.Split(strings.TrimSuffix(body, "\n\n"), "\n\n") {
		switch {
		case event == ": ping":
			pings++
		case strings.HasPrefix(event, "data: "):
			data = append(data, strings.TrimPrefix(event, "data: "))
		default:
			t.Errorf("malformed_event: %q", event)
		}
	}
	return data, pings
}

func readEvents(t *testing.T, resp *http.Response) (data []string, pings int) {
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return parseEvents(t, string(body))
}

func assertEventStreamHeaders(t *testing.T, resp *http.Response) {
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, grpcex.MIMEEventStream, resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "no", resp.Header.Get("X-Accel-Buffering"))
}

func waitRelease(stream grpc_health_v1.Health_WatchServer, release <-chan struct{}) error {
	select {
	case <-release:
		return nil
	case <-stream.Context().Done():
		return stream.Context().Err()
	}
}

func TestEventStreamMessagesAndHeartbeat(t *testing.T) {
	release := make(chan struct{})
	server := newEventStreamTestServer(t, func(stream grpc_health_v1.Health_WatchServer) error {
		if err := stream.Send(serving); err != nil {
			return err
		}
		if err := waitRelease(stream, release); err != nil {
			return err
		}
		return stream.Send(notServing)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp := get(t, ctx, server.URL+"/health/v1/watch", grpcex.MIMEEventStream)
	assertEventStreamHeaders(t, resp)
	reader := bufio.NewReader(resp.Body)
	var head strings.Builder
	for !strings.HasSuffix(head.String(), ": ping\n\n") {
		line, err := reader.ReadString('\n')
		require.NoError(t, err)
		head.WriteString(line)
	}
	close(release)
	rest, err := io.ReadAll(reader)
	require.NoError(t, err)
	data, pings := parseEvents(t, head.String()+string(rest))
	require.Len(t, data, 2)
	assert.JSONEq(t, `{"result":{"status":1}}`, data[0])
	assert.JSONEq(t, `{"result":{"status":2}}`, data[1])
	assert.GreaterOrEqual(t, pings, 1)
}

func TestEventStreamStartsBeforeFirstMessage(t *testing.T) {
	release := make(chan struct{})
	server := newEventStreamTestServer(t, func(stream grpc_health_v1.Health_WatchServer) error {
		if err := waitRelease(stream, release); err != nil {
			return err
		}
		return stream.Send(serving)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp := get(t, ctx, server.URL+"/health/v1/watch", grpcex.MIMEEventStream)
	assertEventStreamHeaders(t, resp)
	close(release)
	data, _ := readEvents(t, resp)
	require.Len(t, data, 1)
	assert.JSONEq(t, `{"result":{"status":1}}`, data[0])
}

func TestEventStreamAcceptAmongMultipleHeaders(t *testing.T) {
	server := newEventStreamTestServer(t, func(stream grpc_health_v1.Health_WatchServer) error {
		return stream.Send(serving)
	})
	resp := get(t, context.Background(), server.URL+"/health/v1/watch", "application/json", grpcex.MIMEEventStream)
	assertEventStreamHeaders(t, resp)
	data, _ := readEvents(t, resp)
	require.Len(t, data, 1)
	assert.JSONEq(t, `{"result":{"status":1}}`, data[0])
}

func TestEventStreamNotCompressed(t *testing.T) {
	release := make(chan struct{})
	mm := newMiddlewareManager()
	mm.RegisterMiddleware("compress", NewCompressMiddleware())
	server := newEventStreamTestServerWithMiddlewares(t, func(stream grpc_health_v1.Health_WatchServer) error {
		if err := stream.Send(serving); err != nil {
			return err
		}
		return waitRelease(stream, release)
	}, mm, []interface{}{"compress"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/health/v1/watch", nil)
	require.NoError(t, err)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept", grpcex.MIMEEventStream)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assertEventStreamHeaders(t, resp)
	assert.Empty(t, resp.Header.Get("Content-Encoding"))
	reader := bufio.NewReader(resp.Body)
	var head strings.Builder
	for !strings.HasSuffix(head.String(), ": ping\n\n") {
		line, err := reader.ReadString('\n')
		require.NoError(t, err)
		head.WriteString(line)
	}
	close(release)
	data, _ := parseEvents(t, head.String())
	require.Len(t, data, 1)
	assert.JSONEq(t, `{"result":{"status":1}}`, data[0])
}

func TestEventStreamHandlerError(t *testing.T) {
	server := newEventStreamTestServer(t, func(stream grpc_health_v1.Health_WatchServer) error {
		return status.Error(codes.Unauthenticated, "unauthenticated")
	})
	resp := get(t, context.Background(), server.URL+"/health/v1/watch", grpcex.MIMEEventStream)
	assertEventStreamHeaders(t, resp)
	data, _ := readEvents(t, resp)
	require.Len(t, data, 1)
	assert.JSONEq(t, `{"error":{"code":16,"message":"unauthenticated","details":[]}}`, data[0])
}

func TestEventStreamInterceptorError(t *testing.T) {
	reject := func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return status.Error(codes.PermissionDenied, "permission_denied")
	}
	server := newEventStreamTestServer(t, nil, reject)
	resp := get(t, context.Background(), server.URL+"/health/v1/watch", grpcex.MIMEEventStream)
	assertEventStreamHeaders(t, resp)
	data, _ := readEvents(t, resp)
	require.Len(t, data, 1)
	assert.JSONEq(t, `{"error":{"code":7,"message":"permission_denied","details":[]}}`, data[0])
}

func TestEventStreamClientDisconnect(t *testing.T) {
	handlerDone := make(chan struct{})
	server := newEventStreamTestServer(t, func(stream grpc_health_v1.Health_WatchServer) error {
		defer close(handlerDone)
		<-stream.Context().Done()
		return stream.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp := get(t, ctx, server.URL+"/health/v1/watch", grpcex.MIMEEventStream)
	assertEventStreamHeaders(t, resp)
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, ": ping\n", line)
	cancel()
	select {
	case <-handlerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("handler_not_canceled")
	}
}

func TestEventStreamNotAccepted(t *testing.T) {
	server := newEventStreamTestServer(t, func(stream grpc_health_v1.Health_WatchServer) error {
		if err := stream.Send(serving); err != nil {
			return err
		}
		time.Sleep(5 * testHeartbeatInterval)
		return stream.Send(notServing)
	})
	resp := get(t, context.Background(), server.URL+"/health/v1/watch", "application/json")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	require.Len(t, lines, 2, "body: %q", body)
	assert.JSONEq(t, `{"result":{"status":1}}`, lines[0])
	assert.JSONEq(t, `{"result":{"status":2}}`, lines[1])
}

func TestEventStreamUnaryUnaffected(t *testing.T) {
	server := newEventStreamTestServer(t, nil)
	resp := get(t, context.Background(), server.URL+"/health/v1/check", grpcex.MIMEEventStream)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"status":1}`, string(body))
}
