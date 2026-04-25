// Package audit writes a structured JSON-line log for every HTTP request and
// every server-emitted error to <data-dir>/audit.log. The log is append-only
// and intentionally simple: a single file, no rotation, no external deps.
// For a single-user self-hosted app the volume is negligible and grep-friendly
// JSON is more useful than rotated archives.
package audit

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var (
	once   sync.Once
	logger *slog.Logger
	file   *os.File
)

// Init opens (or creates) <dir>/audit.log next to the database and configures
// a JSON slog logger that writes to both the file and stderr. Idempotent —
// subsequent calls are no-ops.
func Init(dbPath string) error {
	var err error
	once.Do(func() {
		dir := filepath.Dir(dbPath)
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			err = mkErr
			return
		}
		path := filepath.Join(dir, "audit.log")
		f, openErr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if openErr != nil {
			err = openErr
			return
		}
		file = f
		// Tee: stderr keeps the existing console output, file is the audit trail.
		w := io.MultiWriter(os.Stderr, f)
		logger = slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	})
	return err
}

// L returns the configured audit logger, or slog.Default() if Init wasn't
// called yet (so handlers can safely log even before bootstrap completes).
func L() *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}

// Close flushes and closes the audit log file. Safe to call without Init.
func Close() error {
	if file != nil {
		return file.Close()
	}
	return nil
}
