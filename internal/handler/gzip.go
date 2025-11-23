package handler

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

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
