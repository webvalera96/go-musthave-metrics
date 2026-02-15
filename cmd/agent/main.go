package main

import (
	"context"
	"net/http"
	"time"

	"github.com/webvalera96/go-musthave-metrics/internal/agent/flags"
	"github.com/webvalera96/go-musthave-metrics/internal/agent/metrics"
	"go.uber.org/fx"
)

// MetricsCollectorProvider создает коллектор метрик
func MetricsCollectorProvider(lc fx.Lifecycle, cfg *flags.AgentConfig) *metrics.MetricsCollector {
	collector := metrics.NewMetricsCollector(100) // буфер на 100 батчей

	var runCancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			runCtx, cancel := context.WithCancel(context.Background())
			runCancel = cancel

			pollInterval := time.Duration(cfg.PollInterval) * time.Second
			reportInterval := time.Duration(cfg.ReportPollInterval) * time.Second

			go func() {
				ticker := time.NewTicker(pollInterval)
				defer ticker.Stop()
				for {
					select {
					case <-runCtx.Done():
						return
					case <-ticker.C:
						collector.CollectRuntimeMetrics()
					}
				}
			}()

			go func() {
				ticker := time.NewTicker(pollInterval)
				defer ticker.Stop()
				for {
					select {
					case <-runCtx.Done():
						return
					case <-ticker.C:
						collector.CollectGopsutilMetrics()
					}
				}
			}()

			go func() {
				ticker := time.NewTicker(reportInterval)
				defer ticker.Stop()
				for {
					select {
					case <-runCtx.Done():
						return
					case <-ticker.C:
						collector.SendMetrics()
					}
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if runCancel != nil {
				runCancel()
			}
			return nil
		},
	})

	return collector
}

// HTTPClientProvider создает HTTP клиент
func HTTPClientProvider(lc fx.Lifecycle) *http.Client {
	client := &http.Client{}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			client.CloseIdleConnections()
			return nil
		},
	})
	return client
}

// WorkerPoolProvider создает и запускает пул воркеров
func WorkerPoolProvider(lc fx.Lifecycle, cfg *flags.AgentConfig, collector *metrics.MetricsCollector, client *http.Client) *metrics.WorkerPool {
	rateLimit := cfg.RateLimit
	if rateLimit < 1 {
		rateLimit = 1
	}

	pool := metrics.NewWorkerPool(
		client,
		cfg.MetricsServer,
		cfg.Key,
		rateLimit,
		collector.GetMetricsChan(),
	)

	var runCancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			runCtx, cancel := context.WithCancel(context.Background())
			runCancel = cancel
			pool.Start(runCtx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if runCancel != nil {
				runCancel()
			}
			pool.Stop()
			return nil
		},
	})

	return pool
}

func main() {
	fx.New(
		fx.Provide(
			flags.NewAgentConfig,
			MetricsCollectorProvider,
			HTTPClientProvider,
			WorkerPoolProvider,
		),
		fx.Invoke(func(*metrics.WorkerPool) {}),
	).Run()
}
