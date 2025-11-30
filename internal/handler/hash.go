package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/webvalera96/go-musthave-metrics/internal/flags"
	"github.com/webvalera96/go-musthave-metrics/internal/hash"
)

// HashVerifyMiddleware проверяет хеш входящих запросов
// Должен применяться ДО gzip middleware, чтобы проверять хеш от сжатого тела
func HashVerifyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если ключ не задан, пропускаем проверку
		if flags.FlagKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Получаем хеш из заголовка
		receivedHash := r.Header.Get("HashSHA256")

		// Если хеш передан, проверяем его
		if receivedHash != "" {
			// Читаем тело запроса (на этом этапе оно еще сжато, если есть Content-Encoding: gzip)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Unable to read request body", http.StatusBadRequest)
				return
			}
			if r.Body != nil {
				r.Body.Close()
			}

			// Вычисляем хеш от тела запроса (сжатого, если есть gzip)
			// Хеш должен быть вычислен от того же тела, что отправил агент (сжатого)
			calculatedHash := hash.CalculateHash(body, flags.FlagKey)

			if receivedHash != calculatedHash {
				http.Error(w, "Hash mismatch", http.StatusBadRequest)
				return
			}

			// Восстанавливаем тело для последующего чтения gzip middleware
			// Важно: восстанавливаем именно сжатое тело, чтобы gzip middleware мог его распаковать
			// Создаем новый буфер с исходными данными (сжатыми)
			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		next.ServeHTTP(w, r)
	})
}

// HashResponseMiddleware добавляет хеш в исходящие ответы
// Используется только когда gzip не применяется (иначе используется GzipHandleWithHash)
func HashResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если ключ не задан, пропускаем добавление хеша
		if flags.FlagKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Создаем обертку для перехвата ответа
		hashWriter := &hashResponseWriter{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(hashWriter, r)

		// Вычисляем хеш от тела ответа
		responseBody := hashWriter.body.Bytes()
		hashValue := hash.CalculateHash(responseBody, flags.FlagKey)

		// Добавляем хеш в заголовок
		if hashValue != "" {
			w.Header().Set("HashSHA256", hashValue)
		}

		// Записываем заголовки и тело ответа
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

