package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pak/kita-springer-manager/internal/audit"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
)

// conflictMessage renders a German error message for an assignment conflict.
func conflictMessage(reason store.ConflictReason, other *models.Assignment) string {
	kita := "(unbekannte Kita)"
	if other.Kita != nil && other.Kita.Name != "" {
		kita = other.Kita.Name
	}
	switch reason {
	case store.ConflictSameKita:
		return fmt.Sprintf("Es existiert bereits ein Einsatz in %s am %s.", kita, other.Date)
	case store.ConflictOverlap:
		window := "ganztägig"
		if other.StartTime != "" && other.EndTime != "" {
			window = other.StartTime + "–" + other.EndTime
		}
		return fmt.Sprintf("Überschneidet sich mit Einsatz in %s (%s).", kita, window)
	}
	return "Konflikt mit bestehendem Einsatz."
}

func (h *Handler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	assignments, err := store.ListAssignments(h.db(), from, to)
	if err != nil {
		serverError(w, err)
		return
	}
	if assignments == nil {
		assignments = []models.Assignment{}
	}
	writeJSON(w, 200, assignments)
}

func (h *Handler) GetAssignment(w http.ResponseWriter, r *http.Request) {
	a, err := store.GetAssignment(h.db(), chi.URLParam(r, "id"))
	if err != nil {
		serverError(w, err)
		return
	}
	if a == nil {
		writeError(w, 404, "not found")
		return
	}
	writeJSON(w, 200, a)
}

func (h *Handler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	var a models.Assignment
	if err := decodeJSON(r, &a); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if err := validateAssignment(&a); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := h.resolveAssignmentProvider(&a); err != nil {
		writeError(w, err.status, err.msg)
		return
	}
	if conflict, reason, err := store.FindAssignmentConflict(h.db(), &a, ""); err != nil {
		serverError(w, err)
		return
	} else if conflict != nil {
		writeError(w, 409, conflictMessage(reason, conflict))
		return
	}
	if err := store.CreateAssignment(h.db(), &a); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 201, a)
}

func (h *Handler) UpdateAssignment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := store.GetAssignment(h.db(), id)
	if err != nil {
		serverError(w, err)
		return
	}
	if existing == nil {
		writeError(w, 404, "not found")
		return
	}
	var a models.Assignment
	if err := decodeJSON(r, &a); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	a.ID = id
	if err := validateAssignment(&a); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := h.resolveAssignmentProvider(&a); err != nil {
		writeError(w, err.status, err.msg)
		return
	}
	if conflict, reason, err := store.FindAssignmentConflict(h.db(), &a, id); err != nil {
		serverError(w, err)
		return
	} else if conflict != nil {
		writeError(w, 409, conflictMessage(reason, conflict))
		return
	}
	if err := store.UpdateAssignment(h.db(), &a); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, a)
}

type httpErr struct {
	status int
	msg    string
}

// resolveAssignmentProvider looks up the kita for a.KitaID and overrides
// a.ProviderID with the kita's provider. Any client-supplied provider_id
// is discarded — the server is authoritative.
func (h *Handler) resolveAssignmentProvider(a *models.Assignment) *httpErr {
	kita, err := store.GetKita(h.db(), a.KitaID)
	if err != nil {
		// Log the underlying DB error for diagnostics, but surface a generic
		// message to the client.
		audit.L().Error("server.error", "where", "resolveAssignmentProvider", "err", err.Error())
		return &httpErr{500, "Interner Serverfehler"}
	}
	if kita == nil {
		return &httpErr{400, "unknown kita_id"}
	}
	if kita.ProviderID == "" {
		return &httpErr{400, "kita has no provider"}
	}
	a.ProviderID = kita.ProviderID
	return nil
}

func (h *Handler) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	if err := store.DeleteAssignment(h.db(), chi.URLParam(r, "id")); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) BulkDeleteAssignments(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if len(body.IDs) == 0 {
		writeError(w, 400, "ids required")
		return
	}
	deleted, err := store.BulkDeleteAssignments(h.db(), body.IDs)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, map[string]int64{"deleted": deleted})
}
