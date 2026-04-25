package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
)

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := store.GetSettings(h.db)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, s)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var s models.Settings
	if err := decodeJSON(r, &s); err != nil {
		writeError(w, 400, "invalid request")
		return
	}

	if s.Canton == "" {
		s.Canton = "BE"
	}
	if !store.IsValidCanton(s.Canton) {
		writeError(w, 400, "unknown canton")
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
		serverError(w, err)
		return
	}

	if prev == nil || prev.Canton != s.Canton {
		if err := store.ReseedHolidays(h.db, s.Canton); err != nil {
			log.Printf("reseed holidays for %s: %v", s.Canton, err)
		}
	}

	writeJSON(w, 200, s)
}
