package handler

import (
	"database/sql"
	"net/http"
)

type PingHandler struct {
	db *sql.DB
}

func NewPingHandler(db *sql.DB) *PingHandler {
	return &PingHandler{db: db}
}

func (h *PingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "Database connection not available", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	if err := h.db.PingContext(ctx); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
