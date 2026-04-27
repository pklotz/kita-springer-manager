package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
)

// SQLiteMagic is the 16-byte header every valid SQLite 3 file starts with
// (https://www.sqlite.org/fileformat.html#magic_header_string).
var SQLiteMagic = []byte("SQLite format 3\x00")

// ValidateBackup opens path read-only and verifies it's a usable backup of
// this app: SQLite magic bytes are present, PRAGMA user_version is non-zero
// (= migrations have been applied) and the settings table exists.
//
// Returns nil iff the file looks like a kita-springer DB. Used by the web
// restore endpoint and the local backup CLI.
func ValidateBackup(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	magic := make([]byte, len(SQLiteMagic))
	if _, err := io.ReadFull(f, magic); err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	if !bytes.Equal(magic, SQLiteMagic) {
		return fmt.Errorf("not a SQLite database (magic mismatch)")
	}

	conn, err := sql.Open("sqlite", path+"?mode=ro&_foreign_keys=off")
	if err != nil {
		return fmt.Errorf("sqlite open: %w", err)
	}
	defer conn.Close()

	var version int
	if err := conn.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("user_version: %w", err)
	}
	if version == 0 {
		return fmt.Errorf("user_version is 0 — no migrations applied, not a kita-springer backup")
	}

	var n int
	if err := conn.QueryRow("SELECT count(*) FROM settings").Scan(&n); err != nil {
		return fmt.Errorf("settings table missing: %w", err)
	}
	return nil
}
