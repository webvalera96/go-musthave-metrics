package handler

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/webvalera96/go-musthave-metrics/internal/hash"
	"github.com/webvalera96/go-musthave-metrics/internal/zip"
)

// DecompressGzipRequest распаковывает тело запроса, если оно в gzip.
func DecompressGzipRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to create gzip reader: %v", err), http.StatusBadRequest)
				return
			}
			defer gzr.Close()
			r.Body = io.NopCloser(gzr)
			r.Header.Del("Content-Encoding")
		}
		next.ServeHTTP(w, r)
	})
}

// CompressResponse сжимает ответ gzip, если клиент поддерживает и тип контента подходит.
func CompressResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := w
		if shouldCompress(r) {
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
	})
}

func shouldCompress(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") &&
		(strings.Contains(r.Header.Get("Accept"), "application/json") || strings.Contains(r.Header.Get("Accept"), "text/html"))
}

// CompressAndHashResponse сжимает ответ и добавляет HashSHA256 от сжатого тела.
func CompressAndHashResponse(next http.Handler, hashKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hashKey == "" {
			CompressResponse(next).ServeHTTP(w, r)
			return
		}

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

		writer := &gzipHashWriter{
			ResponseWriter: w,
			gzipWriter:     gzw,
			hashWriter:     hashWriter,
		}

		next.ServeHTTP(writer, r)
		gzw.Close()

		compressedBody := hashWriter.body.Bytes()
		hashValue := hash.CalculateHash(compressedBody, hashKey)

		for k, v := range hashWriter.headers {
			w.Header()[k] = v
		}
		w.Header().Set("Content-Encoding", "gzip")
		if hashValue != "" {
			w.Header().Set("HashSHA256", hashValue)
		}
		statusCode := hashWriter.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		w.WriteHeader(statusCode)
		w.Write(compressedBody)
	})
}

// ResponseEncoding композиция: распаковка запроса и одна из стратегий ответа
// (сжатие+хеш, только хеш, только сжатие или pass-through) в зависимости от hashKey и Accept-Encoding.
func ResponseEncoding(next http.Handler, hashKey string) http.Handler {
	return DecompressGzipRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		compress := shouldCompress(r)
		switch {
		case hashKey != "" && compress:
			CompressAndHashResponse(next, hashKey).ServeHTTP(w, r)
		case hashKey != "":
			HashResponseMiddleware(next, hashKey).ServeHTTP(w, r)
		case compress:
			CompressResponse(next).ServeHTTP(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	}))
}

// Типы для перехвата сжатого тела при подсчёте хеша
type hashGzipResponseWriter struct {
	body       *bytes.Buffer
	statusCode int
	headers    http.Header
}

func (hw *hashGzipResponseWriter) Write(b []byte) (int, error) {
	hw.body.Write(b)
	return len(b), nil
}

func (hw *hashGzipResponseWriter) WriteHeader(statusCode int) {
	hw.statusCode = statusCode
}

type gzipHashWriter struct {
	http.ResponseWriter
	gzipWriter    *gzip.Writer
	hashWriter    *hashGzipResponseWriter
	headerWritten bool
}

func (gw *gzipHashWriter) Write(b []byte) (int, error) {
	if !gw.headerWritten {
		gw.WriteHeader(http.StatusOK)
	}
	return gw.gzipWriter.Write(b)
}

func (gw *gzipHashWriter) WriteHeader(statusCode int) {
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
