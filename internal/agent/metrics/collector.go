package metrics

import (
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"sync"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
)

// MetricsCollector собирает метрики и отправляет их в канал
type MetricsCollector struct {
	metricsChan    chan []models.Metrics
	mu             sync.Mutex
	runtimeData    RuntimeMetrics
	gopsutilMetrics []models.Metrics
}

// NewMetricsCollector создает новый коллектор метрик
func NewMetricsCollector(bufferSize int) *MetricsCollector {
	return &MetricsCollector{
		metricsChan: make(chan []models.Metrics, bufferSize),
	}
}

// GetMetricsChan возвращает канал для получения метрик
func (mc *MetricsCollector) GetMetricsChan() <-chan []models.Metrics {
	return mc.metricsChan
}

// CollectRuntimeMetrics собирает runtime метрики
func (mc *MetricsCollector) CollectRuntimeMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	mc.runtimeData.Set(m)
}

// CollectGopsutilMetrics собирает метрики через gopsutil и сохраняет их
func (mc *MetricsCollector) CollectGopsutilMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var gopsutilMetrics []models.Metrics

	// TotalMemory
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		totalMemory := float64(vmStat.Total)
		gopsutilMetrics = append(gopsutilMetrics, models.Metrics{
			ID:    "TotalMemory",
			MType: models.Gauge,
			Value: &totalMemory,
		})

		// FreeMemory
		freeMemory := float64(vmStat.Free)
		gopsutilMetrics = append(gopsutilMetrics, models.Metrics{
			ID:    "FreeMemory",
			MType: models.Gauge,
			Value: &freeMemory,
		})
	}

	// CPUutilization1 (по количеству CPU)
	cpuCount := runtime.NumCPU()
	cpuPercentages, err := cpu.Percent(0, true) // true = per CPU
	if err == nil {
		for i, cpuPercent := range cpuPercentages {
			if i < cpuCount {
				cpuValue := cpuPercent
				metricName := fmt.Sprintf("CPUutilization%d", i+1)
				gopsutilMetrics = append(gopsutilMetrics, models.Metrics{
					ID:    metricName,
					MType: models.Gauge,
					Value: &cpuValue,
				})
			}
		}
	}

	mc.gopsutilMetrics = gopsutilMetrics
}

// BuildMetricsBatch создает батч метрик для отправки
func (mc *MetricsCollector) BuildMetricsBatch() []models.Metrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var metricsBatch []models.Metrics

	// Собираем runtime метрики
	value := reflect.ValueOf(mc.runtimeData.RuntimeMemoryMetrics)
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
				continue
			}

			metricsBatch = append(metricsBatch, models.Metrics{
				ID:    metricName,
				MType: models.Gauge,
				Value: &metricValue,
			})
		}
	}

	// Добавляем PollCount
	pollCountDelta := int64(mc.runtimeData.PollCount)
	metricsBatch = append(metricsBatch, models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: &pollCountDelta,
	})

	// Добавляем RandomValue
	metricsBatch = append(metricsBatch, models.Metrics{
		ID:    "RandomValue",
		MType: models.Gauge,
		Value: &mc.runtimeData.RandomValue,
	})

	// Добавляем gopsutil метрики (уже собранные)
	metricsBatch = append(metricsBatch, mc.gopsutilMetrics...)

	return metricsBatch
}

// SendMetrics отправляет метрики в канал
func (mc *MetricsCollector) SendMetrics() {
	metricsBatch := mc.BuildMetricsBatch()
	select {
	case mc.metricsChan <- metricsBatch:
	default:
		// Канал переполнен, пропускаем отправку
	}
}

