#!/usr/bin/env bash
# Baut das Docker-Image multi-arch (linux/amd64 + linux/arm64) und pusht
# nach ghcr.io. Tags: :latest und :<short-git-sha>.
#
# Voraussetzungen:
#   - docker login ghcr.io (PAT mit write:packages)
#   - buildx-Builder mit docker-container-Driver (legt das Script bei Bedarf an)
#
# Usage:
#   scripts/docker-push.sh                    # default-Repo (s.u.)
#   IMAGE=ghcr.io/foo/bar scripts/docker-push.sh
#   PLATFORMS=linux/amd64 scripts/docker-push.sh   # nur amd64

set -euo pipefail

IMAGE="${IMAGE:-ghcr.io/pklotz/kita-springer-manager}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
BUILDER_NAME="kita-multiarch"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Working-tree muss sauber sein, sonst zeigt der SHA-Tag auf einen Build mit
# uncommitteten Änderungen — sehr verwirrend bei späteren Rebuilds.
if [[ -n "$(git status --porcelain)" ]]; then
    echo "✗ Working-tree hat uncommittete Änderungen:" >&2
    git status --short >&2
    echo >&2
    echo "Bitte zuerst committen oder stashen — sonst zeigt der :SHA-Tag auf inkonsistenten Code." >&2
    exit 1
fi

SHA="$(git rev-parse --short HEAD)"

# Multi-Platform-Builds brauchen den docker-container-Driver — der Default-Driver
# kann das nicht. Idempotent: existiert er, nutzen; sonst anlegen.
if ! docker buildx inspect "$BUILDER_NAME" >/dev/null 2>&1; then
    echo "→ Lege buildx-Builder '$BUILDER_NAME' an"
    docker buildx create --name "$BUILDER_NAME" --driver docker-container >/dev/null
fi

echo "→ Build & Push"
echo "  Image:     $IMAGE"
echo "  Tags:      latest, $SHA"
echo "  Plattformen: $PLATFORMS"
echo

docker buildx build \
    --builder "$BUILDER_NAME" \
    --platform "$PLATFORMS" \
    -t "${IMAGE}:latest" \
    -t "${IMAGE}:${SHA}" \
    --push \
    .

echo
echo "✓ gepusht: ${IMAGE}:latest und ${IMAGE}:${SHA}"
echo
echo "Inspect:  docker buildx imagetools inspect ${IMAGE}:latest"
