---
last_mapped: 2026-05-29
---

# Testing

## Current state

**No automated tests exist in this codebase.**

As of 2026-05-29:
- No `*_test.go` files found in any Go package
- No `.test.js` or `.spec.js` files in the frontend
- No test runner configured in `package.json` scripts (only `dev`, `build`, `preview`)
- No test-related Make targets

## What exists instead

### Static analysis (Go)
The `Makefile` defines a `check` target that runs:

```bash
make lint   # go vet + staticcheck + go mod tidy check
make vuln   # govulncheck (Go vulnerability database)
```

- `go vet ./...` — standard Go AST checks
- `staticcheck` — `honnef.co/go/tools/cmd/staticcheck` (run via `go run`, no global install needed)
- `go mod tidy` diff — ensures go.mod/go.sum are clean
- `govulncheck` — scans for known CVEs in dependencies

### SBOM & vulnerability scanning
```bash
make sbom         # CycloneDX SBOMs for backend + frontend
make grype        # Grype scan against SBOMs
make grype-image  # Grype scan against built Docker image
```

### Manual QA
The application appears to rely on manual testing via the running app. The `make dev` target starts both backend and frontend dev server simultaneously.

## Framework candidates (if tests are added)

### Go backend
- Standard library `testing` package
- `database/sql` + in-memory SQLite (`?mode=memory` via modernc.org/sqlite) for store-layer integration tests
- `net/http/httptest` for handler-level tests
- No mocking library currently in dependencies

### Frontend
- `vitest` is the natural choice given Vite build tooling (not currently installed)
- `@vue/test-utils` for component testing
- `msw` or `vitest`'s mock capabilities for API mocking

## Risk note

The absence of tests means:
- Regressions in store queries or import logic are caught only at runtime
- The Excel importer (most complex parsing logic) is entirely untested
- The holiday seeding and recurring assignment generation are untested
- Refactoring any store function or validation primitive carries significant regression risk

Adding store-layer integration tests (using in-memory SQLite) would provide the highest value with least friction, as the store functions are pure functions taking `*sql.DB`.
