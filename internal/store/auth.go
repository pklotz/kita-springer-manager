package store

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	authUsernameKey      = "auth_username"
	authPasswordHashKey  = "auth_password_hash"
	authDownloadTokenKey = "auth_download_token"
	defaultAuthUsername  = "admin"
	minPasswordLength    = 8
)

var ErrPasswordTooShort = errors.New("Passwort muss mindestens 8 Zeichen haben")

// IsAuthConfigured reports whether a password hash is stored. While false, the
// server runs in setup mode and the basic-auth middleware lets the setup
// endpoint through without credentials.
func IsAuthConfigured(db *sql.DB) (bool, error) {
	hash, err := getSetting(db, authPasswordHashKey)
	if err != nil {
		return false, err
	}
	return hash != "", nil
}

// GetAuthUsername returns the configured admin username (default "admin").
func GetAuthUsername(db *sql.DB) (string, error) {
	user, err := getSetting(db, authUsernameKey)
	if err != nil {
		return "", err
	}
	if user == "" {
		return defaultAuthUsername, nil
	}
	return user, nil
}

// SetAuthCredentials hashes the password with bcrypt and writes both username
// and hash atomically. An empty username falls back to "admin". A download
// token is also generated on first setup so iCal/PDF subscription URLs work
// without exposing the password.
func SetAuthCredentials(db *sql.DB, username, password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}
	if username == "" {
		username = defaultAuthUsername
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		authUsernameKey, username,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		authPasswordHashKey, string(hash),
	); err != nil {
		return err
	}
	// Only generate the download token once; rotating the password should not
	// invalidate existing calendar subscriptions.
	var existingToken string
	if err := tx.QueryRow(`SELECT value FROM settings WHERE key=?`, authDownloadTokenKey).Scan(&existingToken); err == sql.ErrNoRows || existingToken == "" {
		tok, terr := newDownloadToken()
		if terr != nil {
			return terr
		}
		if _, err := tx.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			authDownloadTokenKey, tok,
		); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return tx.Commit()
}

// GetDownloadToken returns the long-random URL-token used to authenticate
// calendar/PDF subscriptions without basic-auth. Empty string if not yet
// configured (e.g. legacy DB before this feature).
func GetDownloadToken(db *sql.DB) (string, error) {
	return getSetting(db, authDownloadTokenKey)
}

// RegenerateDownloadToken issues a fresh token, invalidating all existing
// subscription URLs.
func RegenerateDownloadToken(db *sql.DB) (string, error) {
	tok, err := newDownloadToken()
	if err != nil {
		return "", err
	}
	_, err = db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		authDownloadTokenKey, tok,
	)
	if err != nil {
		return "", err
	}
	return tok, nil
}

// VerifyDownloadToken constant-time compares the supplied token against the
// stored one. Returns false (without error) if no token is configured yet.
func VerifyDownloadToken(db *sql.DB, supplied string) bool {
	stored, err := getSetting(db, authDownloadTokenKey)
	if err != nil || stored == "" || supplied == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(supplied)) == 1
}

func newDownloadToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// VerifyAuth returns true iff the supplied basic-auth credentials match the
// stored hash. Username comparison uses constant time to avoid trivial
// enumeration; bcrypt itself is constant-time on the secret.
func VerifyAuth(db *sql.DB, username, password string) bool {
	storedUser, err := GetAuthUsername(db)
	if err != nil {
		return false
	}
	storedHash, err := getSetting(db, authPasswordHashKey)
	if err != nil || storedHash == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(storedUser), []byte(username)) != 1 {
		// Still run bcrypt to keep timing similar across user/hash mismatches.
		_ = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}

func getSetting(db *sql.DB, key string) (string, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}
