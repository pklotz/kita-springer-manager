package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
)

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := store.GetSettings(h.db)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, s)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var s models.Settings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, 400, "invalid request")
		return
	}

	// Re-geocode home address if it changed or coords are missing.
	prev, _ := store.GetSettings(h.db)
	addr := strings.TrimSpace(s.HomeAddress)
	needsGeocode := addr != "" && (prev == nil || prev.HomeAddress != addr || s.HomeLat == 0)
	if needsGeocode {
		if lat, lng, err := h.transit.Geocode(addr); err != nil {
			log.Printf("geocode %q: %v", addr, err)
			s.HomeLat, s.HomeLng = 0, 0
		} else {
			s.HomeLat, s.HomeLng = lat, lng
		}
	} else if addr == "" {
		s.HomeLat, s.HomeLng = 0, 0
	}

	if err := store.SaveSettings(h.db, &s); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, s)
}
