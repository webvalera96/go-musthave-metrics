package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/webvalera96/go-musthave-metrics/internal/agent/flags"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
)

var MemoryMetrics = []string{
	"Alloc",
	"BuckHashSys",
	"Frees",
	"GCCPUFraction",
	"GCSys",
	"HeapAlloc",
	"HeapIdle",
	"HeapInuse",
	"HeapObjects",
	"HeapReleased",
	"HeapSys",
	"LastGC",
	"Lookups",
	"MCacheInuse",
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
	requestURL := fmt.Sprintf("http://%s/update/", baseURL)

	var metric models.Metrics

	if metricType == models.Counter {
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return err
		}

		metric = models.Metrics{
			ID:    metricName,
			MType: models.Counter,
			Delta: &delta,
		}
	} else if metricType == models.Gauge {
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return err
		}

		metric = models.Metrics{
			ID:    metricName,
			MType: models.Gauge,
			Value: &value,
		}
	} else {
		return errors.New("wrong type of metric")
	}

	body, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

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
			err := sendMetric(client, flags.FlagMetricsServer, models.Gauge, metricName, strconv.FormatFloat(metricValue, 'f', 3, 64))
			if err != nil {
				return err
			}

		}
	}
	// send poll counts
	err := sendMetric(client, flags.FlagMetricsServer, models.Counter, "PollCount", strconv.FormatUint(rm.PollCount, 10))
	if err != nil {
		return err
	}

	// send random value
	err = sendMetric(client, flags.FlagMetricsServer, models.Gauge, "RandomValue", strconv.FormatFloat(rm.RandomValue, 'f', 3, 64))
	if err != nil {
		return err
	}
	return nil
}

//func sendMetric(client *http.Client, baseURL string, metricType string, metricName string, metricValue string) error {
//	requestURL := fmt.Sprintf("http://%s/update/%s/%s/%s", baseURL, metricType, metricName, metricValue)
//	request, err := http.NewRequest(http.MethodPost, requestURL, nil)
//	if err != nil {
//		return err
//	}
//	request.Header.Set("Content-Type", "text/plain")
//	response, err := client.Do(request)
//	if err != nil {
//		return err
//	}
//	defer response.Body.Close()
//	if response.StatusCode != http.StatusOK {
//		return fmt.Errorf("[%s] unable to send metric: %s, with value %s", time.Now().Format(time.RFC3339), metricName, metricValue)
//	} else {
//		fmt.Printf("[%s] %s is ok\n", time.Now().Format(time.RFC3339), requestURL)
//	}
//
//	return nil
//}
