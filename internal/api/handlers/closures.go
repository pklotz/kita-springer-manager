package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
)

func (h *Handler) ListClosures(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	ctype := r.URL.Query().Get("type")
	closures, err := store.ListClosures(h.db, from, to, ctype)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if closures == nil {
		closures = []models.Closure{}
	}
	writeJSON(w, 200, closures)
}

func (h *Handler) CreateClosure(w http.ResponseWriter, r *http.Request) {
	var c models.Closure
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if c.Date == "" || c.Type == "" {
		writeError(w, 400, "date and type required")
		return
	}
	if err := store.CreateClosure(h.db, &c); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, c)
}

func (h *Handler) DeleteClosure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := store.DeleteClosure(h.db, id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	w.WriteHeader(204)
}
