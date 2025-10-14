package handler

import (
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

type UpdateHandler struct{}

func NewUpdateHandler() *UpdateHandler {
	return &UpdateHandler{}
}

func (*UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s := repository.GetInstance()

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

	if strings.ToLower(metricType) == models.Counter {
		cv, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Wrong counter metric value", http.StatusBadRequest)
			return
		}

		err = s.Set(&models.Metrics{
			ID:    metricName,
			MType: models.Counter,
			Delta: &cv,
		})

		if err != nil {
			http.Error(w, "Unable to save counter metric or delta", http.StatusServiceUnavailable)
			return
		}

	} else if strings.ToLower(metricType) == models.Gauge {
		gv, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Wrong gauge metric value", http.StatusBadRequest)
			return
		}

		err = s.Set(&models.Metrics{
			ID:    metricName,
			MType: models.Gauge,
			Value: &gv,
		})

		if err != nil {
			http.Error(w, "Unable to save gauge metric", http.StatusServiceUnavailable)
			return
		}
	} else {
		http.Error(w, "Wrong metrics type", http.StatusBadRequest)
		return
	}
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}
