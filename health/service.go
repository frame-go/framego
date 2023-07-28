package health

import (
	"context"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/frame-go/framego/health/proto"
)

type Server = grpc_health_v1.HealthServer

type healthServer struct {
	grpc_health_v1.UnimplementedHealthServer

	reporter         Reporter
	statusReportLock sync.Mutex
	statusChans      []chan Status
}

func fillProtoStatus(resp *grpc_health_v1.HealthCheckResponse, status Status) {
	switch status.State {
	case StateHealthy:
		resp.Status = grpc_health_v1.HealthCheckResponse_SERVING
	case StateUnhealthy:
		resp.Status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
		resp.Status = grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN
	}
}

func (s *healthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	resp := &grpc_health_v1.HealthCheckResponse{}
	fillProtoStatus(resp, s.reporter.LastStatus())
	return resp, nil
}

func (s *healthServer) Watch(request *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	resp := &grpc_health_v1.HealthCheckResponse{}
	ch := s.newStatusChan()
	defer s.removeStatusChan(ch)
	for status := range ch {
		fillProtoStatus(resp, status)
		err := stream.Send(resp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *healthServer) newStatusChan() chan Status {
	s.statusReportLock.Lock()
	defer s.statusReportLock.Unlock()
	ch := make(chan Status, 1)
	s.statusChans = append(s.statusChans, ch)
	ch <- s.reporter.LastStatus() // send current status
	return ch
}

func (s *healthServer) removeStatusChan(ch chan Status) {
	s.statusReportLock.Lock()
	defer s.statusReportLock.Unlock()
	for i := range s.statusChans {
		if s.statusChans[i] == ch {
			s.statusChans = append(s.statusChans[:i], s.statusChans[i+1:]...)
		}
	}
	close(ch)
}

func (s *healthServer) broadcastStatus() {
	for status := range s.reporter.StatusReportChan() {
		s.statusReportLock.Lock()
		for _, ch := range s.statusChans {
			ch <- status
		}
		s.statusReportLock.Unlock()
	}
}

func NewServer(healthReporter Reporter) Server {
	s := &healthServer{
		reporter:    healthReporter,
		statusChans: make([]chan Status, 0),
	}
	go s.broadcastStatus()
	return s
}

func RegisterServer(s grpc.ServiceRegistrar, srv Server) {
	grpc_health_v1.RegisterHealthServer(s, srv)
}

func RegisterHandlerClient(ctx context.Context, mux *runtime.ServeMux, conn grpc.ClientConnInterface) error {
	return proto.RegisterHealthHandlerClient(ctx, mux, grpc_health_v1.NewHealthClient(conn))
}
