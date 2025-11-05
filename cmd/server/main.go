package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/webvalera96/go-musthave-metrics/internal/flags"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	"github.com/webvalera96/go-musthave-metrics/internal/handler/log"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {

	flags.ParseFlags()

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
		fx.Invoke(
			Restore,
			func(*http.Server) {},
		),
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
		func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "text/html")
			writer.Write([]byte("<html><body>In which task should i make it ?</body></html>"))
		})

	r.Post(
		"/update/{metricType}/{metricName}/{metricValue}",
		log.WithLogging(updateHandler, sugar).ServeHTTP,
	)

	r.Get(
		"/value/{metricType}/{metricName}",
		log.WithLogging(getHandler, sugar).ServeHTTP,
	)

	r.Post(
		"/update/",
		log.WithLogging(updateJSONHandler, sugar).ServeHTTP,
	)

	r.Post("/value/",
		log.WithLogging(getJSONHandler, sugar).ServeHTTP,
	)

	return r
}

func Restore(lc fx.Lifecycle, ms *repository.MemoryMetricsStorage) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {

			if flags.FlagRestore {
				err := ms.Load(flags.FlagStoragePath)
				if err != nil {
					return err
				}
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})
}

func NewSugaredLogger() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	sugar := *logger.Sugar()

	return &sugar
}

func NewHTTPServer(lc fx.Lifecycle, mux *chi.Mux, ms *repository.MemoryMetricsStorage) *http.Server {
	srv := &http.Server{Addr: flags.FlagRunAddr, Handler: handler.GzipHandle(mux)}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP serve at", srv.Addr)
			go srv.Serve(ln)

			duration := time.Duration(flags.FlagStoreInterval)
			go ms.Reconcile(duration, flags.FlagStoragePath)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
