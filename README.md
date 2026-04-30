# Kita Springer Manager

Verwaltung von Springer-Einsätzen in Kitas: Einsatzplanung, Kita-Stammdaten, Öffnungs-/Schliesstage, ÖV-Verbindungen und Kalender-Export.
Aktuell ist die App für einen einzelnen Benutzer konzipiert, der die App auf dem lokalen Laptop oder 
einem Server hostet. Security ist umfangreicht berücksichtigt unter Verwendung von ASVS (Application Security Verification Standard)
incl. Authentifizierung mit Passwort. Wegen Single-User model muss kein Benutzername angegeben werden. 


## Architektur

- **Backend**: Go 1.25 HTTP-Server (`cmd/server`) auf `chi`, SQLite (`modernc.org/sqlite`), REST-API unter `/api/*`. Das gebaute Frontend wird via `go:embed` aus dem Binary ausgeliefert.
- **Frontend**: Vue 3 + Vite + Pinia + TailwindCSS (`frontend/`), kommuniziert mit dem Backend über REST.
- **Scraper**: Go-CLI (`cmd/scraper`) zum Extrahieren von Kita-Daten (Stadt Bern, Stiftung Bern) in Excel-Dateien, die über die Import-API eingelesen werden.
- **ÖV-Daten**: [transport.opendata.ch](https://transport.opendata.ch) (keine API-Keys nötig).

## Voraussetzungen

- Go **1.25+**
- Node.js **18+** und npm
- SQLite (über `modernc.org/sqlite` einkompiliert, keine externe Abhängigkeit)

## Build & Start

### Entwicklung (Backend + Frontend parallel)

```bash
make dev
```

- Backend: <http://localhost:9092>
- Frontend Dev-Server mit HMR: <http://localhost:5173> (proxyt API-Calls ans Backend)

### Nur Backend

```bash
make run                # go run ./cmd/server
```

### Nur Frontend

```bash
make frontend-dev       # vite dev server
make frontend-build     # Production-Build nach frontend/dist
```

### Produktions-Build

```bash
make frontend-build     # zuerst Assets bauen — werden per go:embed eingebettet
make build              # bin/kita-springer
./bin/kita-springer
```

Plattform-spezifische macOS-Builds (z. B. zur Weitergabe):

```bash
make build-darwin-arm64       # Apple Silicon
make build-darwin-amd64       # Intel
make build-darwin-universal   # Universal Binary für beide
```

### Als macOS-Service betreiben

Für den Hintergrundbetrieb (ohne Terminal-Fenster, Auto-Start beim Login) gibt
es ein Install-Script auf Basis von `launchd`. Details und Verwaltungs-Befehle:
[`docs/macos-service.md`](docs/macos-service.md).

```bash
make build
./scripts/install-macos-service.sh
```

### Docker / Cloud-Hosting

Das `Dockerfile` baut ein ~16 MB großes Distroless-Image (statisches Binary,
keine Shell, läuft als nonroot). Frontend + Backend werden in einem mehrstufigen
Build aus dem Repo gebaut.

```bash
docker build -t kita-springer .

# Lokal mit Volume für die DB, Port 80 nach außen, 8080 intern.
docker volume create kita-data
docker run -d --name kita-springer \
    -p 80:8080 \
    -v kita-data:/data \
    -e KITA_INITIAL_PASSWORD='ein-langes-passwort' \
    kita-springer
```

- Container hört intern auf `:8080`. Privilegierte Ports <1024 darf der nonroot-
  User nicht binden — Mapping erfolgt extern via `-p`.
- `/data` ist als `VOLUME` deklariert (DB + Audit-Log). Für Persistenz ein
  Named Volume oder Bind-Mount angeben.
- Hinter einem TLS-Reverse-Proxy betreiben (Basic-Auth-Credentials sind im
  Klartext) — Cloud-Plattformen wie Cloud Run / Fly.io übernehmen TLS automatisch.

Für Pushes nach ghcr.io gibt's ein Multi-Arch-Wrapper-Script (baut amd64+arm64
und pusht in einem Schritt mit `:latest` und `:<git-sha>`-Tag):

```bash
./scripts/docker-push.sh
```

## Releases

### Fertiges Binary herunterladen

