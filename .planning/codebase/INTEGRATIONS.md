---
last_mapped: 2026-05-29
---

# External Integrations

## Public Transport API — transport.opendata.ch

| | |
|---|---|
| Base URL | `https://transport.opendata.ch/v1` |
| Auth | None (public API) |
| Client | `internal/transit/client.go` |
| Timeout | 10 s per request |

**Used for:**
- `/v1/connections` — fetch next connections between two stops (direction outbound/inbound for assignments)
- `/v1/locations` — search stops by name or proximity (lat/lng)

**Caching:** Connection results are stored in the `cached_connections` table (keyed by `assignment_id + direction`). Past-date entries are cleaned up at server start.

**PWA runtime caching:** The service worker caches transport.opendata.ch responses in a `transit-api` cache (max 50 entries, 1 h TTL).

---

## Geocoding — Nominatim (OpenStreetMap)

| | |
|---|---|
| Base URL | `https://nominatim.openstreetmap.org/search` |
| Auth | None |
| Client | `internal/transit/client.go` — `Geocode()` |
| User-Agent | `kita-springer-manager/1.0 (self-hosted)` |
| Result limit | 1 |

**Used for:** Resolving kita addresses to WGS84 lat/lng coordinates (stored on the `kitas` row).

**Note:** Accept-Language is set to `de` to prefer German place names.

---

## Web Scrapers (Kita data sources)

| Source | File | Target URL |
|--------|------|------------|
| Stadt Bern | `cmd/scraper/sources/stadtbern.go` | `https://www.bern.ch/…/unsere-kitas` |
| Stiftung Bern | `cmd/scraper/sources/stiftungbern.go` | (Stiftung Bern website) |

**Library:** `github.com/PuerkitoBio/goquery` (jQuery-like HTML DOM traversal).

**Output:** Both scrapers produce `.xlsx` files in the standard column format (`Name, Adresse, ÖV-Haltestelle, Telefon, Email, Gruppen, Notizen`) which can then be imported via the regular Excel import flow.

**Run:** `go run ./cmd/scraper` (standalone binary, not part of the server).

---

## Calendar export — iCalendar (RFC 5545)

| | |
|---|---|
| Endpoint | `GET /api/calendar.ics` |
| Handler | `internal/api/handlers/calendar.go` |
| Auth | Basic auth OR `?token=<download-token>` |

**Used for:** Subscribing the assignment schedule in Apple Calendar, Google Calendar, etc. Download token allows subscription URLs without exposing the password.

---

## PDF export — fpdf

| | |
|---|---|
| Endpoint | `GET /api/worktime/export` |
| Library | `github.com/go-pdf/fpdf` |
| Handler | `internal/api/handlers/worktime.go` |
| Generator | `internal/pdf/worktime.go` |
| Auth | Basic auth OR `?token=<download-token>` |

**Used for:** Generating printable monthly worktime reports grouped by provider.

---

## Excel import — excelize

| | |
|---|---|
| Endpoint | `POST /api/providers/{id}/import-excel` |
| Endpoint | `POST /api/kitas/import` |
| Library | `github.com/xuri/excelize/v2` |

**Two distinct import flows:**
1. **Provider schedule import** — parses a provider's weekly schedule Excel to create/update assignments. Configuration (column layout, name-matching) is stored per-provider in `excel_config` (JSON).
2. **Kita import** — bulk-imports kita metadata (name, address, stops, contacts) from a standard-format Excel.

---

## Authentication

Single-user HTTP Basic Auth. No OAuth, no external IdP.

- Password hash: bcrypt (`golang.org/x/crypto/bcrypt`)
- Stored in `settings` table (`auth_username`, `auth_password_hash`)
- Download token: 32-byte random hex, also in `settings`
- Setup mode: if no hash stored, server is open until first password set via `/api/auth/setup`

---

## No integrations (by design)

- No external database (SQLite only)
- No email sending
- No push notifications
- No third-party auth (Google, GitHub, etc.)
- No analytics
- No CDN — all assets embedded in binary
