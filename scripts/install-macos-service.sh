#!/usr/bin/env bash
# Installiert kita-springer als macOS LaunchAgent (Auto-Start beim Login).
# Siehe docs/macos-service.md für Details.

set -euo pipefail

LABEL="ch.kita-springer.server"
INSTALL_DIR="/usr/local/kita-springer"
PLIST_PATH="$HOME/Library/LaunchAgents/${LABEL}.plist"
ADDR="${ADDR:-127.0.0.1:9092}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY_SRC="${REPO_ROOT}/bin/kita-springer"

if [[ "$(uname)" != "Darwin" ]]; then
    echo "Fehler: dieses Script läuft nur auf macOS." >&2
    exit 1
fi

if [[ ! -x "$BINARY_SRC" ]]; then
    echo "Fehler: $BINARY_SRC fehlt. Bitte zuerst 'make build' ausführen." >&2
    exit 1
fi

echo "==> Installationsverzeichnis vorbereiten: $INSTALL_DIR"
sudo mkdir -p "$INSTALL_DIR/data"
sudo chown -R "$(whoami)" "$INSTALL_DIR"

echo "==> Binary kopieren"
cp "$BINARY_SRC" "$INSTALL_DIR/kita-springer"
chmod +x "$INSTALL_DIR/kita-springer"

# Quarantäne-Attribut entfernen, damit Gatekeeper das Binary nicht blockiert.
xattr -d com.apple.quarantine "$INSTALL_DIR/kita-springer" 2>/dev/null || true

echo "==> LaunchAgent-plist schreiben: $PLIST_PATH"
mkdir -p "$(dirname "$PLIST_PATH")"
cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${LABEL}</string>

    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/kita-springer</string>
    </array>

    <key>WorkingDirectory</key>
    <string>${INSTALL_DIR}</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>ADDR</key>
        <string>${ADDR}</string>
    </dict>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>${INSTALL_DIR}/server.log</string>
    <key>StandardErrorPath</key>
    <string>${INSTALL_DIR}/server.err.log</string>
</dict>
</plist>
EOF

echo "==> Service neu laden"
# Bei bestehendem Service zuerst entladen, sonst meckert launchctl.
launchctl unload "$PLIST_PATH" 2>/dev/null || true
launchctl load -w "$PLIST_PATH"

echo
echo "Fertig. Service '${LABEL}' läuft auf ${ADDR}."
echo "Status:    launchctl list | grep kita-springer"
echo "Logs:      tail -f ${INSTALL_DIR}/server.log"
echo "Stoppen:   launchctl stop ${LABEL}"
echo "Entfernen: scripts/uninstall-macos-service.sh"
