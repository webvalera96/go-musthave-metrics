package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/webvalera96/go-musthave-metrics/internal/agent/flags"
	"github.com/webvalera96/go-musthave-metrics/internal/agent/localip"
	"github.com/webvalera96/go-musthave-metrics/internal/agent/metrics"
	"github.com/webvalera96/go-musthave-metrics/internal/securepayload"
	"github.com/webvalera96/go-musthave-metrics/internal/shutdown"
	"go.uber.org/fx"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// MetricsCollectorProvider создает коллектор метрик
func MetricsCollectorProvider(lc fx.Lifecycle, cfg *flags.AgentConfig) *metrics.MetricsCollector {
	collector := metrics.NewMetricsCollector(100)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			runCtx, cancel := context.WithCancel(context.Background())
			collector.SetStopPollers(cancel)

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

// CryptoPublicKey загружает RSA-публичный ключ из пути в конфиге (если путь задан).
func CryptoPublicKey(cfg *flags.AgentConfig) (*rsa.PublicKey, error) {
	if cfg.CryptoKey == "" {
		return nil, nil
	}
	return securepayload.LoadPublicKey(cfg.CryptoKey)
}

// WorkerPoolProvider создает и запускает пул воркеров
func WorkerPoolProvider(lc fx.Lifecycle, cfg *flags.AgentConfig, collector *metrics.MetricsCollector, client *http.Client, pub *rsa.PublicKey) *metrics.WorkerPool {
	rateLimit := cfg.RateLimit
	if rateLimit < 1 {
		rateLimit = 1
	}

	pool := metrics.NewWorkerPool(
		client,
		cfg.MetricsServer,
		cfg.Key,
		pub,
		localip.Host(),
		rateLimit,
		collector.GetMetricsChan(),
	)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			runCtx, cancel := context.WithCancel(context.Background())
			pool.SetCancelRun(cancel)
			pool.Start(runCtx)
			return nil
		},
	})

	return pool
}

// AgentGracefulShutdown останавливает опрос, досылает батчи воркерам и ждёт отправки.
func AgentGracefulShutdown(lc fx.Lifecycle, col *metrics.MetricsCollector, pool *metrics.WorkerPool) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			col.StopPollers()
			col.FlushMetricsToChannel()
			pool.Shutdown()
			return nil
		},
	})
}

func main() {
	printBuildInfo()
	app := fx.New(
		fx.StopTimeout(30*time.Second),
		fx.Provide(
			flags.NewAgentConfig,
			CryptoPublicKey,
			MetricsCollectorProvider,
			HTTPClientProvider,
			WorkerPoolProvider,
		),
		fx.Invoke(func(*metrics.WorkerPool) {}),
		fx.Invoke(AgentGracefulShutdown),
	)

	startCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, shutdown.Signals()...)
	<-sigCh

	stopCtx, cancelStop := context.WithTimeout(context.Background(), app.StopTimeout())
	defer cancelStop()
	if err := app.Stop(stopCtx); err != nil {
		os.Exit(1)
	}
}

// printBuildInfo выводит информацию о версии сборки в stdout
func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}
	date := buildDate
	if date == "" {
		date = "N/A"
	}
	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Fprintf(os.Stdout, "Build version: %s\n", version)
	fmt.Fprintf(os.Stdout, "Build date: %s\n", date)
	fmt.Fprintf(os.Stdout, "Build commit: %s\n", commit)
}
