---
last_mapped: 2026-05-29
---

# Directory Structure

## Top-level layout

```
kita-springer-manager/
├── cmd/                    # Binary entry points
│   ├── server/main.go      # Main application server
│   ├── backup/main.go      # Standalone backup utility
│   └── scraper/            # Kita data web scraper
│       ├── main.go
│       └── sources/        # One file per data source (stadtbern, stiftungbern)
├── internal/               # Application packages (not importable from outside)
│   ├── api/                # HTTP layer
│   │   ├── router.go       # Route + middleware wiring
│   │   ├── handlers/       # Request handlers (one file per domain)
│   │   └── middleware/     # Custom middleware (auth, security, access log)
│   ├── audit/              # Structured JSON audit logger
│   ├── db/                 # SQLite open/migrate + db.Holder
│   ├── importer/           # Excel schedule parser → assignments
│   ├── models/             # Domain model structs (JSON wire + scan targets)
│   ├── pdf/                # PDF report generation
│   ├── seeds/              # Public holiday seeding
│   ├── store/              # Data access layer (raw SQL functions)
│   ├── transit/            # External API clients (opendata.ch, Nominatim)
│   └── validate/           # Input validation primitives
├── frontend/               # Vue 3 SPA
│   ├── src/
│   │   ├── api/index.js    # Axios client, auth interceptors, all API calls
│   │   ├── views/          # Page components (one per route)
│   │   ├── components/     # Shared UI components
│   │   ├── router/         # vue-router config
│   │   ├── utils/          # Utility functions (time, closures)
│   │   ├── toast.js        # Toast notification system
│   │   ├── App.vue         # Root component (auth gate, nav, toast outlet)
│   │   ├── main.js         # Vue + Pinia + Router bootstrap
│   │   └── style.css       # Global styles (Tailwind directives)
│   ├── public/             # Static assets (icons, manifest)
│   ├── embed.go            # go:embed directive for dist/
│   ├── index.html          # SPA shell
│   ├── package.json
│   ├── vite.config.js
│   ├── tailwind.config.js
│   └── postcss.config.js
├── data/                   # Runtime data (gitignored in prod — Docker volume)
│   ├── app.db              # SQLite database
│   └── audit.log           # JSON-line audit log
├── bin/                    # Build outputs (gitignored except release binaries)
├── scripts/                # Shell scripts (backup, docker push, release, etc.)
├── docs/                   # Developer docs (macOS service setup)
├── sbom/                   # CycloneDX SBOM files (backend.cdx.json, frontend.cdx.json)
├── Makefile                # All dev/build/release targets
├── Dockerfile              # Multi-stage build (frontend → backend → distroless)
├── go.mod / go.sum
└── README.md
```

## Key locations at a glance

| What | Where |
|------|-------|
| HTTP routes | `internal/api/router.go` |
| Handler per domain | `internal/api/handlers/<domain>.go` |
| SQL queries | `internal/store/<domain>.go` |
| Domain models | `internal/models/<domain>.go` |
| DB schema + migrations | `internal/db/db.go` — `migrations` slice |
| DB connection management | `internal/db/holder.go` |
| Auth logic | `internal/store/auth.go` + `internal/api/middleware/auth.go` |
| Validation primitives | `internal/validate/validate.go` |
| Frontend API calls | `frontend/src/api/index.js` |
| Frontend routing | `frontend/src/router/index.js` |
| Dev + build commands | `Makefile` |
| Docker config | `Dockerfile` |

## Naming conventions

### Go
- Package names are lowercase, single-word (e.g., `handlers`, `store`, `transit`)
- Handler methods are `VerbNoun` (e.g., `ListAssignments`, `CreateKita`, `DeleteProvider`)
- Store functions are `VerbNoun(db *sql.DB, …)` — pure functions, no receiver
- Model structs use PascalCase with JSON snake_case tags

### Frontend
- View components: `<Domain>View.vue` (e.g., `CalendarView.vue`, `ProvidersView.vue`)
- Shared components: `<Noun>.vue` or `<Noun><Descriptor>.vue` (e.g., `KitaDetailPanel.vue`, `AssignmentForm.vue`)
- Utility modules: `utils/<domain>.js` (e.g., `utils/time.js`, `utils/closures.js`)
- API module exports: `providersApi`, `kitasApi`, `assignmentsApi`, etc.

## Frontend routes

| Path | View | Purpose |
|------|------|---------|
| `/` | `CalendarView.vue` | Monthly calendar grid (main view) |
| `/assignments/:id` | `AssignmentView.vue` | Assignment detail / edit |
| `/history` | `HistoryView.vue` | Past assignments list |
| `/worktime` | `WorktimeView.vue` | Worktime tracking + PDF export |
| `/providers` | `ProvidersView.vue` | Provider management + Excel import |
| `/settings` | `SettingsView.vue` | App settings (name, canton, etc.) |

Login (`LoginView.vue`) and Setup (`SetupView.vue`) are rendered conditionally by `App.vue` based on auth state — not in the router.
