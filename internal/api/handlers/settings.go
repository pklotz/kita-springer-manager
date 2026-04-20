package handlers

import (
	"encoding/json"
	"net/http"

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
	if err := store.SaveSettings(h.db, &s); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, s)
}
