package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"go.uber.org/fx"
)

// MemoryMetricsStorage stores metrics in memory; can persist to file or DB.
type MemoryMetricsStorage struct {
	mu              sync.Mutex
	data            map[string]*models.Metrics
	db              *sql.DB
	fileStoragePath string
	timeout         time.Duration
	syncSave        bool
}

func (ms *MemoryMetricsStorage) Lock() {
	ms.mu.Lock()
}

func (ms *MemoryMetricsStorage) Unlock() {
	ms.mu.Unlock()
}

// runPeriodicTask выполняет задачу периодически с использованием ticker
// Останавливается при отмене контекста
func runPeriodicTask(ctx context.Context, duration time.Duration, task func() error, errorMsg string) {
	if duration == 0 {
		return
	}
	ticker := time.NewTicker(duration * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := task()
			if err != nil {
				log.Printf("%s: %v", errorMsg, err)
				// Продолжаем работу, не паникуем
			}
		}
	}
}

func (ms *MemoryMetricsStorage) Reconcile(ctx context.Context, duration time.Duration, fileStoragePath string) {
	// Если duration == 0, синхронное сохранение уже настроено в Set()
	runPeriodicTask(ctx, duration, func() error {
		return ms.Save(fileStoragePath)
	}, "error saving metrics to file "+fileStoragePath)
}

func (ms *MemoryMetricsStorage) ReconcileDB(ctx context.Context, duration time.Duration,
	db *sql.DB, timeout time.Duration) {
	// Если duration == 0, синхронное сохранение уже настроено в Set()
	runPeriodicTask(ctx, duration, func() error {
		return ms.SaveDB(db, timeout)
	}, "error saving metrics to database")
}

func (ms *MemoryMetricsStorage) Load(fileStoragePath string) error {
	ms.Lock()
	defer ms.Unlock()

	// Проверяем существование файла
	if _, err := os.Stat(fileStoragePath); errors.Is(err, os.ErrNotExist) {
		// Файл не существует - это нормальная ситуация при первом запуске
		// Инициализируем пустое хранилище
		ms.Make()
		return nil
	}

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
		m := metric
		ms.data[m.ID] = &m
	}

	return nil
}

func (ms *MemoryMetricsStorage) LoadDB(db *sql.DB, timeout time.Duration) error {
	ms.Lock()
	defer ms.Unlock()

	metrics, err := models.ReadDB(db, timeout)
	if err != nil {
		return err
	}

	ms.Make()

	for _, metric := range metrics {
		m := metric
		ms.data[m.ID] = &m
	}

	return nil
}

func (ms *MemoryMetricsStorage) SaveDB(db *sql.DB, timeout time.Duration) error {
	ms.Lock()
	defer ms.Unlock()

	for _, metric := range ms.data {
		_, err := metric.SaveDB(db, timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

// FlushPersistence сохраняет все метрики в файл или БД (если настроено). Для graceful shutdown.
func (ms *MemoryMetricsStorage) FlushPersistence() error {
	ms.mu.Lock()
	db := ms.db
	path := ms.fileStoragePath
	timeout := ms.timeout
	ms.mu.Unlock()

	if db != nil {
		return ms.SaveDB(db, timeout)
	}
	if path != "" {
		return ms.Save(path)
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

// Make initializes the in-memory map (idempotent).
func (ms *MemoryMetricsStorage) Make() {
	ms.data = make(map[string]*models.Metrics)
}

// Get returns a metric by ID or error if not found.
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

// Set stores or updates a metric (counter is incremented).
func (ms *MemoryMetricsStorage) Set(m *models.Metrics) error {
	ms.mu.Lock()

	if m.MType == models.Counter && ms.data[m.ID] != nil {
		if ms.data[m.ID].Delta == nil {
			ms.mu.Unlock()
			return errors.New("corrupted database")
		}

		if m.Delta == nil {
			ms.mu.Unlock()
			return errors.New("delta cannot be nil")
		}

		newDelta := *(ms.data[m.ID].Delta) + *(m.Delta)
		ms.data[m.ID].Delta = &newDelta
	} else {
		ms.data[m.ID] = m
	}

	syncSave := ms.syncSave
	db := ms.db
	fileStoragePath := ms.fileStoragePath
	timeout := ms.timeout

	ms.mu.Unlock()

	// Если включено синхронное сохранение, сохраняем сразу после обновления
	if syncSave {
		if db != nil {
			return ms.SaveDB(db, timeout)
		} else if fileStoragePath != "" {
			return ms.Save(fileStoragePath)
		}
	}

	return nil
}

// MetricsStorage is the storage interface for metrics.
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

// SetSyncSaveConfig настраивает параметры для синхронного сохранения
func (ms *MemoryMetricsStorage) SetSyncSaveConfig(db *sql.DB, fileStoragePath string, timeout time.Duration, syncSave bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.db = db
	ms.fileStoragePath = fileStoragePath
	ms.timeout = timeout
	ms.syncSave = syncSave
}
