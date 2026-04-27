package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/pak/kita-springer-manager/internal/audit"
	"github.com/pak/kita-springer-manager/internal/db"
	"github.com/pak/kita-springer-manager/internal/transit"
)

type Handler struct {
	holder  *db.Holder
	transit *transit.Client
}

func New(holder *db.Holder, tc *transit.Client) *Handler {
	return &Handler{holder: holder, transit: tc}
}

// db returns the live SQL connection. Indirected through the holder so the
// pool can be swapped (e.g. after a backup restore) without restarting.
func (h *Handler) db() *sql.DB {
	return h.holder.DB()
}

// holderRef exposes the holder for endpoints (backup/restore) that need to
// swap the pool. Other handlers should stay on h.db().
func (h *Handler) holderRef() *db.Holder {
	return h.holder
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

// serverError logs the underlying err to the audit trail and returns a
// generic 500 to the client. The actual error string never leaves the
// process — it can leak SQL schema, file paths, or library internals.
func serverError(w http.ResponseWriter, err error) {
	audit.L().Error("server.error", "err", err.Error())
	writeError(w, 500, "Interner Serverfehler")
}

// upstreamError reports a failure of an external dependency (transit API,
// geocoding) to the client with a generic 502 and a short German hint, while
// the technical detail goes only to the audit log.
func upstreamError(w http.ResponseWriter, err error, hint string) {
	audit.L().Error("upstream.error", "hint", hint, "err", err.Error())
	if hint == "" {
		hint = "Externer Dienst nicht erreichbar"
	}
	writeError(w, 502, hint)
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
