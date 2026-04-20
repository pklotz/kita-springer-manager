package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
)

func (h *Handler) ListKitas(w http.ResponseWriter, r *http.Request) {
	kitas, err := store.ListKitas(h.db)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if kitas == nil {
		kitas = []models.Kita{}
	}
	writeJSON(w, 200, kitas)
}

func (h *Handler) GetKita(w http.ResponseWriter, r *http.Request) {
	k, err := store.GetKita(h.db, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if k == nil {
		writeError(w, 404, "not found")
		return
	}
	writeJSON(w, 200, k)
}

func (h *Handler) CreateKita(w http.ResponseWriter, r *http.Request) {
	var k models.Kita
	if err := json.NewDecoder(r.Body).Decode(&k); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if k.Name == "" {
		writeError(w, 400, "name required")
		return
	}
	if err := store.CreateKita(h.db, &k); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, k)
}

func (h *Handler) UpdateKita(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := store.GetKita(h.db, id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if existing == nil {
		writeError(w, 404, "not found")
		return
	}
	var k models.Kita
	if err := json.NewDecoder(r.Body).Decode(&k); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	k.ID = id
	if err := store.UpdateKita(h.db, &k); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, k)
}

func (h *Handler) DeleteKita(w http.ResponseWriter, r *http.Request) {
	if err := store.DeleteKita(h.db, chi.URLParam(r, "id")); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	w.WriteHeader(204)
}
