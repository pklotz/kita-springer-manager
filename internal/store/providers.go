package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pak/kita-springer-manager/internal/models"
)

func ListProviders(db *sql.DB) ([]models.Provider, error) {
	rows, err := db.Query(`SELECT id, name, color_hex, notes, excel_config, created_at FROM providers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []models.Provider
	for rows.Next() {
		var p models.Provider
		var cfg string
		if err := rows.Scan(&p.ID, &p.Name, &p.ColorHex, &p.Notes, &cfg, &p.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(cfg), &p.ExcelConfig) //nolint:errcheck
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func GetProvider(db *sql.DB, id string) (*models.Provider, error) {
	var p models.Provider
	var cfg string
	err := db.QueryRow(`SELECT id, name, color_hex, notes, excel_config, created_at FROM providers WHERE id=?`, id).
		Scan(&p.ID, &p.Name, &p.ColorHex, &p.Notes, &cfg, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(cfg), &p.ExcelConfig) //nolint:errcheck
	return &p, nil
}

func CreateProvider(db *sql.DB, p *models.Provider) error {
	p.ID = uuid.New().String()
	p.CreatedAt = time.Now()
	cfg, _ := json.Marshal(p.ExcelConfig)
	_, err := db.Exec(
		`INSERT INTO providers (id, name, color_hex, notes, excel_config, created_at) VALUES (?,?,?,?,?,?)`,
		p.ID, p.Name, p.ColorHex, p.Notes, string(cfg), p.CreatedAt,
	)
	return err
}

func UpdateProvider(db *sql.DB, p *models.Provider) error {
	cfg, _ := json.Marshal(p.ExcelConfig)
	_, err := db.Exec(
		`UPDATE providers SET name=?, color_hex=?, notes=?, excel_config=? WHERE id=?`,
		p.Name, p.ColorHex, p.Notes, string(cfg), p.ID,
	)
	return err
}

func DeleteProvider(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM providers WHERE id=?`, id)
	return err
}
