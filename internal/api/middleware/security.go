package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders sets a conservative set of response headers on every response.
// CSP allows external https images for kita photos and inline styles (Vue/Tailwind
// runtime emits a few). HSTS is only emitted when the request arrived over TLS
// (or via a reverse proxy that set X-Forwarded-Proto=https).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), interest-cohort=()")
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"img-src 'self' https: data:; "+
				"style-src 'self' 'unsafe-inline'; "+
				"script-src 'self'; "+
				"connect-src 'self'; "+
				"font-src 'self' data:; "+
				"frame-ancestors 'none'; "+
				"base-uri 'none'; "+
				"form-action 'self'")
		if isHTTPS(r) {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// isHTTPS reports whether the request reached the server over TLS, either
// directly or via a reverse proxy that set X-Forwarded-Proto.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// MaxBodyBytes wraps r.Body in http.MaxBytesReader so each handler can't be
// asked to decode an unbounded payload. Multipart uploads have their own
// (larger) limits in the handler — those override on a per-handler basis.
func MaxBodyBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
