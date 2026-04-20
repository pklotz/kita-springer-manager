package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pak/kita-springer-manager/internal/models"
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

// SeedHolidays inserts Bern canton public holidays for the given year.
// The unique index on (type, date, reference_id) makes this idempotent.
func SeedHolidays(db *sql.DB, year int) error {
	for _, h := range bernHolidays(year) {
		if err := CreateClosure(db, &h); err != nil {
			return err
		}
	}
	return nil
}

func bernHolidays(year int) []models.Closure {
	easter := easterSunday(year)
	type entry struct {
		t    time.Time
		name string
	}
	dates := []entry{
		{time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC), "Neujahr"},
		{time.Date(year, 1, 2, 0, 0, 0, 0, time.UTC), "Berchtoldstag"},
		{easter.AddDate(0, 0, -2), "Karfreitag"},
		{easter.AddDate(0, 0, 1), "Ostermontag"},
		{time.Date(year, 5, 1, 0, 0, 0, 0, time.UTC), "Tag der Arbeit"},
		{easter.AddDate(0, 0, 39), "Auffahrt"},
		{easter.AddDate(0, 0, 50), "Pfingstmontag"},
		{time.Date(year, 8, 1, 0, 0, 0, 0, time.UTC), "Bundesfeier"},
		{time.Date(year, 12, 25, 0, 0, 0, 0, time.UTC), "Weihnachten"},
		{time.Date(year, 12, 26, 0, 0, 0, 0, time.UTC), "Stephanstag"},
	}
	out := make([]models.Closure, len(dates))
	for i, d := range dates {
		out[i] = models.Closure{
			Type: models.ClosureHoliday,
			Date: d.t.Format("2006-01-02"),
			Note: d.name,
		}
	}
	return out
}

// easterSunday computes Easter Sunday using the Anonymous Gregorian algorithm.
func easterSunday(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h+l-7*m+114)%31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
