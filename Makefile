.PHONY: run build build-backup build-darwin-arm64 build-darwin-amd64 build-darwin-universal frontend-build frontend-dev dev lint vuln check \
        docker-build sbom sbom-backend sbom-frontend sbom-clean grype grype-image tools-install

FRONTEND_DIR    := frontend

DOCKER_IMAGE    := kita-springer
DOCKER_TAG      := latest
DOCKER_PLATFORM := linux/amd64

SBOM_DIR        := sbom
GOBIN           := $(shell go env GOPATH)/bin
CYCLONEDX_GOMOD := $(shell command -v cyclonedx-gomod 2>/dev/null || echo $(GOBIN)/cyclonedx-gomod)
GRYPE           := $(shell command -v grype 2>/dev/null || echo $(GOBIN)/grype)

run:
	go run ./cmd/server

build:
	go build -o bin/kita-springer ./cmd/server

build-backup:
	go build -o bin/kita-springer-backup ./cmd/backup

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

# ── Docker (linux/amd64) ──────────────────────────────────────────────────────

# Lokales Image bauen (single-arch), z.B. damit grype-image es scannen kann.
# Multi-Arch-Push läuft separat über scripts/docker-push.sh.
docker-build:
	docker buildx build --platform $(DOCKER_PLATFORM) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		--load .

# ── SBOM (CycloneDX) ──────────────────────────────────────────────────────────
# Erzeugt zwei separate CycloneDX-SBOMs (Backend + Frontend) unter sbom/.
# Backend = cmd/server (das, was tatsächlich ins Docker-Image kommt).

sbom: sbom-backend sbom-frontend

# Backend: nur die Module, die wirklich ins Server-Binary gelinkt werden.
sbom-backend:
	@mkdir -p $(SBOM_DIR)
	$(CYCLONEDX_GOMOD) app -json \
		-output $(SBOM_DIR)/backend.cdx.json \
		-main cmd/server \
		-licenses .

# Frontend: scannt node_modules. --ignore-npm-errors toleriert kleine
# npm-ls-Inkonsistenzen, ohne den Lauf abzubrechen.
sbom-frontend:
	@mkdir -p $(SBOM_DIR)
	cd $(FRONTEND_DIR) && npx --yes @cyclonedx/cyclonedx-npm \
		--ignore-npm-errors \
		--output-format JSON \
		--output-file $(CURDIR)/$(SBOM_DIR)/frontend.cdx.json

sbom-clean:
	rm -rf $(SBOM_DIR)

# ── Vulnerability-Scan (Grype) ────────────────────────────────────────────────
# Scannt beide CycloneDX-SBOMs. Frischt sie zuvor auf.
grype: sbom
	@echo "── Backend ──"
	$(GRYPE) sbom:$(SBOM_DIR)/backend.cdx.json
	@echo ""
	@echo "── Frontend ──"
	$(GRYPE) sbom:$(SBOM_DIR)/frontend.cdx.json

# Scannt das fertig gebaute Docker-Image direkt (inkl. Basisimage-Layer).
grype-image: docker-build
	$(GRYPE) $(DOCKER_IMAGE):$(DOCKER_TAG)

# Komfort: installiert die nötigen CLI-Tools (Grype kann zusätzlich via brew kommen).
tools-install:
	go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
	go install github.com/anchore/grype/cmd/grype@latest
