package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/pdf"
	"github.com/pak/kita-springer-manager/internal/store"
)

// ExportWorktimePDF generates a printable monthly worktime report.
// Query params:
//   - month=YYYY-MM  (required)
//   - provider_id    (optional — if set, only that provider)
func (h *Handler) ExportWorktimePDF(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	providerID := r.URL.Query().Get("provider_id")

	start, err := time.Parse("2006-01", month)
	if err != nil {
		writeError(w, 400, "month must be YYYY-MM")
		return
	}
	from := start.Format("2006-01-02")
	to := start.AddDate(0, 1, -1).Format("2006-01-02")

	all, err := store.ListAssignments(h.db, from, to)
	if err != nil {
		serverError(w, err)
		return
	}

	items := make([]models.Assignment, 0, len(all))
	for _, a := range all {
		if a.Status == models.StatusFree {
			continue
		}
		if a.ActualStartTime == "" && a.ActualEndTime == "" {
			continue
		}
		if providerID != "" && a.ProviderID != providerID {
			continue
		}
		items = append(items, a)
	}

	filename := buildFilename(month, items, providerID)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := pdf.Generate(w, month, items); err != nil {
		serverError(w, err)
		return
	}
}

func buildFilename(month string, items []models.Assignment, providerID string) string {
	base := "Arbeitszeiten_" + month
	if providerID != "" {
		for _, a := range items {
			if a.Provider != nil && a.Provider.Name != "" {
				base += "_" + sanitize(a.Provider.Name)
				break
			}
		}
	}
	return base + ".pdf"
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
