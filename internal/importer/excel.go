package importer

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/store"
	"github.com/xuri/excelize/v2"
)

type Result struct {
	Created  int      `json:"created"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Warnings []string `json:"warnings,omitempty"`
}

// Options customize the import beyond the provider's ExcelConfig.
type Options struct {
	UserName       string // required — user's name as it appears in Excel; full name OR any token matches
	Year           int    // required — year of the schedule
	Month          int    // optional (1–12) — if set, entries outside this month are skipped
	KitaIDOverride string // optional — if set, all assignments are linked to this kita (ignores KitaMapping)
}

// ImportExcel parses an xlsx file and creates/updates assignments for the configured person.
func ImportExcel(db *sql.DB, r io.Reader, provider *models.Provider, opts Options) (*Result, error) {
	if strings.TrimSpace(opts.UserName) == "" {
		return nil, fmt.Errorf("user name not configured in settings")
	}
	cfg := provider.ExcelConfig
	setDefaults(&cfg)

	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open excel: %w", err)
	}
	defer f.Close()

	result := &Result{}

	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("sheet %q: %v", sheetName, err))
			continue
		}
		if err := processSheet(db, rows, provider, &cfg, opts, sheetName, result); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("sheet %q: %v", sheetName, err))
		}
	}
	return result, nil
}

// nameMatches returns true if the Excel row's name cell equals the user's full
// name or any whitespace-separated token from it (case-insensitive).
func nameMatches(cell, userName string) bool {
	c := strings.ToLower(strings.TrimSpace(cell))
	if c == "" {
		return false
	}
	full := strings.ToLower(strings.TrimSpace(userName))
	if c == full {
		return true
	}
	for _, tok := range strings.Fields(full) {
		if c == tok {
			return true
		}
	}
	return false
}

func processSheet(db *sql.DB, rows [][]string, provider *models.Provider, cfg *models.ExcelConfig, opts Options, sheetName string, result *Result) error {
	if len(rows) < cfg.KitaRow {
		return fmt.Errorf("too few rows")
	}

	// Find header row (0-indexed)
	headerRow := rows[cfg.HeaderRow-1]
	kitaRow := rows[cfg.KitaRow-1]

	// Find person's row
	personRow := -1
	for i, row := range rows {
		if i < cfg.KitaRow {
			continue
		}
		if len(row) > 0 && nameMatches(row[0], opts.UserName) {
			personRow = i
			break
		}
	}
	if personRow == -1 {
		return fmt.Errorf("person %q not found", opts.UserName)
	}

	data := rows[personRow]
	firstCol := colIndex(cfg.FirstDayCol)

	for day := 0; day < cfg.DaysPerWeek; day++ {
		startColIdx := firstCol + day*cfg.ColsPerDay
		endColIdx := startColIdx + 1

		// Parse date from header
		date, dateMonth := parseDate(headerRow, startColIdx, opts.Year)
		if date == "" {
			continue
		}
		if opts.Month > 0 && dateMonth != opts.Month {
			continue
		}

		// Group/kita abbreviation from kita row
		groupName := ""
		if startColIdx < len(kitaRow) {
			groupName = strings.TrimSpace(kitaRow[startColIdx])
		}

		// Resolve kita ID — override wins over mapping.
		kitaID := opts.KitaIDOverride
		if kitaID == "" && groupName != "" {
			if id, ok := cfg.KitaMapping[groupName]; ok {
				kitaID = id
			}
		}

		// Only cells with a time value represent an actual assignment.
		// Anything else ("Frei", "Schule", "Kurs", "Ferien", holiday names, empty)
		// is skipped — those aren't shifts.
		startRaw := cellVal(data, startColIdx)
		endRaw := cellVal(data, endColIdx)

		startTime, ok := parseTime(startRaw)
		if !ok {
			continue
		}
		endTime, _ := parseTime(endRaw)

		hash := importHash(provider.ID, date, startRaw, endRaw)

		a := &models.Assignment{
			KitaID:     kitaID,
			ProviderID: provider.ID,
			GroupName:  groupName,
			Date:       date,
			StartTime:  startTime,
			EndTime:    endTime,
			Status:     models.StatusScheduled,
			Source:     models.SourceExcel,
			ImportHash: hash,
		}

		wasCreated, err := store.UpsertByHash(db, a)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", date, err))
			continue
		}
		if wasCreated {
			result.Created++
		} else {
			result.Updated++
		}
	}
	return nil
}

// colIndex converts Excel column letter to 0-based index ("A"→0, "B"→1 …).
func colIndex(col string) int {
	col = strings.ToUpper(col)
	idx := 0
	for _, c := range col {
		idx = idx*26 + int(c-'A'+1)
	}
	return idx - 1
}

func cellVal(row []string, i int) string {
	if i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}

var dayHeaderRe = regexp.MustCompile(`(\d+)\.(\d+)\.`)

func parseDate(headerRow []string, colIdx int, year int) (string, int) {
	val := cellVal(headerRow, colIdx)
	m := dayHeaderRe.FindStringSubmatch(val)
	if m == nil {
		return "", 0
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day), month
}

var hhmmRe = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)

// parseTime accepts either "HH:MM" strings (excelize returns display values for
// time-formatted cells) or Excel decimal day fractions (e.g. 0.354 for 08:30).
func parseTime(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if m := hhmmRe.FindStringSubmatch(s); m != nil {
		h, _ := strconv.Atoi(m[1])
		mm, _ := strconv.Atoi(m[2])
		if h >= 0 && h <= 23 && mm >= 0 && mm <= 59 {
			return fmt.Sprintf("%02d:%02d", h, mm), true
		}
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil && v >= 0 && v < 1 {
		totalMin := int(math.Round(v * 24 * 60))
		return fmt.Sprintf("%02d:%02d", totalMin/60, totalMin%60), true
	}
	return "", false
}

func importHash(providerID, date, start, end string) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s|%s|%s|%s", providerID, date, start, end)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func setDefaults(cfg *models.ExcelConfig) {
	if cfg.NameCol == "" {
		cfg.NameCol = "A"
	}
	if cfg.HeaderRow == 0 {
		cfg.HeaderRow = 2
	}
	if cfg.KitaRow == 0 {
		cfg.KitaRow = 3
	}
	if cfg.FirstDayCol == "" {
		cfg.FirstDayCol = "B"
	}
	if cfg.ColsPerDay == 0 {
		cfg.ColsPerDay = 2
	}
	if cfg.DaysPerWeek == 0 {
		cfg.DaysPerWeek = 5
	}
	if cfg.KitaMapping == nil {
		cfg.KitaMapping = map[string]string{}
	}
}
