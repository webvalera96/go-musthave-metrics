package repository

import (
	"context"
	"errors"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"go.uber.org/fx"
)

type MemoryMetricsStorage struct {
	data map[string](*models.Metrics)
}

func (ms *MemoryMetricsStorage) Make() {
	ms.data = make(map[string](*models.Metrics))
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
	if m.MType == models.Counter && ms.data[m.ID] != nil {
		newDelta := *(ms.data[m.ID].Delta) + *(m.Delta)
		ms.data[m.ID].Delta = &newDelta
		return nil
	}

	ms.data[m.ID] = m
	return nil
}

type MetricsStorage interface {
	Get(k string) *models.Metrics
	Add(k string, m *models.Metrics) error
}

func CreateMemoryMetricsStorage(lc fx.Lifecycle) *MemoryMetricsStorage {
	storage := MemoryMetricsStorage{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			storage.Make()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			storage = MemoryMetricsStorage{}
			return nil
		},
	})
	return &storage
}
