package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
)

var flagRunAddr string

func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()
}

func main() {
	parseFlags()
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	fmt.Println("Running server on", flagRunAddr)
	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{metricValue}", handler.Update)
	r.Get("/value/{metricType}/{metricName}", handler.Get)

	if err := http.ListenAndServe(":8080", r); err != nil {
		return err
	}
	return nil
}
