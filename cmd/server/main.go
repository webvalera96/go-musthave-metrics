package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
)

func main() {
	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{metricValue}", handler.Update)
	r.Get("/value/{metricType}/{metricName}", handler.Get)

	if err := http.ListenAndServe(":8080", r); err != nil {
		panic(err)
	}
}
