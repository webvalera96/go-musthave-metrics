package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/webvalera96/go-musthave-metrics/internal/audit"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
)

func addChiParams(r *http.Request, params map[string]string) *http.Request {
	ctx := chi.NewRouteContext()
	for k, v := range params {
		ctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}

// ExampleUpdateHandler_updateGauge demonstrates POST /update/{type}/{name}/{value}.
func ExampleUpdateHandler_updateGauge() {
	var storage repository.MemoryMetricsStorage
	storage.Make()
	uh := handler.NewUpdateHandler(&storage, audit.NewSubject())

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/HeapAlloc/1024", nil)
	req = addChiParams(req, map[string]string{
		"metricType": "gauge", "metricName": "HeapAlloc", "metricValue": "1024",
	})
	w := httptest.NewRecorder()
	uh.ServeHTTP(w, req)

	// Output:
}

// ExampleUpdateJSONHandler_update demonstrates POST /update/ with JSON body.
func ExampleUpdateJSONHandler_update() {
	var storage repository.MemoryMetricsStorage
	storage.Make()
	v := 123.45
	body, _ := json.Marshal(models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &v})

	uh := handler.NewUpdateJSONHandler(&storage, audit.NewSubject())
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	uh.ServeHTTP(w, req)

	// Output:
}

// ExampleUpdateBatchHandler_updates demonstrates POST /updates/ with JSON array.
func ExampleUpdateBatchHandler_updates() {
	var storage repository.MemoryMetricsStorage
	storage.Make()
	delta := int64(1)
	val := 1.0
	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &val},
		{ID: "PollCount", MType: models.Counter, Delta: &delta},
	}
	body, _ := json.Marshal(metrics)

	uh := handler.NewUpdateBatchHandler(&storage, audit.NewSubject())
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	uh.ServeHTTP(w, req)

	// Output:
}

// ExampleGetHandler_value demonstrates GET /value/{type}/{name}.
func ExampleGetHandler_value() {
	var storage repository.MemoryMetricsStorage
	storage.Make()
	val := 1024.0
	_ = storage.Set(&models.Metrics{ID: "HeapAlloc", MType: models.Gauge, Value: &val})

	gh := handler.NewGetHandler(&storage)
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/HeapAlloc", nil)
	req = addChiParams(req, map[string]string{"metricType": "gauge", "metricName": "HeapAlloc"})
	w := httptest.NewRecorder()
	gh.ServeHTTP(w, req)

	// Output:
}

// ExampleGetJSONHandler_value demonstrates POST /value/ with JSON body.
func ExampleGetJSONHandler_value() {
	var storage repository.MemoryMetricsStorage
	storage.Make()
	val := 2048.0
	_ = storage.Set(&models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &val})

	body, _ := json.Marshal(models.Metrics{ID: "Alloc"})
	gh := handler.NewGetJSONHandler(&storage)
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	gh.ServeHTTP(w, req)

	// Output:
}
