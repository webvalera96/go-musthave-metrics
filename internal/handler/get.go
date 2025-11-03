package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
)

type GetHandler struct {
	metricStorage *repository.MemoryMetricsStorage
}

type GetJSONHandler struct {
	metricStorage *repository.MemoryMetricsStorage
}

func NewGetJSONHandler(ms *repository.MemoryMetricsStorage) *GetJSONHandler {
	return &GetJSONHandler{metricStorage: ms}
}

func NewGetHandler(ms *repository.MemoryMetricsStorage) *GetHandler {
	return &GetHandler{metricStorage: ms}
}

func (gh *GetJSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "wrong content type", http.StatusBadRequest)
		return
	}
	var data models.Metrics

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
	}

	err = getMetric(data.MType, data.ID, gh.metricStorage, w)
	if err != nil {
		log.Print(err)
	}
}

func (gh *GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "metricType")
	if metricType == "" {
		http.Error(w, "No metric type", http.StatusBadRequest)
		return
	}

	metricName := chi.URLParam(r, "metricName")
	if metricName == "" {
		http.Error(w, "No metric name", http.StatusBadRequest)
		return
	}

	err := getMetric(metricType, metricName, gh.metricStorage, w)
	if err != nil {
		log.Print(err)
	}

}

func getMetric(
	metricType string,
	metricName string,
	metricStorage *repository.MemoryMetricsStorage,
	w http.ResponseWriter) error {

	metric, err := metricStorage.Get(metricName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return err
	}

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if metricType == models.Counter {
		_, err = w.Write([]byte(strconv.FormatInt(*metric.Delta, 10)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

	} else {
		s := strconv.FormatFloat(*metric.Value, 'f', 3, 64)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		_, err = w.Write([]byte(s))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
	}
	return nil
}
