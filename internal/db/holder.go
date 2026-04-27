package db

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
)

// Holder owns the live *sql.DB connection pool and allows it to be replaced
// at runtime (e.g. when a backup is restored via the UI). Handlers and
// middleware go through Holder.DB() instead of holding *sql.DB directly so
// the pool can be swapped without restarting the server — this is also the
// natural seam for a future per-tenant DB lookup.
type Holder struct {
	mu   sync.RWMutex
	db   *sql.DB
	path string
}

// NewHolder opens the DB at path and returns a Holder owning it.
func NewHolder(path string) (*Holder, error) {
	d, err := Open(path)
	if err != nil {
		return nil, err
	}
	return &Holder{db: d, path: path}, nil
}

// DB returns the current connection pool. The returned pointer is only safe
// until the next Swap; callers should fetch it fresh per request rather than
// caching it. Concurrent reads are fine.
func (h *Holder) DB() *sql.DB {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.db
}

// Path returns the on-disk DB file path.
func (h *Holder) Path() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.path
}

// Close closes the underlying DB pool. Holder is not usable afterwards.
func (h *Holder) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.db == nil {
		return nil
	}
	err := h.db.Close()
	h.db = nil
	return err
}

// Swap atomically replaces the current DB file with src and reopens the pool.
// On success the old pool is closed and any subsequent DB() call returns the
// new one. In-flight queries against the old pool may fail with "database is
// closed" — Restore is a deliberate, user-initiated action where this is
// acceptable; the UI redirects to the setup screen right after.
//
// src is renamed (moved), not copied — the caller should write the staged
// backup to a path on the same filesystem as h.path.
func (h *Holder) Swap(src string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.db != nil {
		if err := h.db.Close(); err != nil {
			return fmt.Errorf("close current db: %w", err)
		}
		h.db = nil
	}

	// Orphan WAL/SHM from the closed pool — leaving them around can confuse
	// SQLite when it opens the freshly-renamed file (different inode).
	_ = os.Remove(h.path + "-wal")
	_ = os.Remove(h.path + "-shm")

	if err := os.Rename(src, h.path); err != nil {
		return fmt.Errorf("replace db file: %w", err)
	}

	newDB, err := Open(h.path)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	h.db = newDB
	return nil
}
