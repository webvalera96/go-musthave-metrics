package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	"go.uber.org/fx"
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

	fx.New(
		fx.Provide(
			NewHTTPServer,
			NewChiMux,
			handler.NewGetHandler,
			handler.NewUpdateHandler,
		),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

func NewChiMux(updateHandler *handler.UpdateHandler, getHandler *handler.GetHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{metricValue}", http.HandlerFunc(updateHandler.ServeHTTP))
	r.Get("/value/{metricType}/{metricName}", http.HandlerFunc(getHandler.ServeHTTP))
	return r
}

func NewHTTPServer(lc fx.Lifecycle, mux *chi.Mux) *http.Server {
	srv := &http.Server{Addr: flagRunAddr, Handler: mux}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP serve at", srv.Addr)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
