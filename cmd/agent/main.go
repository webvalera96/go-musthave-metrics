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
func MetricsCollectorProvider(lc fx.Lifecycle) *metrics.MetricsCollector {
	collector := metrics.NewMetricsCollector(100) // буфер на 100 батчей

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Горутина для сбора runtime метрик
			go func() {
				ticker := time.NewTicker(time.Second * time.Duration(flags.FlagPollInterval))
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						collector.CollectRuntimeMetrics()
					}
				}
			}()

			// Горутина для сбора gopsutil метрик
			go func() {
				ticker := time.NewTicker(time.Second * time.Duration(flags.FlagPollInterval))
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						collector.CollectGopsutilMetrics()
					}
				}
			}()

			// Горутина для отправки метрик в канал
			go func() {
				ticker := time.NewTicker(time.Second * time.Duration(flags.FlagReportPollInterval))
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						collector.SendMetrics()
					}
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
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
func WorkerPoolProvider(lc fx.Lifecycle, collector *metrics.MetricsCollector, client *http.Client) *metrics.WorkerPool {
	rateLimit := flags.FlagRateLimit
	if rateLimit < 1 {
		rateLimit = 1
	}

	pool := metrics.NewWorkerPool(
		client,
		flags.FlagMetricsServer,
		rateLimit,
		collector.GetMetricsChan(),
	)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			pool.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			pool.Stop()
			return nil
		},
	})

	return pool
}

func main() {
	flags.ParseFlags()

	fx.New(
		fx.Provide(
			MetricsCollectorProvider,
			HTTPClientProvider,
			WorkerPoolProvider,
		),
		fx.Invoke(func(*metrics.WorkerPool) {}),
	).Run()
}
