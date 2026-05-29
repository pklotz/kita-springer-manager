---
last_mapped: 2026-05-29
---

# Code Conventions

## Go conventions

### Error handling
- Errors are wrapped with context using `fmt.Errorf("context: %w", err)` — never swallowed silently
- HTTP handler errors go through three helpers in `internal/api/handlers/handler.go`:
  - `writeError(w, status, msg)` — client fault (4xx); logs at WARN with human-readable message
  - `serverError(w, err)` — server fault (500); logs underlying error, sends generic German message to client
  - `upstreamError(w, err, hint)` — external dep failure (502); logs technical detail, sends short German hint
- Sensitive error details (SQL schema, file paths, library internals) never appear in HTTP responses
- All validation errors are German strings (`internal/validate`) safe to surface to users

### JSON encoding / decoding
- Custom `decodeJSON(r, v)` wrapper in `handler.go`:
  - Uses `json.Decoder.DisallowUnknownFields()` (mass-assignment guard)
  - Rejects trailing data after first JSON value
  - Body already bounded by `MaxBodyBytes` middleware (1 MiB on JSON routes)
- Custom `writeJSON(w, status, v)` wrapper sets `Content-Type: application/json`

### Store functions
- All store functions are plain functions (no struct receiver): `func ListAssignments(db *sql.DB, …) ([]models.Assignment, error)`
- Functions always return `(Result, error)` — never panics on DB errors
- Row scanning uses `COALESCE` in SQL to handle NULLs before they reach Go structs
- JSON-stored array columns (e.g., `groups`, `stops`) are unmarshalled after scan; empty arrays always initialized to `[]string{}` (never nil)

### DB access
- Handlers always call `h.db()` to get the live connection — never cache `*sql.DB` across calls (Holder.Swap can replace it at runtime)
- Use `db.QueryRow(…).Scan(…)` for single-row reads; `db.Query(…)` with `defer rows.Close()` for lists
- Transactions explicitly use `defer tx.Rollback()` and `tx.Commit()` at the end

### Validation
- All user-facing strings in `internal/validate` are German (no i18n layer needed — single-locale app)
- Validation happens before any DB write; bad input returns 400 with the validation message
- URL validation enforces `https://` only (no `http:` for user-supplied photo URLs — CSP + security)

### Naming
- Package `handlers` imported as `h` (e.g., `h.ListAssignments`, `h.CreateKita`)
- Middleware package imported as `apimw` to avoid chi middleware collision
- Constants for status/source enums in `internal/models` (e.g., `StatusScheduled`, `SourceManual`)

### Logging
- `audit.L()` returns the structured slog logger; used with key-value pairs: `audit.L().Warn("http.error", "status", status, "msg", msg)`
- stdout/stderr used for server-level messages (`log.Printf`); audit.L() for request/error events
- Comments note when logging is intentionally minimal ("Only logged at WARN to avoid drowning normal traffic")

## Frontend conventions

### API layer (`frontend/src/api/index.js`)
- Single axios instance with base URL `/api`
- Auth token stored as `Basic base64(admin:password)` in `localStorage` under key `auth_token`
- Auth interceptor: 401 on non-auth routes → clear token + set `loggedIn.value = false` (no reload)
- Error interceptor: all errors → `showToast(msg, 'error')` for user visibility
- Domain API objects follow pattern: `providersApi.list()`, `kitasApi.create(data)`, etc.

### Component patterns
- Views are in `frontend/src/views/` — fetching data + composing components
- Components are in `frontend/src/components/` — stateless or lightly stateful UI
- Props are typed via Vue's `defineProps`; emits via `defineEmits`
- `KitaDetailPanel.vue` is a slide-in panel pattern reused across views

### State management (Pinia)
- Used selectively — not every domain has a store
- Stores handle cross-view state (e.g., current month, filter state)
- URL query params used for navigation state (month, filters) for back-button support

### Dates & times
- `dayjs` for date manipulation in the frontend
- Backend stores dates as `YYYY-MM-DD` strings (TEXT column in SQLite)
- Times stored as `HH:MM` strings (24-hour)
- `utils/time.js` contains shared date/time utilities

### CSS / Styling
- Tailwind utility classes inline in templates (no custom CSS unless unavoidable)
- `style.css` only contains `@tailwind` directives + minimal global overrides
- Theme color: `#2563eb` (blue-600 in Tailwind)

## Security practices
- HTTP Basic Auth without `WWW-Authenticate` header (SPA renders its own login)
- CSP: `script-src 'self'` (no inline scripts, no eval); `style-src 'unsafe-inline'` required for Vue `:style` bindings
- `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`
- HSTS only emitted when request arrived over TLS
- Download token: 32-byte random hex; constant-time comparison; separate from password (rotating password doesn't break calendar subscriptions)
- `decodeJSON` rejects unknown fields (mass-assignment guard)
- Photo URLs validated as `https://` only

## Commit style (from git log)
German-language commit messages describing the feature from the user perspective. Examples:
- `Kalender: Klick auf leere Zelle öffnet Neuer-Einsatz-Dialog mit Datum vorausgefüllt`
- `Navigation: Monat und Filter in URL speichern für Back-Button`
- `Import: aufklappbare Warnungen + sprechende Fehlermeldung bei fehlender Kita`

Format: `<Modul>: <Was wurde gemacht>` (module-colon-description in German)
