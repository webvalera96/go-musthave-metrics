package metrics

import (
	"bytes"
	"crypto/rsa"
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

	"github.com/webvalera96/go-musthave-metrics/internal/agent/localip"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	"github.com/webvalera96/go-musthave-metrics/internal/hash"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"github.com/webvalera96/go-musthave-metrics/internal/retry"
	"github.com/webvalera96/go-musthave-metrics/internal/securepayload"
	"github.com/webvalera96/go-musthave-metrics/internal/zip"
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

func sendMetric(
	client *http.Client,
	baseURL string,
	metricType string,
	metricName string,
	metricValue string,
) error {
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

	compressedBody, err := zip.Compress(body)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(compressedBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set(handler.XRealIPHeader, localip.Host())

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

func sendMetricsBatch(
	client *http.Client,
	baseURL string,
	hashKey string,
	pubKey *rsa.PublicKey,
	hostIP string,
	metrics []models.Metrics,
) error {
	requestURL := fmt.Sprintf("http://%s/updates/", baseURL)

	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	compressedBody, err := zip.Compress(body)
	if err != nil {
		return err
	}

	payload := compressedBody
	if pubKey != nil {
		payload, err = securepayload.Encrypt(pubKey, compressedBody)
		if err != nil {
			return err
		}
	}

	// Используем retry логику для обработки временных ошибок соединения
	err = retry.Retry(func() error {
		request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Encoding", "gzip")
		if pubKey != nil {
			request.Header.Set(securepayload.HTTPHeaderEncrypted, "1")
		}

		if hashKey != "" {
			hashValue := hash.CalculateHash(compressedBody, hashKey)
			request.Header.Set("HashSHA256", hashValue)
		}
		if hostIP != "" {
			request.Header.Set(handler.XRealIPHeader, hostIP)
		}

		response, err := client.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			// Не retriable ошибка - не повторяем
			return fmt.Errorf("[%s] unable to send metrics batch, status: %d", time.Now().Format(time.RFC3339), response.StatusCode)
		}
		fmt.Printf("[%s] %s is ok (sent %d metrics)\n", time.Now().Format(time.RFC3339), requestURL, len(metrics))
		return nil
	})

	return err
}

func (rm *RuntimeMetrics) SendToMetricsStorage(client *http.Client) error {
	var metricsBatch []models.Metrics

	// collect memory metrics
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

			// add gauge metric to batch
			metricsBatch = append(metricsBatch, models.Metrics{
				ID:    metricName,
				MType: models.Gauge,
				Value: &metricValue,
			})
		}
	}

	// add poll count to batch
	pollCountDelta := int64(rm.PollCount)
	metricsBatch = append(metricsBatch, models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: &pollCountDelta,
	})

	// add random value to batch
	metricsBatch = append(metricsBatch, models.Metrics{
		ID:    "RandomValue",
		MType: models.Gauge,
		Value: &rm.RandomValue,
	})

	// send all metrics in one batch (baseURL and hashKey передаются извне при вызове SendToMetricsStorage)
	return sendMetricsBatch(client, "", "", nil, localip.Host(), metricsBatch)
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
