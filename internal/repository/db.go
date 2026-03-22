package repository

import (
	"context"
	"database/sql"
	"time"

	models "github.com/webvalera96/go-musthave-metrics/internal/model"
	"github.com/webvalera96/go-musthave-metrics/internal/retry"
)

// LoadMetricsFromDB читает все метрики из таблицы metrics.
func LoadMetricsFromDB(db *sql.DB, timeout time.Duration) ([]models.Metrics, error) {
	var metrics []models.Metrics

	err := retry.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		query := "SELECT id, type, delta, value, hash FROM metrics"
		rows, err := db.QueryContext(ctx, query)

		if err != nil {
			return err
		}
		defer rows.Close()

		if rows.Err() != nil {
			return rows.Err()
		}

		metrics = []models.Metrics{}

		for rows.Next() {
			var metric models.Metrics
			var delta sql.NullInt64
			var value sql.NullFloat64
			var hash sql.NullString

			err := rows.Scan(&metric.ID, &metric.MType, &delta, &value, &hash)
			if err != nil {
				return err
			}

			if delta.Valid {
				metric.Delta = &delta.Int64
			}
			if value.Valid {
				metric.Value = &value.Float64
			}
			if hash.Valid {
				metric.Hash = hash.String
			}

			metrics = append(metrics, metric)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return metrics, nil
}

// UpsertMetricInDB вставляет или обновляет одну метрику в БД.
func UpsertMetricInDB(db *sql.DB, m *models.Metrics, timeout time.Duration) error {
	return retry.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		query := "SELECT EXISTS(SELECT 1 FROM metrics WHERE id = $1)"
		exists := false
		err := db.QueryRowContext(ctx, query, m.ID).Scan(&exists)
		if err != nil {
			return err
		}

		if !exists {
			query = "INSERT INTO metrics (id, type, delta, value, hash) VALUES ($1, $2, $3, $4, $5)"
			_, err := db.ExecContext(ctx, query, m.ID, m.MType, m.Delta, m.Value, m.Hash)
			return err
		}

		query = "UPDATE metrics SET type = $1, delta = $2, value = $3, hash = $4 WHERE id = $5"
		_, err = db.ExecContext(ctx, query, m.MType, m.Delta, m.Value, m.Hash, m.ID)
		return err
	})
}
