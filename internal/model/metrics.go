package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/webvalera96/go-musthave-metrics/internal/retry"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func ReadDB(db *sql.DB, timeout time.Duration) ([]Metrics, error) {
	var metrics []Metrics

	err := retry.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
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

		// Очищаем предыдущие результаты перед повторной попыткой
		metrics = []Metrics{}

		for rows.Next() {
			var metric Metrics
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

func (m Metrics) SaveDB(db *sql.DB, timeout time.Duration) (string, error) {
	var result string

	err := retry.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
		defer cancel()

		// Проверяем, есть ли в базе данных такая запись
		query := "SELECT EXISTS(SELECT 1 FROM metrics WHERE id = $1)"
		exists := false
		err := db.QueryRowContext(ctx, query, m.ID).Scan(&exists)
		if err != nil {
			return err
		}

		if !exists {
			// Запись не существует, тогда добавляем запись в базу данных
			query = "INSERT INTO metrics (id, type, delta, value, hash) VALUES ($1, $2, $3, $4, $5)"
			_, err := db.ExecContext(ctx, query, m.ID, m.MType, m.Delta, m.Value, m.Hash)
			if err != nil {
				return err
			}
		} else {
			// Запись существует, тогда ее надо обновить
			query = "UPDATE metrics SET type = $1, delta = $2, value = $3, hash = $4 WHERE id = $5"
			_, err := db.ExecContext(ctx, query, m.MType, m.Delta, m.Value, m.Hash, m.ID)
			if err != nil {
				return err
			}
		}

		result = m.ID
		return nil
	})

	if err != nil {
		return "", err
	}

	return result, nil
}
