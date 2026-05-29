---
last_mapped: 2026-05-29
---

# Technical Concerns

## Critical

### No automated tests
**Severity:** High  
**Area:** Entire codebase  

Zero test coverage — no `*_test.go` files, no frontend tests. CI only runs `go vet`, `staticcheck`, `govulncheck`, and a smoke build. Complex logic (Excel parser, recurring-assignment expansion, import deduplication, PDF generation) has no regression protection.

**Risk:** Any refactoring or feature change can break existing behavior silently. The Excel importer in particular has intricate column-layout parsing that is entirely exercised only at runtime.

**Mitigation path:** Start with store-layer integration tests using in-memory SQLite (`?mode=memory&cache=shared`). Then add handler-level tests using `httptest`. Frontend: add Vitest.

---

## Medium

### Single-user design — no session management
**Severity:** Medium  
**Area:** `internal/api/middleware/auth.go`, `frontend/src/api/index.js`

Basic Auth credentials are stored in `localStorage` as `base64(admin:password)`. Every API call sends the password in the `Authorization` header. There is no session token, no expiry, no "remember me" distinction.

**Implications:**
- Password rotation immediately invalidates all active browser sessions (intentional but abrupt)
- `localStorage` exposure on XSS (mitigated by strict CSP `script-src 'self'`)
- Username is hardcoded as `admin` — changing it requires updating settings but the UI uses `admin:password` format unconditionally

---

### Excel import fragility
**Severity:** Medium  
**Area:** `internal/importer/excel.go`

The Excel parser uses a configurable column-layout (`ExcelConfig` JSON per provider). Name matching is fuzzy (full-name or any whitespace-separated token). Configuration is stored as JSON in the `providers` table, not validated at write time — a bad `ExcelConfig` silently produces empty imports rather than an error.

**Implications:**
- New provider onboarding requires manual `ExcelConfig` tuning
- Name-matching can produce false positives for common first/last names
- Sheet parsing errors are collected as warnings (not failures) — partial imports can succeed silently

---

### DB connection pool exposed via Holder.Swap race window
**Severity:** Medium  
**Area:** `internal/db/holder.go`

During `Holder.Swap()` (backup restore), in-flight requests that already called `h.db()` and cached the `*sql.DB` pointer may get "database is closed" errors. The Holder uses a `sync.RWMutex` to protect the pointer swap, but handlers that initiate long-running operations before Swap completes will fail mid-operation.

**Mitigation:** Current code acknowledges this: "In-flight queries against the old pool may fail with 'database is closed' — Restore is a deliberate, user-initiated action where this is acceptable."

---

### No rate limiting
**Severity:** Medium  
**Area:** `internal/api/router.go`

No rate limiting on the Basic Auth endpoint or any other API route. Brute-force password attempts are only constrained by bcrypt cost (which slows individual attempts but doesn't block volume).

**Context:** Designed for localhost / behind a reverse proxy. Risk acceptable for self-hosted personal use. Not appropriate if exposed directly to the internet without a proxy that rate-limits.

---

### Scraper brittle on DOM changes
**Severity:** Medium  
**Area:** `cmd/scraper/sources/stadtbern.go`, `cmd/scraper/sources/stiftungbern.go`

HTML scrapers for Kita data use CSS selectors against the Stadt Bern and Stiftung Bern websites. Any DOM restructure by those sites breaks the scraper silently (warnings logged, empty output produced).

**Mitigation path:** Add integration smoke tests that verify the scraper produces non-empty output; alert on parse failures.

---

## Low

### `frontend/dist/` committed to git
**Severity:** Low  
**Area:** `frontend/dist/`

The compiled frontend build output is in the repository (`frontend/dist/`). This means:
- Binaries in git history (inflates repo size over time)
- Risk of dist being out of sync with source
- CI build must run frontend before `go build` because `go:embed` reads from `frontend/dist/`

**Context:** This is required by the `go:embed` pattern — the embed path must exist at compile time. The Dockerfile handles this via multi-stage build. A `.gitignore` entry for `frontend/dist/` would break `go:embed` unless the CI stage explicitly builds first.

---

### No structured migration history
**Severity:** Low  
**Area:** `internal/db/db.go`

Migrations are stored as a Go slice literal — adding a new migration requires knowing the current slice length to place it correctly. No tooling enforces migration order or prevents skipped versions. Migration failures mid-slice can leave the DB at a partial version (mitigated by the `user_version` PRAGMA tracking).

---

### Audit log: no rotation, single file
**Severity:** Low  
**Area:** `internal/audit/audit.go`

Audit log is a single append-only file at `<data-dir>/audit.log`. No rotation, no size cap. For a lightly used self-hosted instance this is fine; for high-traffic deployments the file grows unbounded.

---

### Transit connection cache: no size limit
**Severity:** Low  
**Area:** `internal/store/assignments.go` (cached_connections table)

Cached transit connections are cleaned up at startup (past dates only). No cap on cache size for future assignments. A large date range with many assignments could accumulate significant cached data.

---

## Security notes

| Area | Status |
|------|--------|
| SQL injection | Protected — parameterized queries throughout store layer |
| XSS | Protected — strict CSP (`script-src 'self'`), no innerHTML usage in frontend |
| CSRF | Not applicable — SPA uses JSON API with Authorization header (not cookie-based auth) |
| Mass assignment | Protected — `decodeJSON` uses `DisallowUnknownFields()` |
| Path traversal | Protected — no user-controlled file paths in server |
| Secrets in generated docs | Low risk — no API keys in codebase; auth secrets in DB, not code |
| Dependency vulnerabilities | Monitored weekly via `govulncheck` in CI Security workflow |
