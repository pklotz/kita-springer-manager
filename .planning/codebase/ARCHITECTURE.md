---
last_mapped: 2026-05-29
---

# Architecture

## Pattern

**Monolith with embedded SPA.** A single Go binary serves both the REST API and the compiled Vue frontend (via `go:embed`). There is no microservice split, no separate frontend server in production.

```
┌─────────────────────────────────────────────────────┐
│                  Go Binary (kita-springer)           │
│                                                      │
│  ┌──────────┐   ┌──────────────────────────────┐    │
│  │  Vue SPA  │   │          REST API            │    │
│  │ (embedded │   │  /api/*  (chi router)        │    │
│  │   dist/)  │   │                              │    │
│  └──────────┘   │  middleware stack:            │    │
│                  │  AccessLog → Recover →        │    │
│                  │  SecurityHeaders → CORS →     │    │
│                  │  BasicAuth → MaxBodyBytes     │    │
│                  │                              │    │
│                  │  handlers → store → SQLite   │    │
│                  └──────────────────────────────┘    │
│                                                      │
│  ┌──────────────────────────────────────────────┐    │
│  │         SQLite (WAL mode, pure Go)            │    │
│  │         data/app.db  (single file)            │    │
│  └──────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

## Layers

### 1. HTTP layer — `internal/api/`

- **`router.go`** — wires middleware stack and all routes
- **`middleware/`** — `AccessLog`, `BasicAuth`, `SecurityHeaders`, `MaxBodyBytes`
- **`handlers/`** — one file per domain (assignments, kitas, providers, etc.)
  - All handlers use `h.db()` (via `db.Holder`) to get the live connection
  - JSON encoding via `writeJSON()` / `decodeJSON()` helpers in `handler.go`
  - Errors classified as `writeError()` (client fault), `serverError()` (our fault), or `upstreamError()` (external dep failure)

### 2. Store layer — `internal/store/`

Pure functions `func Foo(db *sql.DB, …) (Result, error)`. No ORM. Raw SQL queries with `database/sql`. Each function opens/closes its own rows. Store functions are the only place that touches SQL.

### 3. Model layer — `internal/models/`

Plain Go structs with JSON tags. No ORM annotations. Shared between store (scan targets) and API (JSON wire format). No business logic in models.

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

Vue 3 SPA. No build-time SSR. Communicates with backend only via `/api/*` REST calls (axios).

- **`api/index.js`** — single axios instance with auth interceptor (Basic Auth from localStorage), toast-on-error interceptor
- **`views/`** — page components (one per route)
- **`components/`** — reusable UI (forms, calendar grid, panels)
- **`router/index.js`** — client-side routing (History API)
- **`toast.js`** — lightweight toast notification system (no library)

## Data Flow

### Read path (e.g. calendar view)

```
Vue component
  → api/index.js axios call GET /api/assignments?from=…&to=…
  → chi router → BasicAuth middleware → handler.ListAssignments
  → store.ListAssignments(db, from, to) — SQL JOIN assignments + kitas + providers
  → JSON response → Vue reactive update
```

### Write path (e.g. create assignment)

```
Vue form submit
  → POST /api/assignments { kita_id, date, start_time, end_time, … }
  → BasicAuth → MaxBodyBytes → handler.CreateAssignment
  → validate.Date, validate.TimeHM, …
  → store.CreateAssignment(db, req) — INSERT
  → 201 JSON response → Vue updates local state
```

### Excel import

```
File picker → FormData POST /api/providers/{id}/import-excel
  → handler.ImportExcel
  → importer.ImportExcel(db, reader, provider, opts)
    → excelize parse → name-match rows → compute import_hash
    → store.UpsertAssignment per row (INSERT ON CONFLICT UPDATE)
  → {created, updated, skipped, warnings}
```

## DB Holder (runtime swap pattern)

`db.Holder` wraps `*sql.DB` with a `sync.RWMutex`. All handlers call `h.db()` (not a cached pointer) to get the live pool. The backup-restore handler calls `holder.Swap(src)` to atomically replace the live DB file and reopen the pool — no server restart required.

## Authentication flow

```
First boot (no hash set):
  /api/auth/status → {configured: false} → SPA shows SetupView
  /api/auth/setup POST {password} → sets hash, generates download token
  All /api/* routes open until hash is set

Subsequent boots:
  /api/auth/status → {configured: true} → SPA shows LoginView
  User enters password → stored as Basic Auth in localStorage
  Every request: Authorization: Basic base64(admin:password)
  BasicAuth middleware → bcrypt.CompareHashAndPassword
```

## PWA / Service Worker

`vite-plugin-pwa` generates a Workbox service worker. Key decisions:
- `navigateFallback: null` — index.html is NOT precached (always network-first to pick up auth changes)
- `/api/*` routes are `NetworkOnly` — never served from cache
- `transport.opendata.ch` responses cached 1 h (transit-api cache)

## Key architectural constraints

- **No CGO** — `modernc.org/sqlite` is pure Go; enables cross-compilation and distroless container images
- **Single binary** — frontend embedded via `go:embed`; no nginx, no file server
- **Single-user** — one hardcoded `admin` username; no multi-tenancy, no RBAC
- **Local-first** — designed for self-hosted (personal Mac, home server, or Docker); not a SaaS
