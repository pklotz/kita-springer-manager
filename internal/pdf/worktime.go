// Package pdf generates printable worktime reports grouped by provider.
package pdf

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/pak/kita-springer-manager/internal/models"
)

// translator maps UTF-8 strings to the WinAnsi (CP1252) encoding used by
// fpdf's built-in Helvetica font. Without this, non-ASCII characters like
// ä/ö/ü/–/ß come out as mojibake.
type translator func(string) string

// Generate writes a multi-page PDF (one page per provider) with all recorded
// assignments in the given month. Groups are split by provider; within a
// provider, rows are chronological.
func Generate(w io.Writer, month string, assignments []models.Assignment) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	byProvider := groupByProvider(assignments)
	if len(byProvider) == 0 {
		renderEmpty(pdf, tr, month)
	} else {
		sort.Slice(byProvider, func(i, j int) bool {
			return byProvider[i].name < byProvider[j].name
		})
		for _, g := range byProvider {
			renderProviderPage(pdf, tr, month, g)
		}
	}
	return pdf.Output(w)
}

type providerGroup struct {
	id    string
	name  string
	items []models.Assignment
}

func groupByProvider(assignments []models.Assignment) []providerGroup {
	m := map[string]*providerGroup{}
	for _, a := range assignments {
		pid := a.ProviderID
		name := ""
		if a.Provider != nil {
			name = a.Provider.Name
		}
		if _, ok := m[pid]; !ok {
			m[pid] = &providerGroup{id: pid, name: name}
		}
		m[pid].items = append(m[pid].items, a)
	}
	out := make([]providerGroup, 0, len(m))
	for _, g := range m {
		sort.Slice(g.items, func(i, j int) bool {
			if g.items[i].Date != g.items[j].Date {
				return g.items[i].Date < g.items[j].Date
			}
			return g.items[i].StartTime < g.items[j].StartTime
		})
		out = append(out, *g)
	}
	return out
}

func renderEmpty(pdf *fpdf.Fpdf, tr translator, month string) {
	pdf.AddPage()
	renderHeader(pdf, tr, month, "—")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "I", 10)
	pdf.Cell(0, 6, tr("Keine erfassten Einsätze in diesem Monat."))
}

func renderProviderPage(pdf *fpdf.Fpdf, tr translator, month string, g providerGroup) {
	pdf.AddPage()
	renderHeader(pdf, tr, month, g.name)
	pdf.Ln(2)
	renderTable(pdf, tr, g.items)
	pdf.Ln(4)
	renderSummary(pdf, tr, g.items)
}

func renderHeader(pdf *fpdf.Fpdf, tr translator, month, providerName string) {
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 8, tr("Arbeitszeiten – "+monthLabel(month)))
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(140, 5, tr("Träger: "+providerName))
	pdf.CellFormat(40, 5, tr("Erstellt: "+time.Now().Format("02.01.2006")), "", 1, "R", false, 0, "")
	pdf.Ln(2)
}

// Column widths in mm. Sum must match usable width (A4 minus margins = 180).
var colWidths = struct {
	date, kita, start, pause, end, work, note float64
}{
	date: 22, kita: 42, start: 18, pause: 28, end: 18, work: 16, note: 36,
}

