package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq" // PostgresSQL driver
	"github.com/webvalera96/go-musthave-metrics/internal/flags"
	"github.com/webvalera96/go-musthave-metrics/internal/handler"
	"github.com/webvalera96/go-musthave-metrics/internal/handler/log"
	"github.com/webvalera96/go-musthave-metrics/internal/repository"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const timeout time.Duration = time.Duration(30)

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
			handler.NewUpdateBatchHandler,
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

func Restore(lc fx.Lifecycle, ms *repository.MemoryMetricsStorage, db *sql.DB) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {

			if flags.FlagRestore {
				// Приоритет: DATABASE_DSN > FILE_STORAGE_PATH > память
				if flags.FlagDatabaseDSN != "" {
					err := ms.LoadDB(db, timeout)
					if err != nil {
						return err
					}
				} else if flags.FlagStoragePath != "" {
					err := ms.Load(flags.FlagStoragePath)
					if err != nil {
						return err
					}
				}
				// Если оба пустые - используем память, ничего не загружаем
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

func NewDatabase() (*sql.DB, error) {
	if flags.FlagDatabaseDSN != "" {
		db, err := sql.Open("postgres", flags.FlagDatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}

		// Создадим необходимые таблицы в базе данных
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create postgres driver: %w", err)
		}
		m, err := migrate.NewWithDatabaseInstance(
			"file://migrations",
			"postgres", driver)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create migrate instance: %w", err)
		}
		err = m.Migrate(1) // or m.Steps(2) if you want to explicitly set the number of migrations to run
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			db.Close()
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

func NewHTTPServer(lc fx.Lifecycle, mux *chi.Mux, ms *repository.MemoryMetricsStorage, db *sql.DB) *http.Server {
	// Порядок middleware важен:
	// 1. HashVerifyMiddleware - проверяет хеш от сжатого тела запроса (до gzip распаковки)
	// 2. GzipHandle - распаковывает gzip в запросах и сжимает ответы
	// 3. HashResponseMiddleware - добавляет хеш в ответы (от сжатого тела, если gzip применен)
	// HashResponseMiddleware должен быть внутри GzipHandle, чтобы перехватывать сжатый вывод
	handlerChain := handler.HashVerifyMiddleware(
		handler.GzipHandleWithHash(mux),
	)
	srv := &http.Server{Addr: flags.FlagRunAddr, Handler: handlerChain}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP serve at", srv.Addr)
			go srv.Serve(ln)

			// Приоритет: DATABASE_DSN > FILE_STORAGE_PATH > память
			duration := time.Duration(flags.FlagStoreInterval)
			if flags.FlagDatabaseDSN != "" {
				// Используем БД
				go ms.ReconcileDB(duration, db, timeout)
			} else if flags.FlagStoragePath != "" {
				// Используем файл
				go ms.Reconcile(duration, flags.FlagStoragePath)
			}
			// Если оба пустые - используем память, периодическое сохранение не запускаем

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}
