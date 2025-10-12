package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
)

func Get(w http.ResponseWriter, r *http.Request) {
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

	metricStorage := repository.GetInstance()
	metric, err := metricStorage.Get(metricName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if metricType == models.Counter {
		w.Write([]byte(strconv.FormatInt(*metric.Delta, 10)))
	} else {
		w.Write([]byte(strconv.FormatFloat(*metric.Value, 'f', 3, 64)))
	}

}
