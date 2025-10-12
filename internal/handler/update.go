package handler

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
)

const (
	MetricType  = 0
	MetricName  = 1
	MetricValue = 2
)

func Update(w http.ResponseWriter, r *http.Request) {
	s := repository.GetInstance()
	if r.Method == http.MethodPost {
		u, err := url.Parse(r.URL.Path)
		if err != nil {
			http.Error(w, "Wrong url", http.StatusBadRequest)
		}
		parts := strings.Split(u.Path, "/")[2:]
		if len(parts) == 2 {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		} else if len(parts) != 3 {
			http.Error(w, "Wrong metrics data", http.StatusBadRequest)
			return
		}

		if strings.ToLower(parts[MetricType]) == models.Counter {
			cv, err := strconv.ParseInt(parts[MetricValue], 10, 64)
			if err != nil {
				http.Error(w, "Wrong counter metric value", http.StatusBadRequest)
				return
			}

			err = s.Set(&models.Metrics{
				ID:    parts[MetricName],
				MType: models.Counter,
				Delta: &cv,
			})

			if err != nil {
				http.Error(w, "Unable to save counter metric or delta", http.StatusServiceUnavailable)
				return
			}

		} else if strings.ToLower(parts[MetricType]) == models.Gauge {
			gv, err := strconv.ParseFloat(parts[MetricValue], 64)
			if err != nil {
				http.Error(w, "Wrong gauge metric value", http.StatusBadRequest)
				return
			}

			err = s.Set(&models.Metrics{
				ID:    parts[MetricName],
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
	} else {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}
