package store

import (
	"database/sql"
	"encoding/json"
	"strconv"

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
	s.UserName = kv["user_name"]
	if v, err := strconv.ParseFloat(kv["home_lat"], 64); err == nil {
		s.HomeLat = v
	}
	if v, err := strconv.ParseFloat(kv["home_lng"], 64); err == nil {
		s.HomeLng = v
	}
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

	latStr, lngStr := "", ""
	if s.HomeLat != 0 || s.HomeLng != 0 {
		latStr = strconv.FormatFloat(s.HomeLat, 'f', 6, 64)
		lngStr = strconv.FormatFloat(s.HomeLng, 'f', 6, 64)
	}

	for _, kv := range [][2]string{
		{"home_address", s.HomeAddress},
		{"home_stop", s.HomeStop},
		{"home_lat", latStr},
		{"home_lng", lngStr},
		{"user_name", s.UserName},
		{"transit_prefs", string(prefsJSON)},
	} {
		if err := upsert(kv[0], kv[1]); err != nil {
			return err
		}
	}
	return tx.Commit()
}
