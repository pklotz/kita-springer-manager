package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pak/kita-springer-manager/internal/models"
)

const kitaCols = `id, COALESCE(provider_id,''), name, address, stop_name,
	COALESCE(phone,''), COALESCE(email,''), COALESCE(leitung_name,''), COALESCE(photo_url,''),
	COALESCE(groups,'[]'), lat, lng, notes, created_at`

func ListKitas(db *sql.DB) ([]models.Kita, error) {
	rows, err := db.Query(`SELECT ` + kitaCols + ` FROM kitas ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKitas(rows)
}

func ListKitasByProvider(db *sql.DB, providerID string) ([]models.Kita, error) {
	rows, err := db.Query(`SELECT `+kitaCols+` FROM kitas WHERE provider_id=? ORDER BY name`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKitas(rows)
}

func scanKitas(rows *sql.Rows) ([]models.Kita, error) {
	var kitas []models.Kita
	for rows.Next() {
		var k models.Kita
		var groupsJSON string
		if err := rows.Scan(
			&k.ID, &k.ProviderID, &k.Name, &k.Address, &k.StopName,
			&k.Phone, &k.Email, &k.LeitungName, &k.PhotoURL,
			&groupsJSON, &k.Lat, &k.Lng, &k.Notes, &k.CreatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(groupsJSON), &k.Groups) //nolint:errcheck
		if k.Groups == nil {
			k.Groups = []string{}
		}
		kitas = append(kitas, k)
	}
	return kitas, rows.Err()
}

func GetKita(db *sql.DB, id string) (*models.Kita, error) {
	var k models.Kita
	var groupsJSON string
	err := db.QueryRow(`SELECT `+kitaCols+` FROM kitas WHERE id=?`, id).
		Scan(&k.ID, &k.ProviderID, &k.Name, &k.Address, &k.StopName,
			&k.Phone, &k.Email, &k.LeitungName, &k.PhotoURL,
			&groupsJSON, &k.Lat, &k.Lng, &k.Notes, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(groupsJSON), &k.Groups) //nolint:errcheck
	if k.Groups == nil {
		k.Groups = []string{}
	}
	return &k, nil
}

func CreateKita(db *sql.DB, k *models.Kita) error {
	k.ID = uuid.New().String()
	k.CreatedAt = time.Now()
	groups, _ := json.Marshal(k.Groups)
	providerID := sql.NullString{String: k.ProviderID, Valid: k.ProviderID != ""}
	_, err := db.Exec(
		`INSERT INTO kitas (id, provider_id, name, address, stop_name, phone, email, leitung_name, photo_url, groups, lat, lng, notes, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		k.ID, providerID, k.Name, k.Address, k.StopName, k.Phone, k.Email, k.LeitungName, k.PhotoURL,
		string(groups), k.Lat, k.Lng, k.Notes, k.CreatedAt,
	)
	return err
}

func UpdateKita(db *sql.DB, k *models.Kita) error {
	groups, _ := json.Marshal(k.Groups)
	providerID := sql.NullString{String: k.ProviderID, Valid: k.ProviderID != ""}
	_, err := db.Exec(
		`UPDATE kitas SET provider_id=?, name=?, address=?, stop_name=?, phone=?, email=?, leitung_name=?, photo_url=?, groups=?, lat=?, lng=?, notes=? WHERE id=?`,
		providerID, k.Name, k.Address, k.StopName, k.Phone, k.Email, k.LeitungName, k.PhotoURL,
		string(groups), k.Lat, k.Lng, k.Notes, k.ID,
	)
	return err
}

func DeleteKita(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM kitas WHERE id=?`, id)
	return err
}
