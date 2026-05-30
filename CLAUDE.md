<!-- GSD:project-start source:PROJECT.md -->
## Project

**Kita Springer Manager — Pausen-Umstellung**

Kita Springer Manager ist eine selbst-gehostete Einsatzplanungs-App für Kita-Springerinnen in Bern. Sie verwaltet Einsätze bei verschiedenen Kitas, Anbieter-Importe (Excel), Arbeitszeitnachweise (PDF), Fahrverbindungen und Kalender-Exporte. Die App besteht aus einem Go-Backend mit eingebettetem Vue 3 Frontend und SQLite-Datenbank.

Dieses Planungsprojekt fokussiert auf eine gezielte Verbesserung der Pausen-Erfassung.

**Core Value:** Pausen korrekt in die Arbeitszeitberechnung einfliessen lassen — als Dauer, nicht als Zeitraum.

### Constraints

- **Kompatibilität**: Migration muss bestehende Daten erhalten — `break_end − break_start` in Minuten umrechnen; leere Werte → 0
- **Kein CGO**: `modernc.org/sqlite` — bleibt so
- **Single-User**: Keine Multi-Tenancy-Überlegungen nötig
- **Rückwärtskompatibilität API**: `actual_break_start`/`actual_break_end` aus dem API-Request können ignoriert oder abgelehnt werden
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Language & Runtime
| Layer | Language | Version |
|-------|----------|---------|
| Backend | Go | 1.25 |
| Frontend | JavaScript (ES modules) | Node 20 (build) |
| Database | SQLite (pure Go) | modernc.org/sqlite v1.29.9 |
## Backend Frameworks & Libraries
| Library | Version | Purpose |
|---------|---------|---------|
| `github.com/go-chi/chi/v5` | v5.2.2 | HTTP router + middleware |
| `github.com/go-chi/cors` | v1.2.1 | CORS middleware |
| `modernc.org/sqlite` | v1.29.9 | Pure-Go SQLite driver (no CGO) |
| `golang.org/x/crypto` | v0.49.0 | bcrypt password hashing |
| `github.com/go-pdf/fpdf` | v0.9.0 | PDF generation (worktime reports) |
| `github.com/xuri/excelize/v2` | v2.10.1 | Excel (.xlsx) parsing and writing |
| `github.com/rickar/cal/v2` | v2.1.27 | Swiss public holiday calendar (per-canton) |
| `github.com/google/uuid` | v1.6.0 | UUID generation for primary keys |
| `github.com/PuerkitoBio/goquery` | v1.12.0 | HTML scraping (Kita data sources) |
| `log/slog` | stdlib | Structured JSON audit logging |
## Frontend Frameworks & Libraries
| Library | Version | Purpose |
|---------|---------|---------|
| `vue` | ^3.4.0 | UI framework |
| `vue-router` | ^4.3.0 | Client-side routing |
| `pinia` | ^2.1.0 | State management |
| `@headlessui/vue` | ^1.7.0 | Accessible UI primitives |
| `lucide-vue-next` | ^0.400.0 | Icon set |
| `axios` | ^1.7.0 | HTTP client with interceptors |
| `dayjs` | ^1.11.0 | Date manipulation |
| `tailwindcss` | ^3.4.0 | Utility-first CSS |
| `vite` | ^6.4.2 | Build tool + dev server |
| `vite-plugin-pwa` | ^1.2.0 | Progressive Web App (Service Worker) |
## Build & Deployment
| Tool | Purpose |
|------|---------|
| `make` | All build/dev targets (see `Makefile`) |
| `go build` (CGO_ENABLED=0) | Cross-compile static binaries |
| `lipo` | macOS universal binary (arm64 + amd64) |
| Docker (multi-stage) | Containerized production deployment |
| `gcr.io/distroless/static-debian12:nonroot` | Minimal runtime image |
| `npm ci` + `vite build` | Frontend build (embedded via `go:embed`) |
## Frontend embedding strategy
## Configuration
| Setting | Default | Source |
|---------|---------|--------|
| Listen address | `127.0.0.1:9092` | `--addr` flag / `ADDR` env |
| DB path | `<bin-dir>/data/app.db` | `--db` flag / `DB_PATH` env |
| Initial password | — | `KITA_INITIAL_PASSWORD` env (headless setup) |
| Initial username | `admin` | `KITA_INITIAL_USERNAME` env |
| Docker listen | `:8080` | `ENV ADDR=:8080` in Dockerfile |
| Docker DB path | `/data/app.db` | `ENV DB_PATH=/data/app.db` |
## SQLite Configuration
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Go conventions
### Error handling
- Errors are wrapped with context using `fmt.Errorf("context: %w", err)` — never swallowed silently
- HTTP handler errors go through three helpers in `internal/api/handlers/handler.go`:
- Sensitive error details (SQL schema, file paths, library internals) never appear in HTTP responses
- All validation errors are German strings (`internal/validate`) safe to surface to users
### JSON encoding / decoding
- Custom `decodeJSON(r, v)` wrapper in `handler.go`:
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
- `Kalender: Klick auf leere Zelle öffnet Neuer-Einsatz-Dialog mit Datum vorausgefüllt`
- `Navigation: Monat und Filter in URL speichern für Back-Button`
- `Import: aufklappbare Warnungen + sprechende Fehlermeldung bei fehlender Kita`
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern
```
```
## Layers
### 1. HTTP layer — `internal/api/`
- **`router.go`** — wires middleware stack and all routes
- **`middleware/`** — `AccessLog`, `BasicAuth`, `SecurityHeaders`, `MaxBodyBytes`
- **`handlers/`** — one file per domain (assignments, kitas, providers, etc.)
### 2. Store layer — `internal/store/`
### 3. Model layer — `internal/models/`
### 4. Domain packages
| Package | Responsibility |
|---------|----------------|
| `internal/db` | DB open + migration runner + `Holder` (runtime DB swap) |
| `internal/audit` | Append-only JSON-line audit log (wraps `log/slog`) |
| `internal/pdf` | PDF report generation (fpdf) |
| `internal/importer` | Excel schedule parsing → assignments |
| `internal/transit` | transport.opendata.ch + Nominatim clients |
| `internal/validate` | Input validation primitives (German error strings) |
| `internal/seeds` | Holiday seeding via `rickar/cal` |
### 5. Frontend — `frontend/src/`
- **`api/index.js`** — single axios instance with auth interceptor (Basic Auth from localStorage), toast-on-error interceptor
- **`views/`** — page components (one per route)
- **`components/`** — reusable UI (forms, calendar grid, panels)
- **`router/index.js`** — client-side routing (History API)
- **`toast.js`** — lightweight toast notification system (no library)
## Data Flow
### Read path (e.g. calendar view)
```
```
### Write path (e.g. create assignment)
```
```
### Excel import
```
```
## DB Holder (runtime swap pattern)
## Authentication flow
```
```
## PWA / Service Worker
- `navigateFallback: null` — index.html is NOT precached (always network-first to pick up auth changes)
- `/api/*` routes are `NetworkOnly` — never served from cache
- `transport.opendata.ch` responses cached 1 h (transit-api cache)
## Key architectural constraints
- **No CGO** — `modernc.org/sqlite` is pure Go; enables cross-compilation and distroless container images
- **Single binary** — frontend embedded via `go:embed`; no nginx, no file server
- **Single-user** — one hardcoded `admin` username; no multi-tenancy, no RBAC
- **Local-first** — designed for self-hosted (personal Mac, home server, or Docker); not a SaaS
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
