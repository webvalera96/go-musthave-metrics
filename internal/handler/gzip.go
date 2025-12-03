package handler

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/webvalera96/go-musthave-metrics/internal/flags"
	"github.com/webvalera96/go-musthave-metrics/internal/hash"
	"github.com/webvalera96/go-musthave-metrics/internal/zip"
)

func GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := w
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			if strings.Contains(r.Header.Get("Accept"), "application/json") || strings.Contains(r.Header.Get("Accept"), "text/html") {
				gzw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
				if err != nil {
					io.WriteString(w, err.Error())
					return
				}
				defer gzw.Close()

				w.Header().Set("Content-Encoding", "gzip")
				writer = zip.GzipWriter{ResponseWriter: w, Writer: gzw}
			}
		}

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzr, err := gzip.NewReader(r.Body)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}
			defer gzr.Close()
			r.Body = io.NopCloser(gzr)
			r.Header.Del("Content-Encoding")
		}

		next.ServeHTTP(writer, r)
	})
}

// GzipHandleWithHash объединяет GzipHandle и HashResponseMiddleware
// для правильной обработки хеша от сжатого тела ответа
func GzipHandleWithHash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Обработка входящего gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		if strings.Contains(contentEncoding, "gzip") {
			gzr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to create gzip reader: %v", err), http.StatusBadRequest)
				return
			}
			defer gzr.Close()
			r.Body = io.NopCloser(gzr)
			r.Header.Del("Content-Encoding")
		}

		// Определяем, нужно ли сжимать ответ
		shouldCompress := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") &&
			(strings.Contains(r.Header.Get("Accept"), "application/json") || strings.Contains(r.Header.Get("Accept"), "text/html"))

		// Если ключ задан, используем специальную обертку для вычисления хеша
		if flags.FlagKey != "" && shouldCompress {
			hashWriter := &hashGzipResponseWriter{
				body:       &bytes.Buffer{},
				statusCode: http.StatusOK,
				headers:    make(http.Header),
			}

			gzw, err := gzip.NewWriterLevel(hashWriter, gzip.BestSpeed)
			if err != nil {
				io.WriteString(w, err.Error())
				return
			}

			// Создаем обертку, которая перехватывает заголовки
			writer := &gzipHashWriter{
				ResponseWriter: w,
				gzipWriter:     gzw,
				hashWriter:     hashWriter,
			}

			next.ServeHTTP(writer, r)

			// Закрываем gzip writer, чтобы завершить сжатие
			gzw.Close()

			// Вычисляем хеш от сжатого тела
			compressedBody := hashWriter.body.Bytes()
			hashValue := hash.CalculateHash(compressedBody, flags.FlagKey)

			// Копируем заголовки из обертки (если они были установлены handler'ом)
			for k, v := range hashWriter.headers {
				w.Header()[k] = v
			}

			// Устанавливаем заголовки сжатия и хеша
			w.Header().Set("Content-Encoding", "gzip")
			if hashValue != "" {
				w.Header().Set("HashSHA256", hashValue)
			}

			// Отправляем ответ (заголовки отправляются здесь)
			statusCode := hashWriter.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			w.WriteHeader(statusCode)
			w.Write(compressedBody)
		} else if flags.FlagKey != "" {
			// Ключ задан, но сжатие не применяется
			hashWriter := &hashResponseWriter{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(hashWriter, r)

			// Вычисляем хеш от тела ответа
			responseBody := hashWriter.body.Bytes()
			hashValue := hash.CalculateHash(responseBody, flags.FlagKey)
			if hashValue != "" {
				w.Header().Set("HashSHA256", hashValue)
			}

			if hashWriter.statusCode != 0 {
				w.WriteHeader(hashWriter.statusCode)
			}
			w.Write(responseBody)
		} else {
			// Ключ не задан, обычная обработка
			writer := w
			if shouldCompress {
				gzw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
				if err != nil {
					io.WriteString(w, err.Error())
					return
				}
				defer gzw.Close()

				w.Header().Set("Content-Encoding", "gzip")
				writer = zip.GzipWriter{ResponseWriter: w, Writer: gzw}
			}

			next.ServeHTTP(writer, r)
		}
	})
}

type hashGzipResponseWriter struct {
	body       *bytes.Buffer
	statusCode int
	headers    http.Header
}

func (hw *hashGzipResponseWriter) Write(b []byte) (int, error) {
	// Сохраняем сжатые данные для вычисления хеша
	hw.body.Write(b)
	return len(b), nil
}

func (hw *hashGzipResponseWriter) WriteHeader(statusCode int) {
	hw.statusCode = statusCode
}

type gzipHashWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
	hashWriter *hashGzipResponseWriter
	headerWritten bool
}

func (gw *gzipHashWriter) Write(b []byte) (int, error) {
	// Если WriteHeader еще не был вызван, вызываем его с кодом 200
	if !gw.headerWritten {
		gw.WriteHeader(http.StatusOK)
	}
	// Записываем в gzip writer, который сжимает и пишет в hashWriter
	return gw.gzipWriter.Write(b)
}

func (gw *gzipHashWriter) WriteHeader(statusCode int) {
	// Сохраняем заголовки и статус код, но не отправляем их сразу
	// Копируем заголовки из ResponseWriter в hashWriter
	for k, v := range gw.ResponseWriter.Header() {
		gw.hashWriter.headers[k] = make([]string, len(v))
		copy(gw.hashWriter.headers[k], v)
	}
	gw.hashWriter.statusCode = statusCode
	gw.headerWritten = true
}

func (gw *gzipHashWriter) Header() http.Header {
	return gw.ResponseWriter.Header()
}
