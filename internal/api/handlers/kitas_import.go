package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pak/kita-springer-manager/internal/audit"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
	"github.com/xuri/excelize/v2"
)

// ImportKitasExcel imports Kitas from the standard Excel format.
// Standard columns (row 1 = header skipped, row 2+ = data):
//
//	A: Name   B: Adresse   C: ÖV-Haltestelle   D: Telefon
//	E: Email  F: Gruppen (;-separated)          G: Notizen
//	H: Leitung (director name)                  I: Foto-URL
//
// Query params:
//   - provider_id: optional, assigns all imported Kitas to this provider
func (h *Handler) ImportKitasExcel(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, 400, "multipart form required")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "file field required")
		return
	}
	defer file.Close()

	providerID := r.URL.Query().Get("provider_id")

	f, err := excelize.OpenReader(file)
	if err != nil {
		audit.L().Warn("excel.open", "err", err.Error())
		writeError(w, 422, "Excel-Datei kann nicht geöffnet werden")
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		writeError(w, 422, "Excel-Datei enthält kein Blatt")
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		audit.L().Warn("excel.read", "err", err.Error())
		writeError(w, 422, "Excel-Blatt kann nicht gelesen werden")
		return
	}

	type result struct {
		Imported int      `json:"imported"`
		Skipped  int      `json:"skipped"`
		Warnings []string `json:"warnings,omitempty"`
	}
	res := result{}

	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}
		name := cell(row, 0)
		if name == "" {
			continue
		}

		groups := []string{}
		if g := cell(row, 5); g != "" {
			for _, gr := range strings.Split(g, ";") {
				if t := strings.TrimSpace(gr); t != "" {
					groups = append(groups, t)
				}
			}
		}

		k := &models.Kita{
			ProviderID:  providerID,
			Name:        name,
			Address:     cell(row, 1),
			StopName:    cell(row, 2),
			Phone:       cell(row, 3),
			Email:       cell(row, 4),
			Groups:      groups,
			Notes:       cell(row, 6),
			LeitungName: cell(row, 7),
			PhotoURL:    cell(row, 8),
		}

		if err := validateKita(k); err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("Zeile %d: %s", i+1, err.Error()))
			res.Skipped++
			continue
		}
		if err := store.CreateKita(h.db, k); err != nil {
			audit.L().Warn("kita.import.row", "row", i+1, "err", err.Error())
			res.Warnings = append(res.Warnings, fmt.Sprintf("Zeile %d: konnte nicht gespeichert werden", i+1))
			res.Skipped++
			continue
		}
		res.Imported++
	}

	writeJSON(w, 200, res)
}

func cell(row []string, i int) string {
	if i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}
