package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/webvalera96/go-musthave-metrics/internal/audit"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
)

func BenchmarkUpdateHandler_ServeHTTP(b *testing.B) {
	var storage repository.MemoryMetricsStorage
	storage.Make()
	auditSubject := audit.NewSubject()

	h := NewUpdateHandler(&storage, auditSubject)
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/HeapObjects/3522.0", nil)
	req = AddChiURLParams(req, map[string]string{
		"metricType": "gauge", "metricName": "HeapObjects", "metricValue": "3522.0",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateJSONHandler_ServeHTTP(b *testing.B) {
	var storage repository.MemoryMetricsStorage
	storage.Make()
	auditSubject := audit.NewSubject()

	body := models.Metrics{ID: "Alloc", MType: models.Gauge, Value: ptrFloat(1024.0)}
	jsonBody, _ := json.Marshal(body)

	h := NewUpdateJSONHandler(&storage, auditSubject)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateBatchHandler_ServeHTTP(b *testing.B) {
	var storage repository.MemoryMetricsStorage
	storage.Make()
	auditSubject := audit.NewSubject()

	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: ptrFloat(1024.0)},
		{ID: "Frees", MType: models.Counter, Delta: ptrInt64(1)},
		{ID: "HeapAlloc", MType: models.Gauge, Value: ptrFloat(2048.0)},
	}
	jsonBody, _ := json.Marshal(metrics)

	h := NewUpdateBatchHandler(&storage, auditSubject)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func ptrFloat(f float64) *float64 { return &f }
func ptrInt64(n int64) *int64    { return &n }
