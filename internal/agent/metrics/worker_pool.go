package metrics

import (
	"context"
	"crypto/rsa"
	"net/http"
	"sync"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	pb "github.com/webvalera96/go-musthave-metrics/internal/proto/metrics"
	"github.com/webvalera96/go-musthave-metrics/internal/retry"
)

// SetCancelRun сохраняет cancel для контекста воркеров (вызывается из fx OnStart).
func (wp *WorkerPool) SetCancelRun(cancel context.CancelFunc) {
	wp.cancelRun = cancel
}

// Shutdown отменяет контекст воркеров и ждёт завершения (включая отправку оставшихся батчей).
func (wp *WorkerPool) Shutdown() {
	if wp.cancelRun != nil {
		wp.cancelRun()
	}
	wp.wg.Wait()
}

// WorkerPool управляет пулом воркеров для отправки метрик
type WorkerPool struct {
	client      *http.Client
	baseURL     string
	hashKey     string
	publicKey   *rsa.PublicKey
	grpcClient  pb.MetricsClient
	hostIP      string
	workers     int
	metricsChan <-chan []models.Metrics
	cancelRun   context.CancelFunc
	wg          sync.WaitGroup
}

// NewWorkerPool создает новый пул воркеров. hostIP — для заголовка X-Real-IP / метаданных x-real-ip.
// Если grpcClient != nil, метрики отправляются по gRPC (батч UpdateMetricsRequest); иначе — HTTP /updates/.
func NewWorkerPool(client *http.Client, baseURL string, hashKey string, publicKey *rsa.PublicKey, grpcClient pb.MetricsClient, hostIP string, workers int, metricsChan <-chan []models.Metrics) *WorkerPool {
	return &WorkerPool{
		client:      client,
		baseURL:     baseURL,
		hashKey:     hashKey,
		publicKey:   publicKey,
		grpcClient:  grpcClient,
		hostIP:      hostIP,
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

// worker обрабатывает метрики из канала
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			wp.drainAndSend()
			return
		case metrics, ok := <-wp.metricsChan:
			if !ok {
				return
			}
			err := wp.sendBatch(metrics)
			if err != nil {
				continue
			}
		}
	}
}

func (wp *WorkerPool) drainAndSend() {
	for {
		select {
		case metrics, ok := <-wp.metricsChan:
			if !ok {
				return
			}
			_ = wp.sendBatch(metrics)
		default:
			return
		}
	}
}

func (wp *WorkerPool) sendBatch(metrics []models.Metrics) error {
	if wp.grpcClient != nil {
		return retry.Retry(func() error {
			return sendMetricsBatchGRPC(context.Background(), wp.grpcClient, wp.hostIP, metrics)
		})
	}
	return retry.Retry(func() error {
		return sendMetricsBatch(wp.client, wp.baseURL, wp.hashKey, wp.publicKey, wp.hostIP, metrics)
	})
}
