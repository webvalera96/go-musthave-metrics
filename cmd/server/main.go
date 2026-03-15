package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	_ "net/http/pprof" // регистрация /debug/pprof для профилирования

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq" // PostgresSQL driver
	"github.com/webvalera96/go-musthave-metrics/internal/audit"
	"github.com/webvalera96/go-musthave-metrics/internal/flags"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	"github.com/webvalera96/go-musthave-metrics/internal/handler/log"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const timeout time.Duration = time.Duration(30)

func main() {
	fx.New(
		fx.Provide(
			flags.NewServerConfig,
			repository.CreateMemoryMetricsStorage,
			NewHTTPServer,
			NewSugaredLogger,
			NewChiMux,
			NewDatabase,
			NewAuditSubject,
			handler.NewGetHandler,
			handler.NewUpdateHandler,
			handler.NewGetJSONHandler,
			handler.NewUpdateJSONHandler,
			handler.NewUpdateBatchHandler,
			handler.NewPingHandler,
		),
		fx.Invoke(
			Restore,
			SetupSyncSave,
			StartPprofServer,
			func(*http.Server) {},
		),
	).Run()
}

func NewChiMux(
	updateHandler *handler.UpdateHandler,
	getHandler *handler.GetHandler,
	updateJSONHandler *handler.UpdateJSONHandler,
	getJSONHandler *handler.GetJSONHandler,
	updateBatchHandler *handler.UpdateBatchHandler,
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

	r.Post(
		"/updates/",
		log.WithLogging(updateBatchHandler, sugar).ServeHTTP,
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

func Restore(lc fx.Lifecycle, cfg *flags.ServerConfig, ms *repository.MemoryMetricsStorage, db *sql.DB) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if cfg.Restore {
				if cfg.DatabaseDSN != "" {
					err := ms.LoadDB(db, timeout)
					if err != nil {
						return err
					}
				} else if cfg.StoragePath != "" {
					err := ms.Load(cfg.StoragePath)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if db != nil {
				err := db.Close()
				if err != nil {
					return err
				}
			}
			return nil
		},
	})
}

func NewDatabase(cfg *flags.ServerConfig) (*sql.DB, error) {
	if cfg.DatabaseDSN != "" {
		db, err := sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}

		// Создадим необходимые таблицы в базе данных
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to create postgres driver: %w", err)
		}
		m, err := migrate.NewWithDatabaseInstance(
			"file://migrations",
			"postgres", driver)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to create migrate instance: %w", err)
		}
		err = m.Migrate(1) // or m.Steps(2) if you want to explicitly set the number of migrations to run
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			_ = db.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
		return db, nil
	}

	return nil, nil
}

func NewSugaredLogger() (*zap.SugaredLogger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	sugar := *logger.Sugar()

	return &sugar, nil
}

// NewAuditSubject создаёт субъект аудита с приёмниками по конфигу (файл и/или URL).
func NewAuditSubject(cfg *flags.ServerConfig) *audit.Subject {
	return audit.NewSubjectFromConfig(cfg.AuditFile, cfg.AuditURL)
}

// StartPprofServer запускает HTTP-сервер для pprof на localhost:6060 (heap, goroutine, allocs и т.д.).
func StartPprofServer(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				_ = http.ListenAndServe("localhost:6060", nil)
			}()
			return nil
		},
	})
}

func SetupSyncSave(lc fx.Lifecycle, cfg *flags.ServerConfig, ms *repository.MemoryMetricsStorage, db *sql.DB, srv *http.Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if cfg.StoreInterval == 0 {
				if cfg.DatabaseDSN != "" {
					ms.SetSyncSaveConfig(db, "", timeout, true)
				} else if cfg.StoragePath != "" {
					ms.SetSyncSaveConfig(nil, cfg.StoragePath, timeout, true)
				}
			}
			_ = srv
			return nil
		},
	})
}

func NewHTTPServer(lc fx.Lifecycle, cfg *flags.ServerConfig, mux *chi.Mux, ms *repository.MemoryMetricsStorage, db *sql.DB) *http.Server {
	handlerChain := handler.HashVerifyMiddleware(cfg.Key,
		handler.ResponseEncoding(mux, cfg.Key),
	)
	srv := &http.Server{Addr: cfg.RunAddr, Handler: handlerChain}
	var reconcileCancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP serve at", srv.Addr)
			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					fmt.Printf("HTTP server error: %v\n", err)
				}
			}()

			duration := time.Duration(cfg.StoreInterval)
			if duration > 0 {
				runCtx, cancel := context.WithCancel(context.Background())
				reconcileCancel = cancel
				if cfg.DatabaseDSN != "" {
					go ms.ReconcileDB(runCtx, duration, db, timeout)
				} else if cfg.StoragePath != "" {
					go ms.Reconcile(runCtx, duration, cfg.StoragePath)
				}
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if reconcileCancel != nil {
				reconcileCancel()
			}
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
