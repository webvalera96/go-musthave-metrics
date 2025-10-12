package repository

import (
	"errors"
	"sync"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
)

type MemoryMetricsStorage struct {
	data map[string](*models.Metrics)
}

func (ms *MemoryMetricsStorage) Get(k string) (*models.Metrics, error) {
	value, exists := ms.data[k]
	if !exists {
		return nil, errors.New("metric not exists")
	} else {
		return value, nil
	}
}

func (ms *MemoryMetricsStorage) Set(m *models.Metrics) error {
	// if m.MType == models.Counter && ms.data[m.ID] != nil {
	// 	newDelta := *(ms.data[m.ID].Delta) + *(m.Delta)
	// 	ms.data[m.ID].Delta = &newDelta
	// }
	// if m.MType == models.Counter && ms.data[m.ID] != nil {
	// 	newDelta := *(ms.data[m.ID].Delta) + *(m.Delta)
	// 	ms.data[m.ID].Delta = &newDelta
	// }
	ms.data[m.ID] = m
	return nil
}

type MetricsStorage interface {
	Get(k string) *models.Metrics
	Add(k string, m *models.Metrics) error
}

var (
	storage *MemoryMetricsStorage
	once    sync.Once
)

func GetInstance() *MemoryMetricsStorage {
	once.Do(func() {
		storage = &MemoryMetricsStorage{
			data: make(map[string](*models.Metrics)),
		}
	})
	return storage
}
