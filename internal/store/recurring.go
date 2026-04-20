package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pak/kita-springer-manager/internal/models"
)

func ListRecurring(db *sql.DB) ([]models.RecurringAssignment, error) {
	rows, err := db.Query(`
		SELECT r.id, COALESCE(r.kita_id,''), COALESCE(r.provider_id,''),
		       r.group_name, r.day_of_week, r.start_time, r.end_time,
		       r.valid_from, r.valid_until, r.notes, r.created_at,
		       COALESCE(k.name,''), COALESCE(k.address,''), COALESCE(k.stop_name,'')
		FROM recurring_assignments r
		LEFT JOIN kitas k ON k.id = r.kita_id
		ORDER BY r.day_of_week, r.start_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.RecurringAssignment
	for rows.Next() {
		var r models.RecurringAssignment
		r.Kita = &models.Kita{}
		if err := rows.Scan(
			&r.ID, &r.KitaID, &r.ProviderID,
			&r.GroupName, &r.DayOfWeek, &r.StartTime, &r.EndTime,
			&r.ValidFrom, &r.ValidUntil, &r.Notes, &r.CreatedAt,
			&r.Kita.Name, &r.Kita.Address, &r.Kita.StopName,
		); err != nil {
			return nil, err
		}
		r.Kita.ID = r.KitaID
		out = append(out, r)
	}
	return out, rows.Err()
}

func CreateRecurring(db *sql.DB, r *models.RecurringAssignment) error {
	r.ID = uuid.New().String()
	r.CreatedAt = time.Now()
	kitaID := sql.NullString{String: r.KitaID, Valid: r.KitaID != ""}
	providerID := sql.NullString{String: r.ProviderID, Valid: r.ProviderID != ""}
	_, err := db.Exec(
		`INSERT INTO recurring_assignments (id, kita_id, provider_id, group_name, day_of_week, start_time, end_time, valid_from, valid_until, notes, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID, kitaID, providerID, r.GroupName, r.DayOfWeek,
		r.StartTime, r.EndTime, r.ValidFrom, r.ValidUntil, r.Notes, r.CreatedAt,
	)
	return err
}

func DeleteRecurring(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM recurring_assignments WHERE id=?`, id)
	return err
}

// GenerateFromRecurring creates assignment records for each matching weekday in the rule's date range.
// Returns counts of created and skipped assignments.
func GenerateFromRecurring(db *sql.DB, r *models.RecurringAssignment) (created, skipped int, err error) {
	from, err := time.Parse("2006-01-02", r.ValidFrom)
	if err != nil {
		return 0, 0, err
	}
	until, err := time.Parse("2006-01-02", r.ValidUntil)
	if err != nil {
		return 0, 0, err
	}

	blocked, err := ClosureDates(db, r.ValidFrom, r.ValidUntil, r.ProviderID, r.KitaID)
	if err != nil {
		return 0, 0, err
	}

	for d := from; !d.After(until); d = d.AddDate(0, 0, 1) {
		// Go: Sunday=0 … Saturday=6; our convention: Monday=0 … Sunday=6
		goDay := int(d.Weekday())
		ourDay := (goDay + 6) % 7
		if ourDay != r.DayOfWeek {
			continue
		}

		dateStr := d.Format("2006-01-02")

		if blocked[dateStr] {
			skipped++
			continue
		}

		hash := "recurring:" + r.ID + ":" + dateStr

		exists, dbErr := FindAssignmentByHash(db, hash)
		if dbErr != nil {
			return created, skipped, dbErr
		}
		if exists {
			skipped++
			continue
		}

		a := &models.Assignment{
			KitaID:     r.KitaID,
			ProviderID: r.ProviderID,
			GroupName:  r.GroupName,
			Date:       dateStr,
			StartTime:  r.StartTime,
			EndTime:    r.EndTime,
			Status:     models.StatusScheduled,
			Source:     models.SourceRecurring,
			ImportHash: hash,
			Notes:      r.Notes,
		}
		if dbErr := CreateAssignment(db, a); dbErr != nil {
			return created, skipped, dbErr
		}
		created++
	}
	return created, skipped, nil
}
