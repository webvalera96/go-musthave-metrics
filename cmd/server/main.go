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
		flag.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")
		flag.Parse()
	}
}

func main() {

	parseFlags()

	fx.New(
		fx.Provide(
			repository.CreateMemoryMetricsStorage,
			NewHTTPServer,
			NewSugaredLogger,
			NewChiMux,
			handler.NewGetHandler,
			handler.NewUpdateHandler,
			handler.NewGetJSONHandler,
			handler.NewUpdateJSONHandler,
		),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

func NewChiMux(
	updateHandler *handler.UpdateHandler,
	getHandler *handler.GetHandler,
	updateJSONHandler *handler.UpdateJSONHandler,
	getJSONHandler *handler.GetJSONHandler,
	sugar *zap.SugaredLogger,
) *chi.Mux {
	r := chi.NewRouter()

	r.Get(
		"/",
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "text/html")
			writer.Write([]byte("<html><body>In which task should i make it ?</body></html>"))
		}))

	r.Post(
		"/update/{metricType}/{metricName}/{metricValue}",
		http.HandlerFunc(log.WithLogging(updateHandler, sugar).ServeHTTP),
	)

	r.Get(
		"/value/{metricType}/{metricName}",
		http.HandlerFunc(log.WithLogging(getHandler, sugar).ServeHTTP),
	)

	r.Post(
		"/update/",
		http.HandlerFunc(log.WithLogging(updateJSONHandler, sugar).ServeHTTP),
	)

	r.Post("/value/",
		http.HandlerFunc(log.WithLogging(getJSONHandler, sugar).ServeHTTP),
	)

	return r
}

func NewSugaredLogger() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	sugar := *logger.Sugar()

	return &sugar
}

func NewHTTPServer(lc fx.Lifecycle, mux *chi.Mux) *http.Server {
	srv := &http.Server{Addr: flagRunAddr, Handler: handler.GzipHandle(mux)}
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
