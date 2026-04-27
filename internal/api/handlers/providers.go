package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pak/kita-springer-manager/internal/importer"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/seeds"
	"github.com/pak/kita-springer-manager/internal/store"
)

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := store.ListProviders(h.db())
	if err != nil {
		serverError(w, err)
		return
	}
	if providers == nil {
		providers = []models.Provider{}
	}
	writeJSON(w, 200, providers)
}

func (h *Handler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var p models.Provider
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if p.ColorHex == "" {
		p.ColorHex = "#6366f1"
	}
	if err := validateProvider(&p); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err := store.CreateProvider(h.db(), &p); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 201, p)
}

func (h *Handler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := store.GetProvider(h.db(), id)
	if err != nil {
		serverError(w, err)
		return
	}
	if existing == nil {
		writeError(w, 404, "not found")
		return
	}
	var p models.Provider
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	if err := validateProvider(&p); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	p.ID = id
	if err := store.UpdateProvider(h.db(), &p); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, 200, p)
}

func (h *Handler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	if err := store.DeleteProvider(h.db(), chi.URLParam(r, "id")); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

// SeedKitas loads the built-in Kita list for a provider (e.g. "stadt_bern" or "stiftung_bern").
func (h *Handler) SeedKitas(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	provider, err := store.GetProvider(h.db(), id)
	if err != nil || provider == nil {
		writeError(w, 404, "provider not found")
		return
	}

	seedKey := r.URL.Query().Get("seed")
	kitas, err := seeds.Load(seedKey)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}

	count := 0
	for _, k := range kitas {
		k.ProviderID = provider.ID
		if err := store.CreateKita(h.db(), &k); err != nil {
			continue
		}
		count++
	}
	writeJSON(w, 200, map[string]int{"imported": count})
}

// ImportExcel handles Excel file upload and imports assignments for the provider.
func (h *Handler) ImportExcel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	provider, err := store.GetProvider(h.db(), id)
	if err != nil || provider == nil {
		writeError(w, 404, "provider not found")
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeError(w, 400, "file too large or not multipart")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "file required")
		return
	}
	defer file.Close()

	settings, err := store.GetSettings(h.db())
	if err != nil {
		serverError(w, err)
		return
	}
	if settings == nil || settings.UserName == "" {
		writeError(w, 422, "Name in den Einstellungen fehlt")
		return
	}

	opts := importer.Options{Year: time.Now().Year(), UserName: settings.UserName}
	if y := r.FormValue("year"); y != "" {
		if n, err := strconv.Atoi(y); err == nil {
			opts.Year = n
		}
	}
	if m := r.FormValue("month"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && n >= 1 && n <= 12 {
			opts.Month = n
		}
	}
	opts.KitaIDOverride = r.FormValue("kita_id")

	result, err := importer.ImportExcel(h.db(), file, provider, opts)
	if err != nil {
		writeError(w, 422, err.Error())
		return
	}
	writeJSON(w, 200, result)
}

