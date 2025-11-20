package models

import (
	"context"
	"database/sql"
	"fmt"
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
		query := "SELECT * FROM metrics"
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
			err := rows.Scan(&metric)
			if err != nil {
				return err
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
		// Проверяем, есть ли в базе данных такая запись
		query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM metrics WHERE id = '%s')", m.ID)
		exists := false
		ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
		defer cancel()
		err := db.QueryRowContext(ctx, query).Scan(&exists)
		if err != nil {
			return err
		}

		if !exists {
			// Запись не существует, тогда добавляем запись в базу данных
			query = fmt.Sprintf("INSERT INTO metrics (id, type, delta, value, hash) VALUES ('%s', '%s', %d, %d, '%s')", m.ID, m.MType, m.Delta, m.Value, m.Hash)

			_, err := db.ExecContext(ctx, query)
			if err != nil {
				return err
			}
		} else {
			// Запись существует, тогда ее надо обновить
			query = fmt.Sprintf("UPDATE metrics SET type = '%s', delta = %d, value = %d, hash = '%s' WHERE id = '%s'", m.MType, m.Delta, m.Value, m.Hash, m.ID)
			_, err := db.ExecContext(ctx, query)
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
