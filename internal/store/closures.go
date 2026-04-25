package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/ch"
)

func ListClosures(db *sql.DB, from, to, closureType string) ([]models.Closure, error) {
	q := `SELECT id, type, COALESCE(reference_id,''), date, note, created_at FROM closures WHERE 1=1`
	var args []any
	if from != "" {
		q += " AND date >= ?"
		args = append(args, from)
	}
	if to != "" {
		q += " AND date <= ?"
		args = append(args, to)
	}
	if closureType != "" {
		q += " AND type = ?"
		args = append(args, closureType)
	}
	q += " ORDER BY date"

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Closure
	for rows.Next() {
		var c models.Closure
		if err := rows.Scan(&c.ID, &c.Type, &c.ReferenceID, &c.Date, &c.Note, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func CreateClosure(db *sql.DB, c *models.Closure) error {
	c.ID = uuid.New().String()
	c.CreatedAt = time.Now()
	refID := sql.NullString{String: c.ReferenceID, Valid: c.ReferenceID != ""}
	_, err := db.Exec(
		`INSERT OR IGNORE INTO closures (id, type, reference_id, date, note, created_at) VALUES (?,?,?,?,?,?)`,
		c.ID, c.Type, refID, c.Date, c.Note, c.CreatedAt,
	)
	return err
}

func DeleteClosure(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM closures WHERE id=?`, id)
	return err
}

// ClosureDates returns a set of dates that are blocked for a given recurring rule.
// Blocks: holidays, springerin vacation, provider closures, kita closures.
func ClosureDates(db *sql.DB, from, to, providerID, kitaID string) (map[string]bool, error) {
	rows, err := db.Query(
		`SELECT date FROM closures
		 WHERE date >= ? AND date <= ?
		   AND (
		     type = 'holiday'
		     OR type = 'springerin'
		     OR (type = 'provider' AND reference_id = ?)
		     OR (type = 'kita'     AND reference_id = ?)
		   )`,
		from, to, providerID, kitaID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := map[string]bool{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		set[d] = true
	}
	return set, rows.Err()
}

// cantonHolidayLists maps ISO 3166-2:CH canton codes to the holiday list
// defined by github.com/rickar/cal/v2/ch.
var cantonHolidayLists = map[string][]*cal.Holiday{
	"AG": ch.HolidaysAG, "AI": ch.HolidaysAI, "AR": ch.HolidaysAR,
	"BE": ch.HolidaysBE, "BL": ch.HolidaysBL, "BS": ch.HolidaysBS,
	"FR": ch.HolidaysFR, "GE": ch.HolidaysGE, "GL": ch.HolidaysGL,
	"GR": ch.HolidaysGR, "JU": ch.HolidaysJU, "LU": ch.HolidaysLU,
	"NE": ch.HolidaysNE, "NW": ch.HolidaysNW, "OW": ch.HolidaysOW,
	"SG": ch.HolidaysSG, "SH": ch.HolidaysSH, "SO": ch.HolidaysSO,
	"SZ": ch.HolidaysSZ, "TG": ch.HolidaysTG, "TI": ch.HolidaysTI,
	"UR": ch.HolidaysUR, "VD": ch.HolidaysVD, "VS": ch.HolidaysVS,
	"ZG": ch.HolidaysZG, "ZH": ch.HolidaysZH,
}

// IsValidCanton reports whether the given code is a supported CH canton.
func IsValidCanton(canton string) bool {
	_, ok := cantonHolidayLists[strings.ToUpper(canton)]
	return ok
}

// SeedHolidays inserts public holidays for the given canton and year.
// The unique index on (type, date, reference_id) makes this idempotent.
func SeedHolidays(db *sql.DB, canton string, year int) error {
	list, ok := cantonHolidayLists[strings.ToUpper(canton)]
	if !ok {
		return fmt.Errorf("unknown canton %q", canton)
	}
	for _, h := range list {
		actual, _ := h.Calc(year)
		if actual.IsZero() {
			continue
		}
		c := models.Closure{
			Type: models.ClosureHoliday,
			Date: actual.Format("2006-01-02"),
			Note: h.Name,
		}
		if err := CreateClosure(db, &c); err != nil {
			return err
		}
	}
	return nil
}

// ReseedHolidays wipes all holiday closures and re-seeds them for the given
// canton and a window of [currentYear, currentYear+2]. Called on server start
// and whenever the canton setting changes.
func ReseedHolidays(db *sql.DB, canton string) error {
	if _, err := db.Exec(`DELETE FROM closures WHERE type='holiday'`); err != nil {
		return err
	}
	year := time.Now().Year()
	for y := year; y <= year+2; y++ {
		if err := SeedHolidays(db, canton, y); err != nil {
			return err
		}
	}
	return nil
}
