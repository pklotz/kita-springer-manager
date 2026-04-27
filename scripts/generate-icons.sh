#!/usr/bin/env bash
# Generiert die PNG-App-Icons aus frontend/public/icons/icon.svg.
# Nutzt qlmanage + sips (auf macOS vorinstalliert).

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ICON_DIR="${REPO_ROOT}/frontend/public/icons"
SOURCE="${ICON_DIR}/icon.svg"

if [[ "$(uname)" != "Darwin" ]]; then
    echo "Fehler: dieses Script läuft nur auf macOS (qlmanage/sips)." >&2
    exit 1
fi

if [[ ! -f "$SOURCE" ]]; then
    echo "Fehler: $SOURCE fehlt." >&2
    exit 1
fi

cd "$ICON_DIR"

echo "==> Master-PNG (1024×1024) aus SVG rendern"
qlmanage -t -s 1024 -o . "$(basename "$SOURCE")" >/dev/null
mv "$(basename "$SOURCE").png" icon-1024.png

echo "==> Größen ableiten"
sips -z 512 512 icon-1024.png --out icon-512.png >/dev/null
sips -z 192 192 icon-1024.png --out icon-192.png >/dev/null
sips -z 180 180 icon-1024.png --out apple-touch-icon.png >/dev/null

rm icon-1024.png

echo "Fertig:"
ls -la icon-*.png apple-touch-icon.png
