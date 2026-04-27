// Package middleware contains HTTP middlewares specific to this app.
// Imported as `apimw` to avoid collision with go-chi/chi/v5/middleware.
package middleware

import (
	"net/http"
	"strings"

	"github.com/pak/kita-springer-manager/internal/db"
	"github.com/pak/kita-springer-manager/internal/store"
)

// downloadTokenPaths are GET endpoints that accept ?token=<download-token>
// as an alternative to basic-auth so external clients (Apple Calendar,
// browsers without our login dialog) can subscribe/download.
var downloadTokenPaths = map[string]bool{
	"/api/calendar.ics":    true,
	"/api/worktime/export": true,
}

// publicAPIPaths are /api/* endpoints that must be reachable without auth
// for the SPA to bootstrap (status query) or to break out of a stuck
// state (reset, logout).
var publicAPIPaths = map[string]bool{
	"/api/auth/status": true,
	"/api/auth/logout": true,
	"/api/auth/reset":  true,
}

// BasicAuth gates only /api/* routes. The static SPA shell (index.html, JS,
// CSS, sw.js, manifest, icons) is served without auth — the bundle alone has
// no data; it bootstraps by calling /api/auth/status and rendering either
// the login or the setup screen. Protecting the static shell also prevented
// stuck Service Workers from ever fetching a new /sw.js, because their
// background update fetch had no credentials.
//
// On failure we return a plain 401 WITHOUT a WWW-Authenticate header — the
// SPA renders its own login dialog. External clients (iOS Calendar) embed
// credentials in the URL or use the download token, so they don't need a
// challenge either.
func BasicAuth(holder *db.Holder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Static assets and SPA shell: always public.
			if !strings.HasPrefix(path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Always-open API endpoints (status / logout / reset).
			if publicAPIPaths[path] {
				next.ServeHTTP(w, r)
				return
			}

			// Resolve the live DB once per request; Restore may swap it.
			conn := holder.DB()

			configured, err := store.IsAuthConfigured(conn)
			if err != nil {
				http.Error(w, "Server-Fehler", http.StatusInternalServerError)
				return
			}
			if !configured {
				// Setup mode: API is fully open until the first password is set.
				next.ServeHTTP(w, r)
				return
			}

			// Download-token bypass for subscription URLs (Apple Calendar etc.).
			if r.Method == http.MethodGet && downloadTokenPaths[path] {
				if tok := r.URL.Query().Get("token"); tok != "" && store.VerifyDownloadToken(conn, tok) {
					next.ServeHTTP(w, r)
					return
				}
			}

			user, pass, ok := r.BasicAuth()
			if !ok || !store.VerifyAuth(conn, user, pass) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
