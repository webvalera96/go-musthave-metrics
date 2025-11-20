package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
)

const (
	MetricType  = 0
	MetricName  = 1
	MetricValue = 2
)

type UpdateHandler struct {
	metricStorage *repository.MemoryMetricsStorage
}

type UpdateJSONHandler struct {
	metricStorage *repository.MemoryMetricsStorage
}

type UpdateBatchHandler struct {
	metricStorage *repository.MemoryMetricsStorage
}

func NewUpdateHandler(ms *repository.MemoryMetricsStorage) *UpdateHandler {

	return &UpdateHandler{metricStorage: ms}
}
func NewUpdateJSONHandler(ms *repository.MemoryMetricsStorage) *UpdateJSONHandler {
	return &UpdateJSONHandler{metricStorage: ms}
}

func NewUpdateBatchHandler(ms *repository.MemoryMetricsStorage) *UpdateBatchHandler {
	return &UpdateBatchHandler{metricStorage: ms}
}

func (uh *UpdateJSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "wrong content type", http.StatusBadRequest)
		return
	}

	var data models.Metrics

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
	}

	if data.MType == models.Counter {
		if data.Delta == nil {
			http.Error(w, "Unable to save metric: (delta is empty)", http.StatusServiceUnavailable)
		}
		err = uh.metricStorage.Set(&models.Metrics{
			ID:    data.ID,
			MType: data.MType,
			Delta: data.Delta,
		})
	} else if data.MType == models.Gauge {
		if data.Value == nil {
			http.Error(w, "Unable to save metric: (value is empty)", http.StatusServiceUnavailable)
		}
		err = uh.metricStorage.Set(&models.Metrics{
			ID:    data.ID,
			MType: data.MType,
			Value: data.Value,
		})
	} else {
		err = errors.New("not known type of metrics")
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to save metric: (%s)", err), http.StatusServiceUnavailable)
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(nil)

}

func (uh *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	metricType := chi.URLParam(r, "metricType")
	if metricType == "" {
		http.Error(w, "metric type not specified", http.StatusNotFound)
		return
	}

	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "metric name not specified", http.StatusNotFound)
		return
	}

	metricValue := chi.URLParam(r, "metricValue")
	if metricValue == "" {
		http.Error(w, "metric value not specified", http.StatusBadRequest)
		return
	}

	err := updateMetric(metricName,
		metricType,
		metricValue,
		uh.metricStorage,
		w,
	)
	if err != nil {
		log.Print(err)
	}
}

func updateMetric(
	metricName string,
	metricType string,
	metricValue string,
	metricStorage *repository.MemoryMetricsStorage,
	w http.ResponseWriter,
) error {
	if strings.ToLower(metricType) == models.Counter {
		cv, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Wrong counter metric value", http.StatusBadRequest)
			return err
		}

		err = metricStorage.Set(&models.Metrics{
			ID:    metricName,
			MType: models.Counter,
			Delta: &cv,
		})

		if err != nil {
			http.Error(w, "Unable to save counter metric or delta", http.StatusServiceUnavailable)
			return err
		}

	} else if strings.ToLower(metricType) == models.Gauge {
		gv, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Wrong gauge metric value", http.StatusBadRequest)
			return err
		}

		err = metricStorage.Set(&models.Metrics{
			ID:    metricName,
			MType: models.Gauge,
			Value: &gv,
		})

		if err != nil {
			http.Error(w, "Unable to save gauge metric", http.StatusServiceUnavailable)
			return err
		}
	} else {
		http.Error(w, "Wrong metrics type", http.StatusBadRequest)
		return errors.New("wrong metrics type")
	}
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(nil)
	return nil
}

func (uh *UpdateBatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "wrong content type", http.StatusBadRequest)
		return
	}

	var metrics []models.Metrics

	err := json.NewDecoder(r.Body).Decode(&metrics)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	for _, data := range metrics {
		if data.MType == models.Counter {
			if data.Delta == nil {
				http.Error(w, fmt.Sprintf("Unable to save metric %s: (delta is empty)", data.ID), http.StatusBadRequest)
				return
			}
			err = uh.metricStorage.Set(&models.Metrics{
				ID:    data.ID,
				MType: data.MType,
				Delta: data.Delta,
			})
		} else if data.MType == models.Gauge {
			if data.Value == nil {
				http.Error(w, fmt.Sprintf("Unable to save metric %s: (value is empty)", data.ID), http.StatusBadRequest)
				return
			}
			err = uh.metricStorage.Set(&models.Metrics{
				ID:    data.ID,
				MType: data.MType,
				Value: data.Value,
			})
		} else {
			http.Error(w, fmt.Sprintf("Unknown type of metric: %s for metric %s", data.MType, data.ID), http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, fmt.Sprintf("Unable to save metric %s: (%s)", data.ID, err), http.StatusServiceUnavailable)
			return
		}
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}
