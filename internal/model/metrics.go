package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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

func ReadDB(pCtx context.Context, db *sql.DB) ([]Metrics, error) {
	ctx, cancel := context.WithTimeout(pCtx, 10*time.Second)
	defer cancel()

	var metrics []Metrics

	query := fmt.Sprintf("SELECT * FROM metrics")
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var metric Metrics
		err := rows.Scan(&metric)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func (m Metrics) SaveDB(pCtx context.Context, db *sql.DB) (string, error) {
	// Проверяем, есть ли в базе данных такая запись
	ctx, cancel := context.WithTimeout(pCtx, 10*time.Second)
	defer cancel()

	query := fmt.Sprintf("EXISTS(SELECT * FROM metrics WHERE id == %s)", m.ID)
	exists := false

	err := db.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil {

		// Запись не существует, тогда добавляем запись в базу данных
		query = fmt.Sprintf("INSERT INTO metrics (id, type, delta, value, hash) VALUES (%s, %s, %d, %d, %s)", m.ID, m.MType, m.Delta, m.Value, m.Hash)

		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return "", err
		}

	} else {
		// Запись существует, тогда ее надо обновить
		query = fmt.Sprintf("UPDATE metrics SET id = %s, type = %s, delta = %d, value = %d, hash = %s", m.ID, m.MType, m.Delta, m.Value, m.Hash)
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return "", err
		}
	}

	return m.ID, nil

}
