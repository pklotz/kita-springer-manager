package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
)

// ListRecurring returns all recurring assignment rules.
func (h *Handler) ListRecurring(w http.ResponseWriter, r *http.Request) {
	list, err := store.ListRecurring(h.db)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if list == nil {
		list = []models.RecurringAssignment{}
	}
	writeJSON(w, 200, list)
}

// CreateRecurring creates a recurring rule and generates its assignment records.
func (h *Handler) CreateRecurring(w http.ResponseWriter, r *http.Request) {
	var rec models.RecurringAssignment
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if rec.ValidFrom == "" || rec.ValidUntil == "" {
		writeError(w, 400, "valid_from and valid_until required")
		return
	}
	if err := store.CreateRecurring(h.db, &rec); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	created, skipped, err := store.GenerateFromRecurring(h.db, &rec)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, map[string]any{
		"rule":    rec,
		"created": created,
		"skipped": skipped,
	})
}

func (h *Handler) DeleteRecurring(w http.ResponseWriter, r *http.Request) {
	if err := store.DeleteRecurring(h.db, chi.URLParam(r, "id")); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	w.WriteHeader(204)
}
