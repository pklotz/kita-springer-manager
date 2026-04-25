package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
)

func (h *Handler) ListKitas(w http.ResponseWriter, r *http.Request) {
	kitas, err := store.ListKitas(h.db)
	if err != nil {
		serverError(w, err)
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
		serverError(w, err)
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
	if err := decodeJSON(r, &k); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if k.Name == "" {
		writeError(w, 400, "name required")
		return
	}
	if err := store.CreateKita(h.db, &k); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 201, k)
}

func (h *Handler) UpdateKita(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := store.GetKita(h.db, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if existing == nil {
		writeError(w, 404, "not found")
		return
	}
	var k models.Kita
	if err := decodeJSON(r, &k); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	k.ID = id
	if err := store.UpdateKita(h.db, &k); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, k)
}

func (h *Handler) DeleteKita(w http.ResponseWriter, r *http.Request) {
	if err := store.DeleteKita(h.db, chi.URLParam(r, "id")); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

// LookupStops geocodes the Kita address and stores up to 2 nearest transit stops.
func (h *Handler) LookupStops(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	k, err := store.GetKita(h.db, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if k == nil {
		writeError(w, 404, "not found")
		return
	}
	if k.Address == "" {
		writeError(w, 422, "Keine Adresse hinterlegt")
		return
	}
	lat, lng, err := h.transit.Geocode(k.Address)
	if err != nil {
		upstreamError(w, err, "Geocoding fehlgeschlagen")
		return
	}
	result, err := h.transit.StopsNear(lat, lng)
	if err != nil {
		upstreamError(w, err, "Haltestellen-Suche fehlgeschlagen")
		return
	}
	stops := []string{}
	for _, s := range result.Stations {
		if s.Name == "" {
			continue
		}
		stops = append(stops, s.Name)
		if len(stops) >= 2 {
			break
		}
	}
	if len(stops) == 0 {
		writeError(w, 422, "Keine Haltestellen in der Nähe gefunden")
		return
	}
	k.Stops = stops
	if err := store.UpdateKita(h.db, k); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, k)
}
