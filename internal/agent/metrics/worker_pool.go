package metrics

import (
	"context"
	"net/http"
	"sync"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
)

// WorkerPool управляет пулом воркеров для отправки метрик
type WorkerPool struct {
	client      *http.Client
	baseURL     string
	workers     int
	metricsChan <-chan []models.Metrics
	wg          sync.WaitGroup
}

// NewWorkerPool создает новый пул воркеров
func NewWorkerPool(client *http.Client, baseURL string, workers int, metricsChan <-chan []models.Metrics) *WorkerPool {
	return &WorkerPool{
		client:      client,
		baseURL:     baseURL,
		workers:     workers,
		metricsChan: metricsChan,
	}
}

// Start запускает пул воркеров
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
}

// Stop останавливает пул воркеров
func (wp *WorkerPool) Stop() {
	wp.wg.Wait()
}

// worker обрабатывает метрики из канала
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case metrics, ok := <-wp.metricsChan:
			if !ok {
				return
			}
			// Отправляем метрики
			err := sendMetricsBatch(wp.client, wp.baseURL, metrics)
			if err != nil {
				// Логируем ошибку, но продолжаем работу
				continue
			}
		}
	}
}

