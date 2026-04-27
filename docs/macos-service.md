# kita-springer als macOS-Service betreiben

Diese Anleitung beschreibt, wie der `kita-springer`-Server als Hintergrund-Service
auf macOS registriert wird, sodass er ohne Terminal-Fenster läuft und beim Login
automatisch startet.

## Hintergrund: launchd

macOS hat mit **`launchd`** einen eigenen Service-Manager (vergleichbar mit
`systemd` unter Linux). Services werden über XML-Dateien (`.plist`) beschrieben
und mit `launchctl` verwaltet. Es gibt zwei Varianten:

| | **LaunchAgent** | **LaunchDaemon** |
|---|---|---|
| Pfad | `~/Library/LaunchAgents/` | `/Library/LaunchDaemons/` |
| Startzeitpunkt | beim **User-Login** | beim **System-Boot** (vor Login) |
| Kontext | aktueller User | root |
| Rechte zum Installieren | normal | sudo |

Für `kita-springer` ist der **LaunchAgent** die richtige Wahl: die App hört nur
auf `127.0.0.1:9092`, braucht keine Root-Rechte, und „Auto-Start mit dem System"
heißt auf Desktop-Macs in der Praxis sowieso „nach dem Login".

> **Wann LaunchDaemon?** Nur wenn die App auf einem Server-Mac ohne aktive
> User-Session laufen soll (z. B. headless im Schrank). Dann muss zusätzlich ein
> dedizierter Service-User angelegt werden, sonst läuft sie als root.

## Installation

### Voraussetzungen

- macOS
- `make build` (oder `make build-darwin-arm64` / `make build-darwin-amd64`,
  je nach Mac) wurde ausgeführt, sodass `bin/kita-springer` existiert.

### Installation via Script

```bash
./scripts/install-macos-service.sh
```

Das Script tut Folgendes:

1. Legt `/usr/local/kita-springer/` mit `data/`-Unterordner an.
2. Kopiert das Binary aus `bin/kita-springer` dorthin.
3. Entfernt das Gatekeeper-Quarantäne-Attribut (`com.apple.quarantine`).
4. Schreibt die plist nach `~/Library/LaunchAgents/ch.kita-springer.server.plist`.
5. Lädt den Service mit `launchctl load -w` (das `-w` aktiviert auch Auto-Start
   für künftige Reboots/Logins).

Die Listen-Adresse lässt sich vor dem Aufruf via Umgebungsvariable überschreiben:

```bash
ADDR=127.0.0.1:8080 ./scripts/install-macos-service.sh
```

## Verwaltung

```bash
# Status / PID anschauen
launchctl list | grep kita-springer

# Einmalig stoppen (kommt beim nächsten Login wieder)
launchctl stop ch.kita-springer.server

# Komplett deaktivieren (kein Auto-Start mehr)
launchctl unload -w ~/Library/LaunchAgents/ch.kita-springer.server.plist

# Nach Binary-Update neu starten
launchctl kickstart -k gui/$(id -u)/ch.kita-springer.server

# Logs verfolgen
tail -f /usr/local/kita-springer/server.log
tail -f /usr/local/kita-springer/server.err.log
```

## Update

Nach einem `make build` reicht es, das Binary zu ersetzen und den Service neu zu
starten:

```bash
cp bin/kita-springer /usr/local/kita-springer/
launchctl kickstart -k gui/$(id -u)/ch.kita-springer.server
```

Alternativ noch einmal `./scripts/install-macos-service.sh` aufrufen — das Script
ist idempotent.

## Deinstallation

```bash
./scripts/uninstall-macos-service.sh
```

Entfernt die plist und entlädt den Service. **Daten und Logs unter
`/usr/local/kita-springer/` bleiben erhalten** — die müssen bei Bedarf manuell
gelöscht werden:

```bash
sudo rm -rf /usr/local/kita-springer
```

## Wichtige Hinweise

### Initial-Passwort nicht in die plist schreiben

`KITA_INITIAL_PASSWORD` darf **nicht** in die plist als
`EnvironmentVariable` aufgenommen werden — die plist liegt im Klartext im
Home-Verzeichnis. Stattdessen einmalig manuell beim ersten Start setzen:

```bash
KITA_INITIAL_PASSWORD='…' /usr/local/kita-springer/kita-springer
```

Danach steht der bcrypt-Hash in der DB und die Variable wird nicht mehr benötigt.

### Pfade müssen absolut sein

`launchd` versteht in `.plist`-Dateien weder `~` noch relative Pfade. Alle
Einträge in `ProgramArguments`, `WorkingDirectory`, `Standard{Out,Error}Path`
müssen vollständige Pfade sein.

### Gatekeeper-Quarantäne

Wenn das Binary z. B. per AirDrop oder Browser-Download von einem anderen Mac
übertragen wurde, blockiert Gatekeeper die Ausführung. Das Install-Script
entfernt das Attribut automatisch. Manuell:

```bash
xattr -d com.apple.quarantine /usr/local/kita-springer/kita-springer
```

### KeepAlive vs. RunAtLoad

In der plist sind beide auf `true` gesetzt:

- `RunAtLoad`: startet beim Login (bzw. bei `launchctl load`).
- `KeepAlive`: startet automatisch neu, falls der Prozess crasht.

Falls der Service während Wartungsarbeiten *nicht* automatisch neu starten soll,
vorher `launchctl unload …` statt `launchctl stop …` verwenden.

### Logs

`stdout` und `stderr` landen in `server.log` bzw. `server.err.log` im
Installationsverzeichnis. Die Dateien werden **nicht automatisch rotiert** — bei
langlebigen Installationen ggf. via `newsyslog` oder Cron rotieren.
