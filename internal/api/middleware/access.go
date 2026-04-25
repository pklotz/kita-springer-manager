package middleware

import (
	"net/http"
	"strings"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/pak/kita-springer-manager/internal/audit"
)

// AccessLog records every HTTP request as a structured audit event. Replaces
// chi's default text logger so we get JSON lines compatible with the rest of
// the audit trail. Status >= 400 is logged at WARN, otherwise INFO.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", clientIP(r),
		}
		// Capture the basic-auth username if present — useful when the same
		// app is later shared, and harmless for single-user.
		if u, _, ok := r.BasicAuth(); ok {
			attrs = append(attrs, "user", u)
		}
		// GET query-string aids debugging filter/range issues; POST/PUT bodies
		// are deliberately NOT captured to avoid logging secrets/PII.
		if q := r.URL.RawQuery; q != "" && r.Method == http.MethodGet {
			attrs = append(attrs, "query", q)
		}

		if ww.Status() >= 400 {
			audit.L().Warn("http", attrs...)
		} else {
			audit.L().Info("http", attrs...)
		}
	})
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		// X-F-F can be comma-separated — take the first hop only.
		if i := strings.IndexByte(v, ','); i >= 0 {
			return strings.TrimSpace(v[:i])
		}
		return v
	}
	return r.RemoteAddr
}
