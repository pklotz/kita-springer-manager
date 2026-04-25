package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// Each slice entry is one migration version (1-based index).
var migrations = [][]string{
	// v1: initial schema
	{
		`CREATE TABLE IF NOT EXISTS kitas (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			address    TEXT NOT NULL DEFAULT '',
			stop_name  TEXT NOT NULL DEFAULT '',
			lat        REAL,
			lng        REAL,
			notes      TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS assignments (
			id         TEXT PRIMARY KEY,
			kita_id    TEXT REFERENCES kitas(id) ON DELETE SET NULL,
			date       TEXT NOT NULL,
			start_time TEXT NOT NULL DEFAULT '',
			end_time   TEXT NOT NULL DEFAULT '',
			notes      TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS cached_connections (
			id              TEXT PRIMARY KEY,
			assignment_id   TEXT NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
			direction       TEXT NOT NULL DEFAULT 'outbound',
			departure_time  TEXT NOT NULL,
			connection_json TEXT NOT NULL,
			cached_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_assignments_date ON assignments(date)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_connections_assignment ON cached_connections(assignment_id)`,
	},
	// v2: providers, extended kitas/assignments, recurring
	{
		`CREATE TABLE IF NOT EXISTS providers (
			id           TEXT PRIMARY KEY,
			name         TEXT NOT NULL,
			color_hex    TEXT NOT NULL DEFAULT '#6366f1',
			notes        TEXT NOT NULL DEFAULT '',
			excel_config TEXT NOT NULL DEFAULT '{}',
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE kitas ADD COLUMN provider_id TEXT REFERENCES providers(id) ON DELETE SET NULL`,
		`ALTER TABLE kitas ADD COLUMN groups      TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE kitas ADD COLUMN phone       TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE kitas ADD COLUMN email       TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE assignments ADD COLUMN provider_id  TEXT REFERENCES providers(id) ON DELETE SET NULL`,
		`ALTER TABLE assignments ADD COLUMN group_name   TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE assignments ADD COLUMN status       TEXT NOT NULL DEFAULT 'scheduled'`,
		`ALTER TABLE assignments ADD COLUMN source       TEXT NOT NULL DEFAULT 'manual'`,
		`ALTER TABLE assignments ADD COLUMN import_hash  TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS recurring_assignments (
			id          TEXT PRIMARY KEY,
			kita_id     TEXT REFERENCES kitas(id) ON DELETE SET NULL,
			provider_id TEXT REFERENCES providers(id) ON DELETE CASCADE,
			group_name  TEXT NOT NULL DEFAULT '',
			day_of_week INTEGER NOT NULL,
			start_time  TEXT NOT NULL DEFAULT '',
			end_time    TEXT NOT NULL DEFAULT '',
			valid_from  TEXT NOT NULL,
			valid_until TEXT NOT NULL,
			notes       TEXT NOT NULL DEFAULT '',
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_assignments_provider ON assignments(provider_id)`,
		`CREATE INDEX IF NOT EXISTS idx_assignments_status   ON assignments(status)`,
	},
	// v3: leitung_name and photo_url on kitas
	{
		`ALTER TABLE kitas ADD COLUMN leitung_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE kitas ADD COLUMN photo_url    TEXT NOT NULL DEFAULT ''`,
	},
	// v4: closures (holidays, vacation, provider/kita closure days)
	{
		`CREATE TABLE IF NOT EXISTS closures (
			id           TEXT PRIMARY KEY,
			type         TEXT NOT NULL DEFAULT 'springerin',
			reference_id TEXT,
			date         TEXT NOT NULL,
			note         TEXT NOT NULL DEFAULT '',
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_closures_date ON closures(date)`,
		`CREATE INDEX IF NOT EXISTS idx_closures_type ON closures(type)`,
	},
	// v5: dedup guarantees via unique indexes
	{
		// Remove any pre-existing duplicates before adding UNIQUE constraints
		`DELETE FROM assignments WHERE import_hash != '' AND rowid NOT IN (
			SELECT MIN(rowid) FROM assignments WHERE import_hash != '' GROUP BY import_hash
		)`,
		`DELETE FROM closures WHERE rowid NOT IN (
			SELECT MIN(rowid) FROM closures GROUP BY type, date, COALESCE(reference_id, '')
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uidx_assignments_import_hash
			ON assignments(import_hash) WHERE import_hash != ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uidx_closures_type_date_ref
			ON closures(type, date, COALESCE(reference_id, ''))`,
	},
	// v6: actual worked hours (in addition to planned start_time/end_time)
	{
		`ALTER TABLE assignments ADD COLUMN actual_start_time TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE assignments ADD COLUMN actual_end_time   TEXT NOT NULL DEFAULT ''`,
	},
	// v7: multi-stop support on kitas (stops JSON array, backfilled from stop_name)
	{
		`ALTER TABLE kitas ADD COLUMN stops TEXT NOT NULL DEFAULT '[]'`,
		`UPDATE kitas SET stops = json_array(stop_name) WHERE stops = '[]' AND stop_name != ''`,
	},
	// v8: break tracking on assignments, min-break default on providers
	{
		`ALTER TABLE assignments ADD COLUMN actual_break_start TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE assignments ADD COLUMN actual_break_end   TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE providers   ADD COLUMN min_break_minutes  INTEGER NOT NULL DEFAULT 30`,
	},
	// v9: drop old holiday seed so it can be repopulated per-canton by rickar/cal
	{
		`DELETE FROM closures WHERE type='holiday'`,
	},
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	// Wait up to 5s for the writer lock instead of failing with "database is locked".
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return nil, fmt.Errorf("busy_timeout: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	var version int
	db.QueryRow("PRAGMA user_version").Scan(&version) //nolint:errcheck

	for i := version; i < len(migrations); i++ {
		for _, stmt := range migrations[i] {
			if _, err := db.Exec(stmt); err != nil {
				if isDuplicateColumn(err) {
					continue
				}
				return fmt.Errorf("migration v%d: %w\nSQL: %s", i+1, err, stmt)
			}
		}
		if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			return err
		}
	}
	return nil
}

func isDuplicateColumn(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "already exists")
}
