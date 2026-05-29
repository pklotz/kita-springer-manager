---
last_mapped: 2026-05-29
---

# Technology Stack

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

The frontend build output (`frontend/dist/`) is embedded into the Go binary at compile time via `//go:embed` in `frontend/embed.go`. The server serves the SPA as a file system (`fs.FS`) — no separate static file server needed.

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

WAL journal mode enabled (`_journal_mode=WAL`), foreign keys enforced (`_foreign_keys=on`), busy timeout 5 s. Migration is inline code-driven (11 versions as of map date).
