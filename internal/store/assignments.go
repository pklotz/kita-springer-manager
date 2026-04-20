package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pak/kita-springer-manager/internal/models"
)

const assignmentSelect = `
	SELECT a.id,
	       COALESCE(a.kita_id,''), COALESCE(a.provider_id,''),
	       COALESCE(a.group_name,''), a.date, a.start_time, a.end_time,
	       COALESCE(a.actual_start_time,''), COALESCE(a.actual_end_time,''),
	       COALESCE(a.status,'scheduled'), COALESCE(a.source,'manual'),
	       COALESCE(a.import_hash,''), COALESCE(a.notes,''), a.created_at,
	       COALESCE(k.name,''), COALESCE(k.address,''), COALESCE(k.stop_name,''),
	       COALESCE(p.name,''), COALESCE(p.color_hex,'')
	FROM assignments a
	LEFT JOIN kitas    k ON k.id = a.kita_id
	LEFT JOIN providers p ON p.id = a.provider_id`

func ListAssignments(db *sql.DB, from, to string) ([]models.Assignment, error) {
	query := assignmentSelect + " WHERE 1=1"
	args := []any{}
	if from != "" {
		query += " AND a.date >= ?"
		args = append(args, from)
	}
	if to != "" {
		query += " AND a.date <= ?"
		args = append(args, to)
	}
	query += " ORDER BY a.date, a.start_time"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssignments(rows)
}

func GetAssignment(db *sql.DB, id string) (*models.Assignment, error) {
	rows, err := db.Query(assignmentSelect+" WHERE a.id=?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanAssignments(rows)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

func scanAssignments(rows *sql.Rows) ([]models.Assignment, error) {
	var out []models.Assignment
	for rows.Next() {
		var a models.Assignment
		a.Kita = &models.Kita{}
		a.Provider = &models.Provider{}
		if err := rows.Scan(
			&a.ID, &a.KitaID, &a.ProviderID,
			&a.GroupName, &a.Date, &a.StartTime, &a.EndTime,
			&a.ActualStartTime, &a.ActualEndTime,
			&a.Status, &a.Source, &a.ImportHash, &a.Notes, &a.CreatedAt,
			&a.Kita.Name, &a.Kita.Address, &a.Kita.StopName,
			&a.Provider.Name, &a.Provider.ColorHex,
		); err != nil {
			return nil, err
		}
		a.Kita.ID = a.KitaID
		a.Provider.ID = a.ProviderID
		out = append(out, a)
	}
	return out, rows.Err()
}

// ConflictReason identifies why a proposed assignment clashes with an existing one.
type ConflictReason string

const (
	ConflictSameKita ConflictReason = "same_kita" // same Kita on the same day
	ConflictOverlap  ConflictReason = "overlap"   // different Kita, overlapping time window
)

// FindAssignmentConflict returns the first scheduled assignment on the same date
// that conflicts with a — either because it is the same Kita, or because it is a
// different Kita with an overlapping time window. Assignments with no times are
// treated as spanning the whole day. Free/absence markers are ignored.
// excludeID lets callers skip a specific row (the one being updated).
func FindAssignmentConflict(db *sql.DB, a *models.Assignment, excludeID string) (*models.Assignment, ConflictReason, error) {
	if a.Date == "" || a.Status == models.StatusFree {
		return nil, "", nil
	}
	query := assignmentSelect + ` WHERE a.date = ? AND COALESCE(a.status,'scheduled') = 'scheduled'`
	args := []any{a.Date}
	if excludeID != "" {
		query += " AND a.id != ?"
		args = append(args, excludeID)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	candidates, err := scanAssignments(rows)
	if err != nil {
		return nil, "", err
	}

	for i := range candidates {
		c := &candidates[i]
		if a.KitaID != "" && c.KitaID == a.KitaID {
			return c, ConflictSameKita, nil
		}
		if timesOverlap(a.StartTime, a.EndTime, c.StartTime, c.EndTime) {
			return c, ConflictOverlap, nil
		}
	}
	return nil, "", nil
}

// timesOverlap returns true if [aStart,aEnd) intersects [bStart,bEnd).
// Missing start or end on either side is treated as a full-day window.
func timesOverlap(aStart, aEnd, bStart, bEnd string) bool {
	if aStart == "" || aEnd == "" || bStart == "" || bEnd == "" {
		return true
	}
	return aStart < bEnd && bStart < aEnd
}

func FindAssignmentByHash(db *sql.DB, hash string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM assignments WHERE import_hash=?`, hash).Scan(&count)
	return count > 0, err
}

func CreateAssignment(db *sql.DB, a *models.Assignment) error {
	a.ID = uuid.New().String()
	a.CreatedAt = time.Now()
	if a.Status == "" {
		a.Status = models.StatusScheduled
	}
	if a.Source == "" {
		a.Source = models.SourceManual
	}
	kitaID := sql.NullString{String: a.KitaID, Valid: a.KitaID != ""}
	providerID := sql.NullString{String: a.ProviderID, Valid: a.ProviderID != ""}
	_, err := db.Exec(
		`INSERT INTO assignments (id, kita_id, provider_id, group_name, date, start_time, end_time, actual_start_time, actual_end_time, status, source, import_hash, notes, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, kitaID, providerID, a.GroupName, a.Date, a.StartTime, a.EndTime,
		a.ActualStartTime, a.ActualEndTime,
		a.Status, a.Source, a.ImportHash, a.Notes, a.CreatedAt,
	)
	return err
}

func UpdateAssignment(db *sql.DB, a *models.Assignment) error {
	kitaID := sql.NullString{String: a.KitaID, Valid: a.KitaID != ""}
	providerID := sql.NullString{String: a.ProviderID, Valid: a.ProviderID != ""}
	_, err := db.Exec(
		`UPDATE assignments SET kita_id=?, provider_id=?, group_name=?, date=?, start_time=?, end_time=?,
		 actual_start_time=?, actual_end_time=?, status=?, notes=? WHERE id=?`,
		kitaID, providerID, a.GroupName, a.Date, a.StartTime, a.EndTime,
		a.ActualStartTime, a.ActualEndTime, a.Status, a.Notes, a.ID,
	)
	return err
}

func UpsertByHash(db *sql.DB, a *models.Assignment) (created bool, err error) {
	var existingID string
	err = db.QueryRow(`SELECT id FROM assignments WHERE import_hash=?`, a.ImportHash).Scan(&existingID)
	if err == sql.ErrNoRows {
		return true, CreateAssignment(db, a)
	}
	if err != nil {
		return false, err
	}
	a.ID = existingID
	return false, UpdateAssignment(db, a)
}

func DeleteAssignment(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM assignments WHERE id=?`, id)
	return err
}

// BulkDeleteAssignments removes many assignments in a single transaction.
// Returns the number of rows actually deleted.
func BulkDeleteAssignments(db *sql.DB, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]byte, 0, 2*len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
		args[i] = id
	}
	res, err := db.Exec(
		`DELETE FROM assignments WHERE id IN (`+string(placeholders)+`)`,
		args...,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// CleanupPastConnections removes cached transit connections for assignments
// whose date is before today. Past assignments stay (for Historie/Arbeitszeiten)
// but their ÖV-data is irrelevant.
func CleanupPastConnections(db *sql.DB, today string) (int64, error) {
	res, err := db.Exec(
		`DELETE FROM cached_connections
		 WHERE assignment_id IN (SELECT id FROM assignments WHERE date < ?)`,
		today,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
