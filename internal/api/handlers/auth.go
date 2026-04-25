package handlers

import (
	"errors"
	"net/http"

	"github.com/pak/kita-springer-manager/internal/store"
)

// GetAuthStatus reports whether a password has been set yet. Used by the
// frontend on bootstrap to decide between the setup screen and the app.
// This endpoint is unauthenticated.
func (h *Handler) GetAuthStatus(w http.ResponseWriter, r *http.Request) {
	configured, err := store.IsAuthConfigured(h.db)
	if err != nil {
		writeError(w, 500, "Server-Fehler")
		return
	}
	username := ""
	if configured {
		if u, err := store.GetAuthUsername(h.db); err == nil {
			username = u
		}
	}
	writeJSON(w, 200, map[string]any{
		"configured": configured,
		"username":   username,
	})
}

// SetupAuth sets the initial credentials. Only callable while no password is
// configured (the auth middleware lets it through in that state). After the
// first successful call, this endpoint requires basic-auth like every other.
func (h *Handler) SetupAuth(w http.ResponseWriter, r *http.Request) {
	configured, err := store.IsAuthConfigured(h.db)
	if err != nil {
		writeError(w, 500, "Server-Fehler")
		return
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, 400, "Ungültige Anfrage")
		return
	}

	// Once configured, this endpoint behaves like ChangePassword and requires
	// the caller to authenticate via Basic Auth (already enforced by the
	// middleware) and supply the current password as `username` field is
	// ignored.
	if configured {
		writeError(w, 409, "Bereits eingerichtet — bitte Passwort über die Einstellungen ändern")
		return
	}

	if err := store.SetAuthCredentials(h.db, body.Username, body.Password); err != nil {
		if errors.Is(err, store.ErrPasswordTooShort) {
			writeError(w, 400, err.Error())
			return
		}
		serverError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// Logout is a no-op on the server (Basic Auth is stateless) — the frontend
// just clears its localStorage token and reloads. This endpoint exists so a
// logout action shows up in the audit log.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// GetDownloadToken returns the current ?token=... value for subscription URLs
// (calendar.ics, worktime/export). Used by the Settings UI to render the
// webcal:// link.
func (h *Handler) GetDownloadToken(w http.ResponseWriter, r *http.Request) {
	tok, err := store.GetDownloadToken(h.db)
	if err != nil {
		writeError(w, 500, "Server-Fehler")
		return
	}
	if tok == "" {
		// Should only happen on legacy DBs that pre-date this feature;
		// generate one on demand so the UI always has something to show.
		tok, err = store.RegenerateDownloadToken(h.db)
		if err != nil {
			writeError(w, 500, "Server-Fehler")
			return
		}
	}
	writeJSON(w, 200, map[string]string{"token": tok})
}

// RegenerateDownloadToken rotates the token, invalidating any existing
// subscriptions. Caller is already authenticated via Basic-Auth middleware.
func (h *Handler) RegenerateDownloadToken(w http.ResponseWriter, r *http.Request) {
	tok, err := store.RegenerateDownloadToken(h.db)
	if err != nil {
		writeError(w, 500, "Server-Fehler")
		return
	}
	writeJSON(w, 200, map[string]string{"token": tok})
}

// ChangePassword updates the password. Caller is already authenticated by the
// middleware (Basic Auth), but we still require old_password in the body to
// guard against an unattended browser re-using cached credentials.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
		Username    string `json:"username"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, 400, "Ungültige Anfrage")
		return
	}

	user, err := store.GetAuthUsername(h.db)
	if err != nil {
		writeError(w, 500, "Server-Fehler")
		return
	}
	if !store.VerifyAuth(h.db, user, body.OldPassword) {
		writeError(w, 403, "Aktuelles Passwort falsch")
		return
	}
	if err := store.SetAuthCredentials(h.db, body.Username, body.NewPassword); err != nil {
		if errors.Is(err, store.ErrPasswordTooShort) {
			writeError(w, 400, err.Error())
			return
		}
		serverError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
