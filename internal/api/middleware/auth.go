// Package middleware contains HTTP middlewares specific to this app.
// Imported as `apimw` to avoid collision with go-chi/chi/v5/middleware.
package middleware

import (
	"database/sql"
	"net/http"

	"github.com/pak/kita-springer-manager/internal/store"
)

// downloadTokenPaths are GET endpoints that accept ?token=<download-token>
// as an alternative to basic-auth so external clients (Apple Calendar,
// browsers without our login dialog) can subscribe/download.
var downloadTokenPaths = map[string]bool{
	"/api/calendar.ics":     true,
	"/api/worktime/export":  true,
}

// BasicAuth requires HTTP Basic Auth on every request, except:
//   - the auth-status and -logout endpoints (always public);
//   - everything while no password is configured (setup mode);
//   - download endpoints called with a valid ?token=... parameter.
//
// On failure we return a plain 401 WITHOUT a WWW-Authenticate header — the
// SPA renders its own login dialog. External clients (iOS Calendar) embed
// credentials in the URL or use the download token, so they don't need a
// challenge either.
func BasicAuth(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			if path == "/api/auth/status" || path == "/api/auth/logout" {
				next.ServeHTTP(w, r)
				return
			}

			configured, err := store.IsAuthConfigured(db)
			if err != nil {
				http.Error(w, "Server-Fehler", http.StatusInternalServerError)
				return
			}
			if !configured {
				next.ServeHTTP(w, r)
				return
			}

			// Download-token bypass for subscription URLs.
			if r.Method == http.MethodGet && downloadTokenPaths[path] {
				if tok := r.URL.Query().Get("token"); tok != "" && store.VerifyDownloadToken(db, tok) {
					next.ServeHTTP(w, r)
					return
				}
			}

			user, pass, ok := r.BasicAuth()
			if !ok || !store.VerifyAuth(db, user, pass) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
