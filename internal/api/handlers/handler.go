package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/pak/kita-springer-manager/internal/transit"
)

type Handler struct {
	db      *sql.DB
	transit *transit.Client
}

func New(db *sql.DB, tc *transit.Client) *Handler {
	return &Handler{db: db, transit: tc}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
