package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
)

var flagRunAddr string

const (
	EnvAddress = "ADDRESS"
)

func parseFlags() {
	var exist bool
	flagRunAddr, exist = os.LookupEnv(EnvAddress)
	if !exist {
		flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
		flag.Parse()
	}
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

	if err := http.ListenAndServe(flagRunAddr, r); err != nil {
		return err
	}
	return nil
}
