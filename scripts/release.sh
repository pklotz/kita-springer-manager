#!/usr/bin/env bash
# Baut Cross-Plattform-Binaries und legt ein GitHub-Release an.
#
# Voraussetzungen:
#   - gh CLI authentifiziert (`gh auth status`)
#   - lipo (macOS, für die universal-Binary) — wird übersprungen, wenn nicht vorhanden
#   - npm + Go-Toolchain
#
# Usage:
#   scripts/release.sh                          # Tag muss auf HEAD liegen
#   VERSION=v1.0.0 scripts/release.sh           # explizit
#   ALLOW_DIRTY=1  scripts/release.sh           # ignoriert lokale Modifikationen
#   OVERWRITE=1    scripts/release.sh           # überschreibt ein bestehendes Release

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# ── Version bestimmen ─────────────────────────────────────────────────────────
# Vorrang: VERSION-env > Tag auf HEAD. Tag-Format vX.Y.Z erzwingen, damit
# Sortierung in Releases und SemVer-Tooling konsistent funktioniert.
if [[ -z "${VERSION:-}" ]]; then
    if ! VERSION="$(git describe --tags --exact-match HEAD 2>/dev/null)"; then
        echo "Kein Tag auf HEAD. Entweder 'git tag vX.Y.Z' setzen oder VERSION=… mitgeben." >&2
        exit 1
    fi
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?$ ]]; then
    echo "VERSION '$VERSION' entspricht nicht vX.Y.Z[-prerelease]." >&2
    exit 1
fi

# ── Working Tree prüfen ───────────────────────────────────────────────────────
if [[ -z "${ALLOW_DIRTY:-}" ]] && ! git diff --quiet HEAD; then
    echo "Working tree ist nicht clean — entweder committen oder ALLOW_DIRTY=1 setzen:" >&2
    git diff --name-only HEAD | sed 's/^/   /' >&2
    exit 1
fi

# ── gh-CLI prüfen ─────────────────────────────────────────────────────────────
if ! command -v gh >/dev/null 2>&1; then
    echo "gh CLI nicht gefunden. Bitte installieren: https://cli.github.com/" >&2
    exit 1
fi
if ! gh auth status >/dev/null 2>&1; then
    echo "gh CLI nicht authentifiziert — 'gh auth login' ausführen." >&2
    exit 1
fi

# ── Existiert das Release schon? ──────────────────────────────────────────────
if gh release view "$VERSION" >/dev/null 2>&1; then
    if [[ -z "${OVERWRITE:-}" ]]; then
        echo "Release $VERSION existiert bereits. OVERWRITE=1 setzen, um es zu ersetzen." >&2
        exit 1
    fi
    echo "→ Release $VERSION existiert — wird überschrieben."
    gh release delete "$VERSION" --yes --cleanup-tag=false
fi

OUT_DIR="bin/release/$VERSION"
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

# ── Frontend einmalig bauen ───────────────────────────────────────────────────
# Output landet in frontend/dist und wird via go:embed in jedes Binary gezogen,
# unabhängig von Ziel-OS/-Arch.
echo "→ Frontend bauen"
(cd frontend && npm ci && npm run build)

SHA="$(git rev-parse --short HEAD)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X main.version=$VERSION -X main.commit=$SHA -X main.buildDate=$DATE"

# ── Cross-Compile ─────────────────────────────────────────────────────────────
# Pure-Go-Build (CGO_ENABLED=0) — modernc.org/sqlite ist pure Go, also keine
# Cross-Toolchain nötig. Output direkt als einzelnes Binary mit Plattform im
# Namen — keine Archive, damit User mit einem Klick herunterladen, chmod +x,
# fertig.
build_one() {
    local goos="$1" goarch="$2"
    local out="$OUT_DIR/kita-springer-${VERSION}-${goos}-${goarch}"
    echo "→ Build ${goos}/${goarch}"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
        -trimpath \
        -ldflags="$LDFLAGS" \
        -o "$out" \
        ./cmd/server
    echo "   $out"
}

build_one linux  amd64
build_one linux  arm64
build_one darwin amd64
build_one darwin arm64

# ── Darwin Universal Binary (optional) ────────────────────────────────────────
# lipo gibt's nur auf macOS. Skipped wenn nicht verfügbar — die einzelnen
# darwin-arm64/-amd64 Binaries bleiben ja vorhanden.
if command -v lipo >/dev/null 2>&1; then
    echo "→ darwin-universal (lipo)"
    lipo -create \
        -output "$OUT_DIR/kita-springer-${VERSION}-darwin-universal" \
        "$OUT_DIR/kita-springer-${VERSION}-darwin-amd64" \
        "$OUT_DIR/kita-springer-${VERSION}-darwin-arm64"
    echo "   $OUT_DIR/kita-springer-${VERSION}-darwin-universal"
else
    echo "→ lipo nicht gefunden — überspringe darwin-universal"
fi

# ── Checksums ─────────────────────────────────────────────────────────────────
echo "→ SHA256SUMS"
(
    cd "$OUT_DIR"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum kita-springer-${VERSION}-* > SHA256SUMS.txt
    else
        # macOS hat shasum statt sha256sum.
        shasum -a 256 kita-springer-${VERSION}-* > SHA256SUMS.txt
    fi
)

# ── GitHub Release anlegen ────────────────────────────────────────────────────
NOTES_FILE="$OUT_DIR/release-notes.md"
cat > "$NOTES_FILE" <<EOF
Release $VERSION

Commit: \`$SHA\`
Build: $DATE

## Downloads

Direkt das passende Binary herunterladen, ausführbar machen (\`chmod +x …\`)
und starten — Frontend ist eingebettet, keine weiteren Dateien nötig.

| Plattform | Datei |
|-----------|-------|
| Linux x86_64 | \`kita-springer-${VERSION}-linux-amd64\` |
| Linux arm64 (z.B. Raspberry Pi 4/5) | \`kita-springer-${VERSION}-linux-arm64\` |
| macOS Apple Silicon | \`kita-springer-${VERSION}-darwin-arm64\` |
| macOS Intel | \`kita-springer-${VERSION}-darwin-amd64\` |
| macOS Universal | \`kita-springer-${VERSION}-darwin-universal\` |

Die SHA256-Summen aller Binaries liegen in \`SHA256SUMS.txt\`.

## Docker

\`\`\`
docker pull ghcr.io/pklotz/kita-springer-manager:${VERSION#v}
\`\`\`
EOF

echo "→ gh release create $VERSION"
gh release create "$VERSION" \
    --title "$VERSION" \
    --notes-file "$NOTES_FILE" \
    "$OUT_DIR"/kita-springer-${VERSION}-* \
    "$OUT_DIR/SHA256SUMS.txt"

echo
echo "✓ Release $VERSION veröffentlicht."
echo "  $(gh release view "$VERSION" --json url -q .url)"
