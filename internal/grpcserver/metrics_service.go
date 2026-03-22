package grpcserver

import (
	"context"
	"fmt"
	"time"

	"github.com/webvalera96/go-musthave-metrics/internal/audit"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	pb "github.com/webvalera96/go-musthave-metrics/internal/proto/metrics"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UpdateMetricsHandler реализует gRPC-сервис Metrics.
type UpdateMetricsHandler struct {
	pb.UnimplementedMetricsServer
	storage      *repository.MemoryMetricsStorage
	auditSubject *audit.Subject
}

// NewUpdateMetricsHandler создаёт обработчик gRPC.
func NewUpdateMetricsHandler(ms *repository.MemoryMetricsStorage, audit *audit.Subject) *UpdateMetricsHandler {
	return &UpdateMetricsHandler{storage: ms, auditSubject: audit}
}

// UpdateMetrics сохраняет батч метрик.
func (s *UpdateMetricsHandler) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	metricNames := make([]string, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		if m == nil || m.Id == "" {
			return nil, status.Error(codes.InvalidArgument, "metric id is required")
		}
		model, err := protoMetricToModel(m)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		if err := s.storage.Set(model); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		metricNames = append(metricNames, m.Id)
	}

	ip := clientIPFromGRPCContext(ctx)
	s.auditSubject.NotifyAll(audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   metricNames,
		IPAddress: ip,
	})

	return &pb.UpdateMetricsResponse{}, nil
}

func protoMetricToModel(m *pb.Metric) (*models.Metrics, error) {
	switch m.Type {
	case pb.Metric_GAUGE:
		return &models.Metrics{
			ID:    m.Id,
			MType: models.Gauge,
			Value: &m.Value,
		}, nil
	case pb.Metric_COUNTER:
		d := m.Delta
		return &models.Metrics{
			ID:    m.Id,
			MType: models.Counter,
			Delta: &d,
		}, nil
	default:
		return nil, fmt.Errorf("unknown metric type")
	}
}

func clientIPFromGRPCContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(handler.XRealIPMetadataKey)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
