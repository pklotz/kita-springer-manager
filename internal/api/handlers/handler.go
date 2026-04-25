package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/pak/kita-springer-manager/internal/audit"
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
	// Echo every server-side error into the audit log — the access middleware
	// records status codes but not the human-readable message, which is what
	// we actually need to debug "delete failed but the row is gone"-type
	// issues. Only logged at WARN to avoid drowning normal traffic.
	audit.L().Warn("http.error", "status", status, "msg", msg)
	writeJSON(w, status, map[string]string{"error": msg})
}

// decodeJSON decodes r.Body into v using a stricter decoder: unknown fields are
// rejected (mass-assignment guard) and any trailing content after the first
// JSON value is rejected too. The body has already been wrapped in
// http.MaxBytesReader by the global middleware, so size is bounded.
func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	// Reject trailing data so clients can't smuggle a second payload.
	if dec.More() {
		return errTrailingJSON
	}
	return nil
}

var errTrailingJSON = jsonErr("unexpected trailing data in request body")

type jsonErr string

func (e jsonErr) Error() string { return string(e) }
