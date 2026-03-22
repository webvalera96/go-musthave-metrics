package repository

import (
	"testing"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
)

func BenchmarkMemoryMetricsStorage_Set(b *testing.B) {
	var ms MemoryMetricsStorage
	ms.Make()

	value := 1.0
	m := &models.Metrics{ID: "test", MType: models.Gauge, Value: &value}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.ID = "metric_x"
		_ = ms.Set(m)
	}
}

func BenchmarkMemoryMetricsStorage_SetCounter(b *testing.B) {
	var ms MemoryMetricsStorage
	ms.Make()

	delta := int64(1)
	m := &models.Metrics{ID: "counter_metric", MType: models.Counter, Delta: &delta}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ms.Set(m)
	}
}

func BenchmarkMemoryMetricsStorage_Get(b *testing.B) {
	var ms MemoryMetricsStorage
	ms.Make()
	value := 1.0
	_ = ms.Set(&models.Metrics{ID: "key", MType: models.Gauge, Value: &value})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ms.Get("key")
	}
}

func BenchmarkMemoryMetricsStorage_SetParallel(b *testing.B) {
	var ms MemoryMetricsStorage
	ms.Make()
	value := 1.0

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m := &models.Metrics{
				ID:    "metric_parallel",
				MType: models.Gauge,
				Value: &value,
			}
			_ = ms.Set(m)
		}
	})
}
