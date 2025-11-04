package main

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/webvalera96/go-musthave-metrics/internal/agent/flags"
	"github.com/webvalera96/go-musthave-metrics/internal/agent/metrics"
	"go.uber.org/fx"
)

// const pollInterval = 2
// const reportInterval = 10
// const baseURL = "localhost:8080" //TODO: move to configuration of agent

func MetricsUpdater(lc fx.Lifecycle) *metrics.RuntimeMetrics {
	rm := metrics.RuntimeMetrics{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				for {
					// Pause between metrics gathering
					time.Sleep(time.Second * time.Duration(flags.FlagPollInterval))

					// update current runtimeMetrics
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					rm.Set(m)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			rm = metrics.RuntimeMetrics{}
			return nil
		},
	})
	return &rm
}

func MetricsSender(lc fx.Lifecycle, runtimeMetrics *metrics.RuntimeMetrics) *http.Client {
	client := http.Client{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				for {
					time.Sleep(time.Second * time.Duration(flags.FlagReportPollInterval))
					runtimeMetrics.SendToMetricsStorage(&client)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			client.CloseIdleConnections()
			return nil
		},
	})
	return &client
}

func main() {

	flags.ParseFlags()

	fx.New(
		fx.Provide(
			MetricsUpdater,
			MetricsSender,
		),
		fx.Invoke(func(*http.Client) {}),
	).Run()

}
