package handler

import (
	"net"
	"net/http"
	"strings"
)

// XRealIPHeader — заголовок с IP хоста агента (проверка trusted subnet на сервере).
const XRealIPHeader = "X-Real-IP"

// XRealIPMetadataKey — ключ метаданных gRPC для передачи IP агента (нижний регистр, как принято в gRPC).
const XRealIPMetadataKey = "x-real-ip"

// TrustedSubnetMiddleware отклоняет POST запросы с метриками, если X-Real-IP не входит в подсеть.
// trusted == nil — ограничений нет (пусто в конфиге).
func TrustedSubnetMiddleware(trusted *net.IPNet, next http.Handler) http.Handler {
	if trusted == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAgentMetricsPost(r) {
			next.ServeHTTP(w, r)
			return
		}
		ipStr := strings.TrimSpace(r.Header.Get(XRealIPHeader))
		if ipStr == "" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		ip := net.ParseIP(ipStr)
		if ip == nil || !trusted.Contains(ip) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isAgentMetricsPost(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	p := r.URL.Path
	switch {
	case p == "/updates" || p == "/updates/":
		return true
	case p == "/update" || p == "/update/":
		return true
	default:
		if strings.HasPrefix(p, "/update/") {
			rest := strings.TrimPrefix(p, "/update/")
			parts := strings.Split(rest, "/")
			return len(parts) >= 3
		}
		return false
	}
}
