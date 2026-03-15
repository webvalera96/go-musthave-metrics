package handler

import (
	"database/sql"
	"net/http"

	"github.com/webvalera96/go-musthave-metrics/internal/retry"
)

// PingHandler handles GET /ping and checks DB connectivity.
type PingHandler struct {
	db *sql.DB
}

// NewPingHandler creates a handler for /ping.
func NewPingHandler(db *sql.DB) *PingHandler {
	return &PingHandler{db: db}
}

func (h *PingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "Database connection not available", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	err := retry.Retry(func() error {
		return h.db.PingContext(ctx)
	})

	if err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
