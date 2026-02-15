package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/webvalera96/go-musthave-metrics/internal/hash"
)

// HashVerifyMiddleware проверяет хеш входящих запросов.
// Должен применяться до gzip middleware, чтобы проверять хеш от сжатого тела.
// hashKey — ключ для проверки подписи (если пустой, проверка не выполняется).
func HashVerifyMiddleware(hashKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hashKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		receivedHash := r.Header.Get("HashSHA256")
		if receivedHash != "" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Unable to read request body", http.StatusBadRequest)
				return
			}
			if r.Body != nil {
				r.Body.Close()
			}

			calculatedHash := hash.CalculateHash(body, hashKey)
			if receivedHash != calculatedHash {
				http.Error(w, "Hash mismatch", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		next.ServeHTTP(w, r)
	})
}

// HashResponseMiddleware добавляет хеш в исходящие ответы (без сжатия).
// hashKey — ключ для подписи (если пустой, хеш не добавляется).
func HashResponseMiddleware(next http.Handler, hashKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hashKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		hashWriter := &hashResponseWriter{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(hashWriter, r)

		responseBody := hashWriter.body.Bytes()
		hashValue := hash.CalculateHash(responseBody, hashKey)
		if hashValue != "" {
			w.Header().Set("HashSHA256", hashValue)
		}
		if hashWriter.statusCode != 0 {
			w.WriteHeader(hashWriter.statusCode)
		}
		w.Write(responseBody)
	})
}

type hashResponseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (hw *hashResponseWriter) Write(b []byte) (int, error) {
	// Сохраняем тело ответа для вычисления хеша
	// Не записываем сразу, а буферизуем
	hw.body.Write(b)
	return len(b), nil
}

func (hw *hashResponseWriter) WriteHeader(statusCode int) {
	// Сохраняем статус код, но не вызываем ResponseWriter.WriteHeader
	// чтобы можно было добавить хеш в заголовки перед отправкой
	hw.statusCode = statusCode
}

