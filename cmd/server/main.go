package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	"github.com/webvalera96/go-musthave-metrics/internal/handler/log"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

	fx.New(
		fx.Provide(
			repository.CreateMemoryMetricsStorage,
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
	r.Post("/update/{metricType}/{metricName}/{metricValue}", http.HandlerFunc(log.WithLogging(updateHandler, zap.SugaredLogger{}).ServeHTTP))
	r.Get("/value/{metricType}/{metricName}", http.HandlerFunc(log.WithLogging(getHandler, zap.SugaredLogger{}).ServeHTTP))
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
			go func() {
				err := srv.Serve(ln)
				if err != nil {
					panic(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
