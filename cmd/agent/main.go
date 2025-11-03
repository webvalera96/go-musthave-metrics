package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"time"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"go.uber.org/fx"
)

// const pollInterval = 2
// const reportInterval = 10
// const baseURL = "localhost:8080" //TODO: move to configuration of agent

var MemoryMetrics = []string{
	"Alloc",
	"BuckHashSys",
	"Frees",
	"GCCPUFraction",
	"GCSys",
	"HeapAlloc",
	"HeapIdel",
	"HeapInuse",
	"HeapObjects",
	"HeapReleased",
	"HeapSys",
	"LastGC",
	"Lookups",
	"McacheInuse",
	"MCacheSys",
	"MSpanInuse",
	"MSpanSys",
	"Mallocs",
	"NextGC",
	"NumForcedGC",
	"NumGC",
	"OtherSys",
	"PauseTotalNs",
	"StackInuse",
	"StackSys",
	"Sys",
	"TotalAlloc",
}

type RuntimeMetrics struct {
	mu                   sync.Mutex
	RuntimeMemoryMetrics runtime.MemStats
	PollCount            uint64
	RandomValue          float64
}

func (rm *RuntimeMetrics) Set(m runtime.MemStats) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.RuntimeMemoryMetrics = m
	rm.PollCount++
	rm.RandomValue = rand.Float64()
}

func (rm *RuntimeMetrics) Get() runtime.MemStats {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.RuntimeMemoryMetrics
}

func sendMetric(client *http.Client, baseURL string, metricType string, metricName string, metricValue string) error {
	requestURL := fmt.Sprintf("http://%s/update/%s/%s/%s", baseURL, metricType, metricName, metricValue)
	request, err := http.NewRequest(http.MethodPost, requestURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("[%s] unable to send metric: %s, with value %s", time.Now().Format(time.RFC3339), metricName, metricValue)
	} else {
		fmt.Printf("[%s] %s is ok\n", time.Now().Format(time.RFC3339), requestURL)
	}

	return nil
}

func (rm *RuntimeMetrics) SendToMetricsStorage(client *http.Client) error {
	// send memory metrics
	value := reflect.ValueOf(rm.RuntimeMemoryMetrics)
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		metricName := typ.Field(i).Name
		if slices.Contains(MemoryMetrics, metricName) {

			fieldValue := value.Field(i).Interface()
			fieldType := reflect.TypeOf(fieldValue)
			var metricValue float64
			if fieldType == reflect.TypeOf(uint64(0)) {
				metricValue = float64(fieldValue.(uint64))
			} else if fieldType == reflect.TypeOf(uint32(0)) {
				metricValue = float64(fieldValue.(uint32))
			} else if fieldType == reflect.TypeOf(float64(0)) {
				metricValue = fieldValue.(float64)
			} else {
				fmt.Printf("unknown type of metrics: %v\n", fieldType)
				continue
			}

			// send gauge metrics of agent mem stats
			err := sendMetric(client, flagMetricsServer, models.Gauge, metricName, strconv.FormatFloat(metricValue, 'f', 3, 64))
			if err != nil {
				return err
			}
		}
	}
	// send poll counts
	err := sendMetric(client, flagMetricsServer, models.Counter, "PollCount", strconv.FormatUint(rm.PollCount, 10))
	if err != nil {
		return err
	}

	// send random value
	err = sendMetric(client, flagMetricsServer, models.Gauge, "RandomValue", strconv.FormatFloat(rm.RandomValue, 'f', 3, 64))
	if err != nil {
		return err
	}
	return nil
}

var flagMetricsServer string
var flagPollInterval int
var flagReportPollInterval int

const (
	EnvAddress        = "ADDRESS"
	EnvReportInterval = "REPORT_INTERVAL"
	EnvPollInterval   = "POLL_INTERVAL"
)

func parseFlags() {
	// Parse metrics server address
	var exist bool
	flagMetricsServer, exist = os.LookupEnv(EnvAddress)
	if !exist {
		flag.StringVar(&flagMetricsServer, "a", "localhost:8080", "address and port of metric server")
	}

	// Parse poll interval
	var pollInterval string
	pollInterval, exist = os.LookupEnv(EnvPollInterval)
	if exist {
		var err error
		flagPollInterval, err = strconv.Atoi(pollInterval)

		if err != nil {
			log.Fatal("unable to parse POLL_INTERVAL env value")
		}
	} else {
		flag.IntVar(&flagPollInterval, "p", 2, "poll interval in seconds")
	}

	// Parse flagreport interval
	var reportPollInterval string
	reportPollInterval, exist = os.LookupEnv(EnvReportInterval)
	if exist {
		var err error
		flagReportPollInterval, err = strconv.Atoi(reportPollInterval)

		if err != nil {
			log.Fatal("unable to parse REPORT_INTERVAL env value")
		}
	} else {
		flag.IntVar(&flagReportPollInterval, "r", 10, "report interval in seconds")
	}

	flag.Parse()
}

func MetricsUpdater(lc fx.Lifecycle) *RuntimeMetrics {
	rm := RuntimeMetrics{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				for {
					// Pause between metrics gathering
					time.Sleep(time.Second * time.Duration(flagPollInterval))

					// update current runtimeMetrics
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					rm.Set(m)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			rm = RuntimeMetrics{}
			return nil
		},
	})
	return &rm
}

func MetricsSender(lc fx.Lifecycle, runtimeMetrics *RuntimeMetrics) *http.Client {
	client := http.Client{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				for {
					time.Sleep(time.Second * time.Duration(flagReportPollInterval))
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

	parseFlags()

	fx.New(
		fx.Provide(
			MetricsUpdater,
			MetricsSender,
		),
		fx.Invoke(func(*http.Client) {}),
	).Run()

}
