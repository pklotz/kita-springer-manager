# syntax=docker/dockerfile:1.7

# ---------- Stage 1: Frontend bauen ----------
# --platform=$BUILDPLATFORM zwingt diese Stage auf die Host-Architektur — das
# Frontend-Build ist arch-unabhängig (JS/CSS), also keine Notwendigkeit für
# QEMU-Emulation beim Multi-Arch-Build.
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend

WORKDIR /app/frontend

# Lockfile-Layer cachen, damit Dep-Installation nur bei package.json-Änderungen läuft.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build


# ---------- Stage 2: Backend bauen (statisch, ohne CGO) ----------
# Builder läuft ebenfalls auf der Host-Architektur und cross-compiliert via
# GOARCH zur Ziel-Plattform — schneller als unter QEMU zu emulieren.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Module-Layer cachen.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Frontend-Build-Output muss vor `go build` an seinem Platz sein, weil cmd/server
# die Assets per `go:embed` einzieht.
COPY --from=frontend /app/frontend/dist ./frontend/dist

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/kita-springer \
        ./cmd/server

# Leeres /data-Skelett mit korrekter Ownership vorbauen — distroless hat
# weder Shell noch chown, deshalb hier vorbereiten und unten per --chown
# rüberkopieren.
RUN mkdir -p /out/data


# ---------- Stage 3: Minimaler Runtime-Container ----------
# distroless/static reicht: keine libc, keine Shell, nur CA-Zertifikate und
# tzdata. Funktioniert nur, weil modernc.org/sqlite pure Go ist (kein CGO).
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=backend --chown=nonroot:nonroot /out/kita-springer /app/kita-springer
# /data (DB + Audit-Log) muss dem nonroot-User gehören, sonst schreibt der
# Server nichts. Inhalt wird beim ersten `docker run` ins Volume übernommen.
COPY --from=backend --chown=nonroot:nonroot /out/data /data

# Listen-Adresse: 0.0.0.0:8080 (Default 127.0.0.1 wäre im Container nicht
# erreichbar). Privilegierte Ports <1024 darf der nonroot-User nicht binden —
# Mapping nach 80 erfolgt extern via `docker run -p 80:8080 …`.
ENV ADDR=:8080
ENV DB_PATH=/data/app.db

# Persistente SQLite-DB liegt unter /data — bei `docker run` ein Volume mounten.
# Der nonroot-User (uid 65532) muss Schreibrechte haben.
VOLUME ["/data"]

EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/kita-springer"]
