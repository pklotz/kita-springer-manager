#!/usr/bin/env bash
# Convenience-Wrapper um die Go-CLI cmd/backup. Baut bin/kita-springer-backup
# beim ersten Aufruf, danach nur noch falls der Quellcode jünger ist.
#
# Usage:
#   scripts/backup.sh export  [-db PATH] [-out PATH]
#   scripts/backup.sh verify   -in PATH
#   scripts/backup.sh restore  -in PATH [-db PATH] [-y] [-reset-password]
#
# Beispiele:
#   scripts/backup.sh export                       # default: data/app.db → kita-springer-YYYY-MM-DD.db
#   scripts/backup.sh export -db /usr/local/kita-springer/data/app.db
#   scripts/backup.sh restore -in backup.db -y     # ACHTUNG: vorher Server stoppen
#
# Für reines SQLite-CLI ohne diesen Wrapper:
#   sqlite3 data/app.db "VACUUM INTO 'backup.db'"

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${REPO_ROOT}/bin/kita-springer-backup"
SRC_DIR="${REPO_ROOT}/cmd/backup"

needs_build() {
    [[ ! -x "$BIN" ]] && return 0
    # Rebuild, falls eine .go-Datei jünger ist als das Binary.
    if find "$SRC_DIR" "${REPO_ROOT}/internal/db" -name '*.go' -newer "$BIN" 2>/dev/null | grep -q .; then
        return 0
    fi
    return 1
}

if needs_build; then
    echo "→ Baue $BIN" >&2
    (cd "$REPO_ROOT" && go build -o "$BIN" ./cmd/backup)
fi

exec "$BIN" "$@"