Auf der [Releases-Seite](https://github.com/pklotz/kita-springer-manager/releases)
gibt's für jede veröffentlichte Version ein nacktes Binary pro Plattform —
Frontend ist eingebettet, also keine weiteren Dateien nötig:

| Datei | Plattform |
|-------|-----------|
| `kita-springer-vX.Y.Z-linux-amd64`     | Linux x86_64 |
| `kita-springer-vX.Y.Z-linux-arm64`     | Linux arm64 (z. B. Raspberry Pi 4/5) |
| `kita-springer-vX.Y.Z-darwin-arm64`    | macOS Apple Silicon |
| `kita-springer-vX.Y.Z-darwin-amd64`    | macOS Intel |
| `kita-springer-vX.Y.Z-darwin-universal`| macOS Universal (beide Architekturen) |
| `SHA256SUMS.txt`                        | Prüfsummen aller Binaries |

```bash
# Beispiel macOS Apple Silicon:
curl -LO https://github.com/pklotz/kita-springer-manager/releases/download/v0.1.0/kita-springer-v0.1.0-darwin-arm64
chmod +x kita-springer-v0.1.0-darwin-arm64
./kita-springer-v0.1.0-darwin-arm64
```

Auf macOS muss beim ersten Start ggf. die Quarantäne-Markierung entfernt werden
(`xattr -d com.apple.quarantine <binary>`), weil das Binary nicht signiert ist.

### Neues Release erstellen

Versionierung folgt [SemVer](https://semver.org) (`vMAJOR.MINOR.PATCH`):
**PATCH** für Bugfixes, **MINOR** für rückwärtskompatible Features, **MAJOR**
für Breaking Changes. Solange das Projekt im `0.x.y`-Bereich ist, dürfen
Breaking Changes auch in MINOR-Bumps mit.

```bash
git tag v0.1.0
git push origin v0.1.0
make release
```

`make release` führt `scripts/release.sh` aus und macht folgendes:

1. Validiert: Tag liegt auf HEAD (oder `VERSION=…` mitgegeben), Format `vX.Y.Z[-prerelease]`, Working Tree clean, `gh` ist eingeloggt.
2. Baut das Frontend einmal (`go:embed` zieht es in jedes Binary).
3. Cross-compiled für `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` (pure Go, kein CGO).
4. Wenn `lipo` verfügbar: zusätzlich `darwin-universal`.
5. Erzeugt `SHA256SUMS.txt` und legt das GitHub-Release mit `gh release create` an.

Voraussetzungen einmalig: [`gh` CLI](https://cli.github.com/) installiert und
`gh auth login` gemacht.

Escape hatches:

```bash
VERSION=v0.1.0 make release    # ohne Tag auf HEAD
ALLOW_DIRTY=1   make release   # ungebackte lokale Modifikationen erlauben
OVERWRITE=1     make release   # bestehendes Release ersetzen (Tag bleibt)
```

### Konfiguration

Server-Flags bzw. Env-Variablen (Flag hat Vorrang):

| Flag       | Env                       | Default              | Zweck                                            |
|------------|---------------------------|----------------------|--------------------------------------------------|
| `--addr`   | `ADDR`                    | `127.0.0.1:9092`     | HTTP Listen-Adresse (nur loopback ist Default)   |
| `--db`     | `DB_PATH`                 | `data/app.db`        | Pfad zur SQLite-Datenbank                        |
| –          | `KITA_INITIAL_USERNAME`   | `admin`              | Initial-Benutzer (nur beim ersten Start)         |
| –          | `KITA_INITIAL_PASSWORD`   | –                    | Initial-Passwort (nur beim ersten Start)         |

Beispiel (Headless-Bootstrap):

```bash
KITA_INITIAL_PASSWORD='ein-langes-passwort' ADDR=127.0.0.1:9092 ./bin/kita-springer
```

## Authentifizierung & Internet-Betrieb

Die App nutzt **HTTP Basic Auth** (single-user). Das Passwort wird mit `bcrypt`
gehasht und im `settings`-KV der SQLite-DB abgelegt.

- **Erster Start:** Es ist noch kein Passwort gesetzt. Die UI zeigt automatisch
  ein Setup-Formular. Alternativ: `KITA_INITIAL_PASSWORD` als Env-Variable
  vorgeben (Headless-/Container-Setups).
- **Folgestart:** Der Browser fragt nach Benutzer und Passwort (Basic-Auth-Dialog).
- **Passwort ändern:** Über *Einstellungen → Passwort ändern* in der UI.

### Pflicht: HTTPS-Reverse-Proxy

Basic-Auth-Credentials gehen bei jeder Anfrage als Klartext-Header über die
Leitung. Die App **muss** im Internet hinter einem TLS-terminierenden Reverse
Proxy laufen (Caddy, nginx, Traefik, …). Der Default-Bind ist `127.0.0.1:9092`
— der Proxy verbindet sich lokal.

Beispiel `Caddyfile`:

```
kita.example.com {
    reverse_proxy 127.0.0.1:9092
}
```

Caddy holt automatisch ein Let's-Encrypt-Zertifikat. Setze `ADDR=:9092` nur,
wenn du bewusst direkt ans Netz binden willst (dann eigenes TLS oder Tunnel
notwendig).

### iPhone-Zugriff

- **Web-UI:** `https://kita.example.com` im Safari öffnen — der iOS-Basic-Auth-Dialog
  taucht einmal auf, danach werden die Credentials gecached.
- **Kalender-Abo (iOS Calendar):** `webcal://benutzer:passwort@kita.example.com/api/calendar.ics`
  abonnieren. Apple Calendar akzeptiert Credentials in der URL.
- **PDF-Export:** Wird aus der UI heraus heruntergeladen (Browser sendet
  gecachte Credentials beim Klick auf den Download-Link automatisch mit).

## Scraper

Extrahiert Kita-Daten von Provider-Websites in eine Excel-Datei, die über `POST /api/kitas/import` oder `POST /api/providers/{id}/import-excel` eingelesen werden kann.

```bash
go run ./cmd/scraper --source=stadt_bern     --output=kitas_stadt_bern.xlsx
go run ./cmd/scraper --source=stiftung_bern  --output=kitas_stiftung_bern.xlsx
```

## Backup-CLI

Für Backup/Restore ohne laufenden Server (z. B. Cron-Snapshot, Migration zwischen
Maschinen). Funktional identisch zum Web-UI in *Einstellungen → Datenbank-Backup*.

Schnellster Weg via Wrapper-Script — baut die Go-CLI bei Bedarf automatisch:

```bash
# Backup erstellen (default: data/app.db → kita-springer-YYYY-MM-DD.db)
./scripts/backup.sh export

# Backup-Datei prüfen, ohne irgendwas zu ändern
./scripts/backup.sh verify -in kita-springer-2026-04-27.db

# Restore — VOR DEM AUFRUF DEN SERVER STOPPEN
./scripts/backup.sh restore -in kita-springer-2026-04-27.db -y
```

Direkt das Binary bauen und nutzen:

```bash
make build-backup    # → bin/kita-springer-backup
./bin/kita-springer-backup export --db data/app.db
```

Mit reinem `sqlite3` geht's auch in einer Zeile:

```bash
sqlite3 data/app.db "VACUUM INTO 'kita-springer-$(date +%F).db'"
```

Der CLI-Vorteil gegenüber `sqlite3` direkt: validiert das Schema vor dem Restore,
kennt den Default-DB-Pfad und braucht kein extern installiertes `sqlite3`-Paket
(z. B. im Distroless-Container).

## REST-API (Referenz)

Alle Routen unter `/api`.

### Providers
- `GET /providers` — Liste
- `POST /providers` — anlegen
- `PUT /providers/{id}` — aktualisieren
- `DELETE /providers/{id}` — löschen
- `POST /providers/{id}/seed-kitas` — Seeds aus `internal/seeds/*.json` einspielen
- `POST /providers/{id}/import-excel` — Excel-Import (multipart)

### Kitas
- `GET /kitas` · `POST /kitas` · `GET /kitas/{id}` · `PUT /kitas/{id}` · `DELETE /kitas/{id}`
- `POST /kitas/import` — Excel-Import

### Assignments (Springer-Einsätze)
- `GET /assignments` · `POST /assignments` · `GET/PUT/DELETE /assignments/{id}`
- `POST /assignments/bulk-delete`

### Recurring / Closures / Settings
- `GET|POST /recurring`, `DELETE /recurring/{id}`
- `GET|POST /closures`, `DELETE /closures/{id}`
- `GET|PUT /settings`

### Transit
- `GET /transit/connections` — ÖV-Verbindungen (cached)
- `GET /transit/stops` — Haltestellensuche

### Kalender
- `GET /calendar.ics` — iCalendar-Export aller Einsätze

## Projektstruktur

```
cmd/
  server/       HTTP-Server
  scraper/      CLI zum Extrahieren von Kita-Daten
internal/
  api/          Router & Handler (chi)
  db/           SQLite-Verbindung + Migrationen
  importer/     Excel-Import
  models/       Datenstrukturen
  seeds/        JSON-Seeds für initialen Bestand
  store/        Datenzugriff (CRUD)
  transit/      Client für transport.opendata.ch
frontend/       Vue 3 + Vite App (per go:embed eingebettet)
data/           SQLite-DB (zur Laufzeit angelegt)
```

## Datenbank

SQLite mit versionierten Migrationen in `internal/db/db.go`. Beim Start werden:

- Migrationen automatisch angewendet
- Feiertage für das aktuelle und die zwei folgenden Jahre geseedet
- Gecachte ÖV-Verbindungen vergangener Einsätze aufgeräumt

## Lizenz

Siehe `LICENSE`.
