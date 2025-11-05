package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"go.uber.org/fx"
)

type MemoryMetricsStorage struct {
	mu   sync.Mutex
	data map[string]*models.Metrics
}

func (ms *MemoryMetricsStorage) Lock() {
	ms.mu.Lock()
}

func (ms *MemoryMetricsStorage) Unlock() {
	ms.mu.Unlock()
}

func (ms *MemoryMetricsStorage) Reconcile(duration time.Duration, fileStoragePath string) {
	for {
		time.Sleep(duration * time.Second)
		err := ms.Save(fileStoragePath)
		if err != nil {
			panic("unable to save")
		}
	}
}

func (ms *MemoryMetricsStorage) Load(fileStoragePath string) error {
	ms.Lock()
	defer ms.Unlock()

	var metrics []models.Metrics

	data, err := os.ReadFile(fileStoragePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &metrics)
	if err != nil {
		return err
	}

	ms.Make()

	for _, metric := range metrics {
		ms.data[metric.ID] = &metric
	}

	return nil
}

func (ms *MemoryMetricsStorage) Save(fileStoragePath string) error {
	ms.Lock()
	defer ms.Unlock()

	var metrics []models.Metrics

	for _, metric := range ms.data {
		metrics = append(metrics, *metric)
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fileStoragePath, data, 0666)
}

func (ms *MemoryMetricsStorage) Make() {
	ms.data = make(map[string]*models.Metrics)
}

func (ms *MemoryMetricsStorage) Get(k string) (*models.Metrics, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	value, exists := ms.data[k]
	if !exists {
		return nil, errors.New("metric not exists")
	} else {
		return value, nil
	}
}

func (ms *MemoryMetricsStorage) Set(m *models.Metrics) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if m.MType == models.Counter && ms.data[m.ID] != nil {
		if ms.data[m.ID].Delta == nil {
			return errors.New("corrupted database")
		}

		if m.Delta == nil {
			return errors.New("delta cannot be nil")
		}

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
