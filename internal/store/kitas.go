package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pak/kita-springer-manager/internal/models"
)

const kitaCols = `id, COALESCE(provider_id,''), name, address, stop_name,
	COALESCE(stops,'[]'),
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
		var groupsJSON, stopsJSON string
		if err := rows.Scan(
			&k.ID, &k.ProviderID, &k.Name, &k.Address, &k.StopName, &stopsJSON,
			&k.Phone, &k.Email, &k.LeitungName, &k.PhotoURL,
			&groupsJSON, &k.Lat, &k.Lng, &k.Notes, &k.CreatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(groupsJSON), &k.Groups) //nolint:errcheck
		json.Unmarshal([]byte(stopsJSON), &k.Stops)   //nolint:errcheck
		if k.Groups == nil {
			k.Groups = []string{}
		}
		if k.Stops == nil {
			k.Stops = []string{}
		}
		kitas = append(kitas, k)
	}
	return kitas, rows.Err()
}

func GetKita(db *sql.DB, id string) (*models.Kita, error) {
	var k models.Kita
	var groupsJSON, stopsJSON string
	err := db.QueryRow(`SELECT `+kitaCols+` FROM kitas WHERE id=?`, id).
		Scan(&k.ID, &k.ProviderID, &k.Name, &k.Address, &k.StopName, &stopsJSON,
			&k.Phone, &k.Email, &k.LeitungName, &k.PhotoURL,
			&groupsJSON, &k.Lat, &k.Lng, &k.Notes, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(groupsJSON), &k.Groups) //nolint:errcheck
	json.Unmarshal([]byte(stopsJSON), &k.Stops)   //nolint:errcheck
	if k.Groups == nil {
		k.Groups = []string{}
	}
	if k.Stops == nil {
		k.Stops = []string{}
	}
	return &k, nil
}

// normalizeStops keeps Stops and StopName in sync: Stops[0] is the primary.
// If only StopName is set, promote it to Stops[0].
func normalizeStops(k *models.Kita) {
	cleaned := make([]string, 0, len(k.Stops))
	seen := map[string]bool{}
	for _, s := range k.Stops {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		cleaned = append(cleaned, s)
	}
	k.Stops = cleaned
	if len(k.Stops) == 0 && k.StopName != "" {
		k.Stops = []string{k.StopName}
	}
	if len(k.Stops) > 0 {
		k.StopName = k.Stops[0]
	} else {
		k.StopName = ""
	}
}

func CreateKita(db *sql.DB, k *models.Kita) error {
	k.ID = uuid.New().String()
	k.CreatedAt = time.Now()
	normalizeStops(k)
	groups, _ := json.Marshal(k.Groups)
	stops, _ := json.Marshal(k.Stops)
	providerID := sql.NullString{String: k.ProviderID, Valid: k.ProviderID != ""}
	_, err := db.Exec(
		`INSERT INTO kitas (id, provider_id, name, address, stop_name, stops, phone, email, leitung_name, photo_url, groups, lat, lng, notes, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		k.ID, providerID, k.Name, k.Address, k.StopName, string(stops),
		k.Phone, k.Email, k.LeitungName, k.PhotoURL,
		string(groups), k.Lat, k.Lng, k.Notes, k.CreatedAt,
	)
	return err
}

func UpdateKita(db *sql.DB, k *models.Kita) error {
	normalizeStops(k)
	groups, _ := json.Marshal(k.Groups)
	stops, _ := json.Marshal(k.Stops)
	providerID := sql.NullString{String: k.ProviderID, Valid: k.ProviderID != ""}
	_, err := db.Exec(
		`UPDATE kitas SET provider_id=?, name=?, address=?, stop_name=?, stops=?, phone=?, email=?, leitung_name=?, photo_url=?, groups=?, lat=?, lng=?, notes=? WHERE id=?`,
		providerID, k.Name, k.Address, k.StopName, string(stops),
		k.Phone, k.Email, k.LeitungName, k.PhotoURL,
		string(groups), k.Lat, k.Lng, k.Notes, k.ID,
	)
	return err
}

func DeleteKita(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM kitas WHERE id=?`, id)
	return err
}
