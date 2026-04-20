package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
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
	assignments, err := store.ListAssignments(h.db, from, to)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if assignments == nil {
		assignments = []models.Assignment{}
	}
	writeJSON(w, 200, assignments)
}

func (h *Handler) GetAssignment(w http.ResponseWriter, r *http.Request) {
	a, err := store.GetAssignment(h.db, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 500, err.Error())
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
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if a.KitaID == "" || a.Date == "" {
		writeError(w, 400, "kita_id and date required")
		return
	}
	if conflict, reason, err := store.FindAssignmentConflict(h.db, &a, ""); err != nil {
		writeError(w, 500, err.Error())
		return
	} else if conflict != nil {
		writeError(w, 409, conflictMessage(reason, conflict))
		return
	}
	if err := store.CreateAssignment(h.db, &a); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, a)
}

func (h *Handler) UpdateAssignment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := store.GetAssignment(h.db, id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if existing == nil {
		writeError(w, 404, "not found")
		return
	}
	var a models.Assignment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	a.ID = id
	if conflict, reason, err := store.FindAssignmentConflict(h.db, &a, id); err != nil {
		writeError(w, 500, err.Error())
		return
	} else if conflict != nil {
		writeError(w, 409, conflictMessage(reason, conflict))
		return
	}
	if err := store.UpdateAssignment(h.db, &a); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, a)
}

func (h *Handler) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	if err := store.DeleteAssignment(h.db, chi.URLParam(r, "id")); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) BulkDeleteAssignments(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if len(body.IDs) == 0 {
		writeError(w, 400, "ids required")
		return
	}
	deleted, err := store.BulkDeleteAssignments(h.db, body.IDs)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]int64{"deleted": deleted})
}
