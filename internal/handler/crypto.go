package handler

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/webvalera96/go-musthave-metrics/internal/securepayload"
)

// DecryptRequestMiddleware расшифровывает тело запроса до проверки хеша и gzip.
// priv == nil — пропуск без изменений.
func DecryptRequestMiddleware(priv *rsa.PrivateKey, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := r.Header.Get(securepayload.HTTPHeaderEncrypted)
		if enc == "" {
			next.ServeHTTP(w, r)
			return
		}
		if priv == nil {
			http.Error(w, "encrypted request but server has no crypto key", http.StatusBadRequest)
			return
		}

		encBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()

		plain, err := securepayload.Decrypt(priv, encBody)
		if err != nil {
			http.Error(w, "Unable to decrypt request body", http.StatusBadRequest)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(plain))
		r.Header.Del(securepayload.HTTPHeaderEncrypted)
		r.ContentLength = int64(len(plain))

		next.ServeHTTP(w, r)
	})
}
