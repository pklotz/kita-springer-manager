.PHONY: run build build-darwin-arm64 build-darwin-amd64 build-darwin-universal frontend-build frontend-dev dev lint vuln check

run:
	go run ./cmd/server

build:
	go build -o bin/kita-springer ./cmd/server

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o bin/kita-springer-darwin-arm64 ./cmd/server

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o bin/kita-springer-darwin-amd64 ./cmd/server

# Universal Binary für Intel- und Apple-Silicon-Macs (benötigt lipo, ist auf macOS vorinstalliert).
build-darwin-universal: build-darwin-arm64 build-darwin-amd64
	lipo -create -output bin/kita-springer-darwin-universal \
		bin/kita-springer-darwin-arm64 \
		bin/kita-springer-darwin-amd64

frontend-build:
	cd frontend && npm run build

frontend-dev:
	cd frontend && npm run dev

# Full dev workflow: backend + frontend in parallel
dev:
	@echo "Starting backend on :9092 and frontend dev server on :5173"
	@trap 'kill 0' INT; \
	  go run ./cmd/server & \
	  cd frontend && npm run dev & \
	  wait

# Static analysis: vet + staticcheck + go.mod cleanliness.
# Uses `go run` so contributors don't need staticcheck installed globally.
lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...
	@cp go.mod go.mod.bak; cp go.sum go.sum.bak; \
	  go mod tidy; \
	  diff -q go.mod go.mod.bak >/dev/null && diff -q go.sum go.sum.bak >/dev/null; \
	  ec=$$?; mv go.mod.bak go.mod; mv go.sum.bak go.sum; \
	  if [ $$ec -ne 0 ]; then echo "go.mod/go.sum not tidy — run 'go mod tidy' and commit"; exit 1; fi

# Vulnerability scan against the Go vulnerability database.
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Composite quality gate — same checks the CI runs.
check: lint vuln
