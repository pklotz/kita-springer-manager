package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
	"github.com/pak/kita-springer-manager/internal/validate"
)

func (h *Handler) ListClosures(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	ctype := r.URL.Query().Get("type")
	closures, err := store.ListClosures(h.db(), from, to, ctype)
	if err != nil {
		serverError(w, err)
		return
	}
	if closures == nil {
		closures = []models.Closure{}
	}
	writeJSON(w, 200, closures)
}

func (h *Handler) CreateClosure(w http.ResponseWriter, r *http.Request) {
	var c models.Closure
	if err := decodeJSON(r, &c); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if err := validateClosure(&c); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := store.CreateClosure(h.db(), &c); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 201, c)
}

func (h *Handler) DeleteClosure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := store.DeleteClosure(h.db(), id); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

// batchClosureRequest is the request body for POST /api/closures/batch.
type batchClosureRequest struct {
	Type        string `json:"type"`
	ReferenceID string `json:"reference_id"`
	DateFrom    string `json:"date_from"`
	DateTo      string `json:"date_to"`
	Note        string `json:"note"`
	Force       bool   `json:"force"`
}

// CreateClosuresBatch implements POST /api/closures/batch.
// It atomically creates one closure per day in the given date range.
// For springerin closures without force=true, it first checks for assignment
// conflicts and returns 409 if any are found.
func (h *Handler) CreateClosuresBatch(w http.ResponseWriter, r *http.Request) {
	var req batchClosureRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, 400, "Ungültiges JSON")
		return
	}

	// Validate type
	switch req.Type {
	case models.ClosureSpringerin, models.ClosureProvider, models.ClosureKita, models.ClosureHoliday:
	default:
		writeError(w, 400, "type: ungültiger Wert")
		return
	}

	// Validate date_from
	if err := validate.Date(req.DateFrom, "date_from"); err != nil {
		writeError(w, 400, err.Error())
		return
	}

	// date_to defaults to date_from if empty
	dateTo := req.DateTo
	if dateTo == "" {
		dateTo = req.DateFrom
	}
	if err := validate.DateRange(req.DateFrom, dateTo); err != nil {
		writeError(w, 400, err.Error())
		return
	}

	// Validate note length
	if err := validate.MaxLen(req.Note, "note", 500); err != nil {
		writeError(w, 400, err.Error())
		return
	}

	// Build date list
	from, _ := time.Parse("2006-01-02", req.DateFrom)
	to, _ := time.Parse("2006-01-02", dateTo)
	days := int(to.Sub(from).Hours()/24) + 1
	if days > 366 {
		writeError(w, 400, "Zeitraum zu gross (max. 366 Tage)")
		return
	}
	dates := make([]string, 0, days)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}

	// Conflict check: only for springerin closures without force
	if req.Type == models.ClosureSpringerin && !req.Force {
		conflicts, err := store.ClosureAssignmentConflicts(h.db(), dates)
		if err != nil {
			serverError(w, err)
			return
		}
		if len(conflicts) > 0 {
			parts := make([]string, 0, len(conflicts))
			for _, a := range conflicts {
				kitaName := a.Kita.Name
				if kitaName == "" {
					kitaName = "unbekannte Kita"
				}
				parts = append(parts, fmt.Sprintf("%s (%s)", a.Date, kitaName))
			}
			msg := "An folgenden Tagen sind bereits Einsätze geplant: " +
				strings.Join(parts, ", ") +
				" — trotzdem als Abwesenheit eintragen?"
			writeJSON(w, 409, map[string]any{
				"error":     msg,
				"conflicts": conflicts,
			})
			return
		}
	}

	// Build closure list and insert atomically
	closures := make([]models.Closure, len(dates))
	for i, d := range dates {
		closures[i] = models.Closure{
			Type:        req.Type,
			ReferenceID: req.ReferenceID,
			Date:        d,
			Note:        req.Note,
		}
	}
	if err := store.CreateClosuresBatch(h.db(), closures); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 201, map[string]int{"created": len(dates)})
}
