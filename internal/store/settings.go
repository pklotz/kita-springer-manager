package store

import (
	"database/sql"
	"encoding/json"

	"github.com/pak/kita-springer-manager/internal/models"
)

func GetSettings(db *sql.DB) (*models.Settings, error) {
	rows, err := db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kv := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		kv[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s := &models.Settings{}
	s.HomeAddress = kv["home_address"]
	s.HomeStop = kv["home_stop"]
	if v, ok := kv["transit_prefs"]; ok && v != "" {
		json.Unmarshal([]byte(v), &s.TransitPrefs) //nolint:errcheck
	}
	return s, nil
}

func SaveSettings(db *sql.DB, s *models.Settings) error {
	prefsJSON, _ := json.Marshal(s.TransitPrefs)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	upsert := func(key, value string) error {
		_, err := tx.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			key, value,
		)
		return err
	}

	for _, kv := range [][2]string{
		{"home_address", s.HomeAddress},
		{"home_stop", s.HomeStop},
		{"transit_prefs", string(prefsJSON)},
	} {
		if err := upsert(kv[0], kv[1]); err != nil {
			return err
		}
	}
	return tx.Commit()
}
