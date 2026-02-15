package retry

import (
	"errors"
	"net"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
)

// IsRetryableError проверяет, является ли ошибка retriable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Проверяем ошибки сетевого соединения (для HTTP запросов)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Проверяем ошибки PostgreSQL класса 08 (Connection Exception)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// Проверяем через pgerrcode, является ли это ошибкой соединения
		if pgerrcode.IsConnectionException(string(pqErr.Code)) {
			return true
		}
	}

	return false
}

// Retry выполняет функцию с повторами при retriable ошибках
// Интервалы между повторами: 1s, 3s, 5s (3 дополнительные попытки, всего 4 попытки)
func Retry(fn func() error) error {
	intervals := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	maxAttempts := len(intervals) + 1 // 4 попытки всего

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Если ошибка не retriable, не повторяем
		if !IsRetryableError(err) {
			return err
		}

		// Если это не последняя попытка, ждем перед следующим повтором
		if attempt < maxAttempts-1 {
			time.Sleep(intervals[attempt])
		}
	}

	return lastErr
}

