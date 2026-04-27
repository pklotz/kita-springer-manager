package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pak/kita-springer-manager/internal/audit"
	"github.com/pak/kita-springer-manager/internal/db"
)

// maxRestoreUploadBytes caps the upload size at 50 MiB. The DB is just KV
// settings + assignments + transit cache for one user — anything larger is
// almost certainly the wrong file.
const maxRestoreUploadBytes = 50 << 20

// ExportBackup streams a clean single-file SQLite snapshot of the live DB as
// a download. Uses VACUUM INTO so the result is consistent and free of WAL
// sidecars — a single file the user can save and later upload to /api/restore.
func (h *Handler) ExportBackup(w http.ResponseWriter, r *http.Request) {
	dbPath := h.holderRef().Path()
	dir := filepath.Dir(dbPath)

	tmp, err := os.CreateTemp(dir, "backup-*.db")
	if err != nil {
		serverError(w, fmt.Errorf("create temp: %w", err))
		return
	}
	tmpPath := tmp.Name()
	// Close the FD; SQLite needs to open the path itself for VACUUM INTO.
	_ = tmp.Close()
	// VACUUM INTO refuses to write to an existing file.
	_ = os.Remove(tmpPath)
	defer os.Remove(tmpPath) //nolint:errcheck

	if _, err := h.db().ExecContext(r.Context(), "VACUUM INTO ?", tmpPath); err != nil {
		serverError(w, fmt.Errorf("vacuum into: %w", err))
		return
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		serverError(w, fmt.Errorf("open backup: %w", err))
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		serverError(w, fmt.Errorf("stat backup: %w", err))
		return
	}

	filename := fmt.Sprintf("kita-springer-%s.db", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("Cache-Control", "no-store")

	audit.L().Info("backup.export", "size_bytes", stat.Size(), "filename", filename)

	if _, err := io.Copy(w, f); err != nil {
		// Client disconnect — log and move on; headers are already flushed.
		audit.L().Warn("backup.export.stream", "err", err.Error())
	}
}

// RestoreBackup accepts a SQLite-file upload, validates it, replaces the live
// DB and clears the auth password so the next request lands in setup mode.
// Form field name: "file".
func (h *Handler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxRestoreUploadBytes); err != nil {
		writeError(w, 400, "Upload zu groß oder ungültig (max 50 MB)")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "Feld 'file' fehlt")
		return
	}
	defer file.Close()

	// Magic-byte check upfront — fail fast on obviously-wrong uploads.
	magic := make([]byte, len(db.SQLiteMagic))
	if _, err := io.ReadFull(file, magic); err != nil || !bytes.Equal(magic, db.SQLiteMagic) {
		audit.L().Warn("backup.restore.magic_mismatch", "filename", header.Filename)
		writeError(w, 422, "Datei ist keine SQLite-Datenbank")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		serverError(w, fmt.Errorf("seek upload: %w", err))
		return
	}

	// Stage to a temp file on the same filesystem as the target — Holder.Swap
	// uses os.Rename which requires same-FS for atomicity.
	dbPath := h.holderRef().Path()
	dir := filepath.Dir(dbPath)
	staged, err := os.CreateTemp(dir, "restore-*.db")
	if err != nil {
		serverError(w, fmt.Errorf("stage temp: %w", err))
		return
	}
	stagedPath := staged.Name()
	cleanupStaged := true
	defer func() {
		if cleanupStaged {
			_ = os.Remove(stagedPath)
		}
	}()

	if _, err := io.Copy(staged, file); err != nil {
		_ = staged.Close()
		serverError(w, fmt.Errorf("write staged: %w", err))
		return
	}
	if err := staged.Close(); err != nil {
		serverError(w, fmt.Errorf("close staged: %w", err))
		return
	}

	// Validate by opening read-only and probing the schema. Catches truncated
	// uploads, files with the right magic but corrupt body, and DBs that
	// pre-date the auth feature (no settings table).
	if err := db.ValidateBackup(stagedPath); err != nil {
		audit.L().Warn("backup.restore.validate", "err", err.Error(), "filename", header.Filename)
		writeError(w, 422, "Datei ist keine gültige Backup-Datenbank")
		return
	}

	// Hot-swap. Old pool gets closed; the new one runs migrations on open so
	// older backups get upgraded to the current schema.
	if err := h.holderRef().Swap(stagedPath); err != nil {
		serverError(w, fmt.Errorf("swap db: %w", err))
		return
	}
	cleanupStaged = false // Swap renamed the file; nothing to clean up.

	// Wipe the password hash on the freshly-restored DB so the next request
	// hits the setup flow. Backups carry the original hash; we don't want
	// access to a leaked backup file to be access to the restored instance.
	if _, err := h.db().Exec(
		`UPDATE settings SET value='' WHERE key='auth_password_hash'`,
	); err != nil {
		// DB is restored but password reset failed. Log loudly — the operator
		// should set a new password manually before exposing the instance.
		audit.L().Error("backup.restore.password_reset", "err", err.Error())
	}

	audit.L().Info("backup.restore", "filename", header.Filename, "size_bytes", header.Size)
	writeJSON(w, 200, map[string]string{
		"status":  "ok",
		"message": "Datenbank wiederhergestellt. Bitte neues Passwort setzen.",
	})
}

