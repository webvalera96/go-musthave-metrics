package grpcserver

import (
	"context"
	"fmt"
	"net"

	"github.com/webvalera96/go-musthave-metrics/internal/audit"
	"github.com/webvalera96/go-musthave-metrics/internal/flags"
	pb "github.com/webvalera96/go-musthave-metrics/internal/proto/metrics"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

// RegisterGRPC запускает gRPC-сервер Metrics, если в конфиге задан GRPCAddr.
func RegisterGRPC(lc fx.Lifecycle, cfg *flags.ServerConfig, ms *repository.MemoryMetricsStorage, audit *audit.Subject) error {
	if cfg.GRPCAddr == "" {
		return nil
	}

	var trusted *net.IPNet
	if cfg.TrustedSubnet != "" {
		_, n, err := net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			return fmt.Errorf("invalid trusted_subnet: %w", err)
		}
		trusted = n
	}

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(TrustedSubnetUnaryInterceptor(trusted)))
	pb.RegisterMetricsServer(srv, NewUpdateMetricsHandler(ms, audit))

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", cfg.GRPCAddr)
			if err != nil {
				return err
			}
			fmt.Println("Starting gRPC serve at", cfg.GRPCAddr)
			go func() {
				if err := srv.Serve(ln); err != nil {
					fmt.Printf("gRPC server error: %v\n", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopped := make(chan struct{})
			go func() {
				srv.GracefulStop()
				close(stopped)
			}()
			select {
			case <-stopped:
			case <-ctx.Done():
				srv.Stop()
			}
			return nil
		},
	})
	return nil
}
