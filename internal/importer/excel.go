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

// ImportExcel parses an xlsx file and creates/updates assignments for the configured person.
func ImportExcel(db *sql.DB, r io.Reader, provider *models.Provider, year int) (*Result, error) {
	cfg := provider.ExcelConfig
	if cfg.PersonName == "" {
		return nil, fmt.Errorf("excel_config.person_name not configured for this provider")
	}
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
		if err := processSheet(db, rows, provider, &cfg, year, sheetName, result); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("sheet %q: %v", sheetName, err))
		}
	}
	return result, nil
}

func processSheet(db *sql.DB, rows [][]string, provider *models.Provider, cfg *models.ExcelConfig, year int, sheetName string, result *Result) error {
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
		if len(row) > 0 && strings.EqualFold(strings.TrimSpace(row[0]), strings.TrimSpace(cfg.PersonName)) {
			personRow = i
			break
		}
	}
	if personRow == -1 {
		return fmt.Errorf("person %q not found", cfg.PersonName)
	}

	data := rows[personRow]
	firstCol := colIndex(cfg.FirstDayCol)

	for day := 0; day < cfg.DaysPerWeek; day++ {
		startColIdx := firstCol + day*cfg.ColsPerDay
		endColIdx := startColIdx + 1

		// Parse date from header
		date := parseDate(headerRow, startColIdx, year)
		if date == "" {
			continue
		}

		// Group/kita abbreviation from kita row
		groupName := ""
		if startColIdx < len(kitaRow) {
			groupName = strings.TrimSpace(kitaRow[startColIdx])
		}

		// Resolve kita ID from mapping
		kitaID := ""
		if groupName != "" {
			if id, ok := cfg.KitaMapping[groupName]; ok {
				kitaID = id
			}
		}

		// Start/end time values
		startRaw := cellVal(data, startColIdx)
		endRaw := cellVal(data, endColIdx)

		var status, startTime, endTime string
		if isTimeValue(startRaw) {
			status = models.StatusScheduled
			startTime = decimalToTime(startRaw)
			if isTimeValue(endRaw) {
				endTime = decimalToTime(endRaw)
			}
		} else if startRaw != "" {
			// "Frei", "Schule", "Kurs", etc.
			status = models.StatusFree
		} else {
			continue // empty cell → no entry
		}

		hash := importHash(provider.ID, date, startRaw, endRaw)

		a := &models.Assignment{
			KitaID:     kitaID,
			ProviderID: provider.ID,
			GroupName:  groupName,
			Date:       date,
			StartTime:  startTime,
			EndTime:    endTime,
			Status:     status,
			Source:     models.SourceExcel,
			ImportHash: hash,
			Notes:      freeLabel(startRaw, status),
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

func parseDate(headerRow []string, colIdx int, year int) string {
	val := cellVal(headerRow, colIdx)
	m := dayHeaderRe.FindStringSubmatch(val)
	if m == nil {
		return ""
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func isTimeValue(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func decimalToTime(s string) string {
	v, _ := strconv.ParseFloat(s, 64)
	totalMin := int(math.Round(v * 24 * 60))
	h := totalMin / 60
	m := totalMin % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func freeLabel(raw, status string) string {
	if status == models.StatusFree && raw != "" {
		return raw
	}
	return ""
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
