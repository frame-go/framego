package grpcex

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const MIMEEventStream = "text/event-stream"

var eventStreamPing = []byte(": ping\n\n")

type EventStreamMarshaler struct {
	runtime.Marshaler
}

func (m *EventStreamMarshaler) Delimiter() []byte {
	return nil
}

func (m *EventStreamMarshaler) Marshal(v interface{}) ([]byte, error) {
	switch v.(type) {
	case map[string]interface{}, map[string]proto.Message:
		data, err := m.Marshaler.Marshal(v)
		if err != nil {
			return nil, err
		}
		event := make([]byte, 0, len(data)+8)
		event = append(event, "data: "...)
		event = append(event, data...)
		return append(event, "\n\n"...), nil
	}
	return m.Marshaler.Marshal(v)
}

func EventStreamGatewayMiddleware(heartbeatInterval time.Duration) runtime.Middleware {
	return func(next runtime.HandlerFunc) runtime.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			if !slices.Contains(r.Header.Values("Accept"), MIMEEventStream) {
				next(w, r, pathParams)
				return
			}
			s := &eventStream{ResponseWriter: w, heartbeatInterval: heartbeatInterval}
			defer s.close()
			next(s, r.WithContext(context.WithValue(r.Context(), eventStreamKey{}, s)), pathParams)
		}
	}
}

// The gateway passes a nil message only when a stream begins.
func EventStreamForwardResponseOption(ctx context.Context, w http.ResponseWriter, msg proto.Message) error {
	s, ok := ctx.Value(eventStreamKey{}).(*eventStream)
	if !ok || msg != nil {
		return nil
	}
	return s.start()
}

// The gateway waits for the first server frame before forwarding a stream, and the in-process channel drops a
// SendHeader with empty metadata.
func EventStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, _ := metadata.FromIncomingContext(stream.Context())
		if info.IsServerStream && slices.Contains(md.Get(runtime.MetadataPrefix+"accept"), MIMEEventStream) {
			if err := stream.SendHeader(metadata.Pairs("x-event-stream", "1")); err != nil {
				return err
			}
		}
		return handler(srv, stream)
	}
}

type eventStreamKey struct{}

type eventStream struct {
	http.ResponseWriter
	heartbeatInterval time.Duration
	mu                sync.Mutex
	started           bool
	stop              chan struct{}
	done              chan struct{}
}

func (s *eventStream) WriteHeader(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		s.ResponseWriter.WriteHeader(code)
	}
}

func (s *eventStream) Write(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ResponseWriter.Write(b)
}

func (s *eventStream) FlushError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return http.NewResponseController(s.ResponseWriter).Flush()
}

func (s *eventStream) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func (s *eventStream) start() error {
	if s.started {
		return nil
	}
	h := s.Header()
	h.Set("Content-Type", MIMEEventStream)
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	s.WriteHeader(http.StatusOK)
	if err := s.FlushError(); err != nil {
		return err
	}
	s.started = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	go s.heartbeat()
	return nil
}

func (s *eventStream) heartbeat() {
	defer close(s.done)
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			if err := s.ping(); err != nil {
				return
			}
		}
	}
}

func (s *eventStream) ping() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.ResponseWriter.Write(eventStreamPing); err != nil {
		return err
	}
	return http.NewResponseController(s.ResponseWriter).Flush()
}

func (s *eventStream) close() {
	if s.started {
		close(s.stop)
		<-s.done
	}
}
