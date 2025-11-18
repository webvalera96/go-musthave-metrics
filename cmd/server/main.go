package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq" // PostgresSQL driver
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
			NewDatabase,
			handler.NewGetHandler,
			handler.NewUpdateHandler,
			handler.NewGetJSONHandler,
			handler.NewUpdateJSONHandler,
			handler.NewPingHandler,
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
	pingHandler *handler.PingHandler,
) *chi.Mux {
	r := chi.NewRouter()

	r.Get(
		"/",
		func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "text/html")
			_, _ = writer.Write([]byte("<html><body>In which task should i make it ?</body></html>"))

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

	r.Get(
		"/ping",
		log.WithLogging(pingHandler, sugar).ServeHTTP,
	)

	return r
}

func Restore(lc fx.Lifecycle, ms *repository.MemoryMetricsStorage, db *sql.DB) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {

			if flags.FlagRestore {

				if flags.FlagStoragePath != "" {
					if _, err := os.Stat(flags.FlagStoragePath); errors.Is(err, os.ErrNotExist) {
						return nil
					}

					err := ms.Load(flags.FlagStoragePath)
					if err != nil {
						return err
					}
				} else if flags.FlagDatabaseDSN != "" {
					err := ms.LoadDB(ctx, db)
					if err != nil {
						return err
					}
				}

			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			err := db.Close()
			if err != nil {
				return err
			}
			return nil
		},
	})
}

func NewDatabase() *sql.DB {

	db, err := sql.Open("postgres", flags.FlagDatabaseDSN)
	if err != nil {
		panic(err)
	}
	return db
}

func NewSugaredLogger() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	sugar := *logger.Sugar()

	return &sugar
}

func NewHTTPServer(lc fx.Lifecycle, mux *chi.Mux, ms *repository.MemoryMetricsStorage, db *sql.DB) *http.Server {
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
			if flags.FlagStoragePath != "" {
				go ms.Reconcile(duration, flags.FlagStoragePath)
			} else if flags.FlagDatabaseDSN != "" {
				go ms.ReconcileDB(ctx, duration, db)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