func renderTable(pdf *fpdf.Fpdf, tr translator, items []models.Assignment) {
	cw := colWidths

	// Header row
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(235, 235, 235)
	pdf.SetDrawColor(150, 150, 150)
	pdf.SetLineWidth(0.2)
	h := 6.5
	pdf.CellFormat(cw.date, h, tr("Datum"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(cw.kita, h, tr("Kita"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(cw.start, h, tr("Beginn"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(cw.pause, h, tr("Pause"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(cw.end, h, tr("Ende"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(cw.work, h, tr("Arbeitszeit"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(cw.note, h, tr("Bemerkung"), "1", 1, "L", true, 0, "")

	// Data rows
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetFillColor(250, 250, 250)
	fill := false
	rh := 5.5
	for _, a := range items {
		netMin := netWorkMinutes(a)
		pdf.CellFormat(cw.date, rh, tr(dateLabel(a.Date)), "1", 0, "L", fill, 0, "")
		pdf.CellFormat(cw.kita, rh, tr(truncate(kitaName(a), 30)), "1", 0, "L", fill, 0, "")
		pdf.CellFormat(cw.start, rh, orDash(a.ActualStartTime), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(cw.pause, rh, tr(pauseLabel(a)), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(cw.end, rh, orDash(a.ActualEndTime), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(cw.work, rh, formatHours(netMin), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(cw.note, rh, tr(truncate(noteLabel(a), 26)), "1", 1, "L", fill, 0, "")
		fill = !fill
	}
}

func renderSummary(pdf *fpdf.Fpdf, tr translator, items []models.Assignment) {
	var netTotal, breakTotal int
	kitaStats := map[string]*struct {
		name string
		net  int
		cnt  int
	}{}
	for _, a := range items {
		net := netWorkMinutes(a)
		brk := breakMinutes(a)
		netTotal += net
		breakTotal += brk
		k := kitaName(a)
		if _, ok := kitaStats[k]; !ok {
			kitaStats[k] = &struct {
				name string
				net  int
				cnt  int
			}{name: k}
		}
		kitaStats[k].net += net
		kitaStats[k].cnt++
	}

	// Single-line, bold summary
	pdf.SetFont("Helvetica", "B", 11)
	line := fmt.Sprintf("Monatstotal:  %s  ·  %s Std. Arbeit  ·  %s Std. Pause",
		einsatzLabel(len(items)),
		formatHours(netTotal),
		formatHours(breakTotal),
	)
	pdf.Cell(0, 7, tr(line))
	pdf.Ln(10)

	if len(kitaStats) > 1 {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Cell(0, 6, tr("Pro Kita"))
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "", 9)
		names := make([]string, 0, len(kitaStats))
		for n := range kitaStats {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			s := kitaStats[n]
			pdf.CellFormat(70, 5, tr(truncate(s.name, 40)), "", 0, "L", false, 0, "")
			pdf.CellFormat(30, 5, formatHours(s.net)+" Std.", "", 0, "R", false, 0, "")
			pdf.CellFormat(30, 5, tr(einsatzLabel(s.cnt)), "", 1, "R", false, 0, "")
		}
	}
}

// --- helpers -----------------------------------------------------------------

func einsatzLabel(n int) string {
	if n == 1 {
		return "1 Einsatz"
	}
	return fmt.Sprintf("%d Einsätze", n)
}

func kitaName(a models.Assignment) string {
	if a.Kita != nil && a.Kita.Name != "" {
		return a.Kita.Name
	}
	return "–"
}

func noteLabel(a models.Assignment) string {
	parts := []string{}
	if a.GroupName != "" {
		parts = append(parts, a.GroupName)
	}
	if a.Notes != "" {
		parts = append(parts, a.Notes)
	}
	return strings.Join(parts, " · ")
}

func pauseLabel(a models.Assignment) string {
	if a.ActualBreakStart == "" || a.ActualBreakEnd == "" {
		return "–"
	}
	return a.ActualBreakStart + "–" + a.ActualBreakEnd
}

func orDash(s string) string {
	if s == "" {
		return "–"
	}
	return s
}

func dateLabel(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return germanWeekday(t) + " " + t.Format("02.01.")
}

func germanWeekday(t time.Time) string {
	switch t.Weekday() {
	case time.Monday:
		return "Mo"
	case time.Tuesday:
		return "Di"
	case time.Wednesday:
		return "Mi"
	case time.Thursday:
		return "Do"
	case time.Friday:
		return "Fr"
	case time.Saturday:
		return "Sa"
	default:
		return "So"
	}
}

func monthLabel(ym string) string {
	t, err := time.Parse("2006-01", ym)
	if err != nil {
		return ym
	}
	months := []string{
		"Januar", "Februar", "März", "April", "Mai", "Juni",
		"Juli", "August", "September", "Oktober", "November", "Dezember",
	}
	return fmt.Sprintf("%s %d", months[t.Month()-1], t.Year())
}

// parseHM turns "HH:MM" into total minutes. Returns (0, false) for empty/bad.
func parseHM(s string) (int, bool) {
	if s == "" || len(s) < 4 {
		return 0, false
	}
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, false
	}
	return h*60 + m, true
}

func breakMinutes(a models.Assignment) int {
	s, ok1 := parseHM(a.ActualBreakStart)
	e, ok2 := parseHM(a.ActualBreakEnd)
	if !ok1 || !ok2 || e <= s {
		return 0
	}
	return e - s
}

func grossMinutes(a models.Assignment) int {
	s, ok1 := parseHM(a.ActualStartTime)
	e, ok2 := parseHM(a.ActualEndTime)
	if !ok1 || !ok2 || e <= s {
		return 0
	}
	return e - s
}

func netWorkMinutes(a models.Assignment) int {
	n := grossMinutes(a) - breakMinutes(a)
	if n < 0 {
		return 0
	}
	return n
}

func formatHours(minutes int) string {
	return fmt.Sprintf("%.2f", float64(minutes)/60.0)
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}
