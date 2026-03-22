package grpcserver

import (
	"context"
	"net"
	"strings"

	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TrustedSubnetUnaryInterceptor проверяет метаданные x-real-ip при заданной доверенной подсети.
// trusted == nil — проверка отключена.
func TrustedSubnetUnaryInterceptor(trusted *net.IPNet) grpc.UnaryServerInterceptor {
	if trusted == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
			return h(ctx, req)
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing metadata")
		}
		vals := md.Get(handler.XRealIPMetadataKey)
		if len(vals) == 0 {
			vals = md.Get(strings.ToLower(handler.XRealIPHeader))
		}
		ipStr := ""
		if len(vals) > 0 {
			ipStr = strings.TrimSpace(vals[0])
		}
		if ipStr == "" {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip")
		}
		ip := net.ParseIP(ipStr)
		if ip == nil || !trusted.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "ip not in trusted subnet")
		}
		return h(ctx, req)
	}
}
