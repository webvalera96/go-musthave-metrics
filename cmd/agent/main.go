package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"time"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
)

const pollInterval = 2
const reportInterval = 10
const baseURL = "localhost:8080" //TODO: move to configuration of agent

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
			err := sendMetric(client, baseURL, models.Gauge, metricName, strconv.FormatFloat(metricValue, 'f', 6, 64))
			if err != nil {
				return err
			}
		}
	}
	// send poll counts
	err := sendMetric(client, baseURL, models.Counter, "PollCount", strconv.FormatUint(rm.PollCount, 10))
	if err != nil {
		return err
	}

	// send random value
	err = sendMetric(client, baseURL, models.Gauge, "RandomValue", strconv.FormatFloat(rm.RandomValue, 'f', 6, 64))
	if err != nil {
		return err
	}
	return nil
}

func main() {

	var wg sync.WaitGroup

	wg.Add(2)

	var runtimeMetrics RuntimeMetrics
	client := http.Client{}

	// thread to update metrics
	go func(wg *sync.WaitGroup, runtimeMetrics *RuntimeMetrics) {
		defer wg.Done()
		for {
			// Pause between metrics gathering
			time.Sleep(time.Second * pollInterval)

			// update current runtimeMetrics
			var rm runtime.MemStats
			runtime.ReadMemStats(&rm)
			runtimeMetrics.Set(rm)
		}
	}(&wg, &runtimeMetrics)

	// thread to periodically send metrics on server
	go func(wg *sync.WaitGroup, runtimeMetrics *RuntimeMetrics, client *http.Client) {
		defer wg.Done()
		for {
			// Pause between metrics sending
			time.Sleep(time.Second * reportInterval)
			runtimeMetrics.SendToMetricsStorage(client)
		}
	}(&wg, &runtimeMetrics, &client)

	wg.Wait()

}
