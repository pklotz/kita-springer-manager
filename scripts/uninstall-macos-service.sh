#!/usr/bin/env bash
# Entfernt den kita-springer LaunchAgent. Lässt die Datenbank und Logs unter
# /usr/local/kita-springer/ unangetastet — die müssen bei Bedarf manuell gelöscht
# werden. Siehe docs/macos-service.md.

set -euo pipefail

LABEL="ch.kita-springer.server"
PLIST_PATH="$HOME/Library/LaunchAgents/${LABEL}.plist"
INSTALL_DIR="/usr/local/kita-springer"

if [[ "$(uname)" != "Darwin" ]]; then
    echo "Fehler: dieses Script läuft nur auf macOS." >&2
    exit 1
fi

if [[ -f "$PLIST_PATH" ]]; then
    echo "==> Service entladen"
    launchctl unload -w "$PLIST_PATH" 2>/dev/null || true
    rm -f "$PLIST_PATH"
    echo "    plist entfernt: $PLIST_PATH"
else
    echo "==> Keine plist gefunden, überspringe."
fi

echo
echo "Fertig. Daten unter ${INSTALL_DIR} sind erhalten geblieben."
echo "Für vollständiges Entfernen:  sudo rm -rf ${INSTALL_DIR}"
