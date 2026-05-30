# Kita Springer Manager — Pausen-Umstellung

## What This Is

Kita Springer Manager ist eine selbst-gehostete Einsatzplanungs-App für Kita-Springerinnen in Bern. Sie verwaltet Einsätze bei verschiedenen Kitas, Anbieter-Importe (Excel), Arbeitszeitnachweise (PDF), Fahrverbindungen und Kalender-Exporte. Die App besteht aus einem Go-Backend mit eingebettetem Vue 3 Frontend und SQLite-Datenbank.

Dieses Planungsprojekt fokussiert auf eine gezielte Verbesserung der Pausen-Erfassung.

## Core Value

Pausen korrekt in die Arbeitszeitberechnung einfliessen lassen — als Dauer, nicht als Zeitraum.

## Requirements

### Validated

- ✓ Einsatz-Verwaltung (erstellen, bearbeiten, löschen) — bestehend
- ✓ Anbieter-Management mit Farben und Excel-Konfiguration — bestehend
- ✓ Kita-Datenbank mit ÖV-Haltestellen und Geocoding — bestehend
- ✓ Arbeitszeitnachweis als PDF (gruppiert nach Anbieter) — bestehend
- ✓ Kalender-Export (iCal) mit Download-Token — bestehend
- ✓ HTTP Basic Auth mit bcrypt, Setup-Flow — bestehend
- ✓ Swiss ArG Art. 15 Pausen-Logik (legalMinBreakMinutes) — bestehend
- ✓ Provider-Default `min_break_minutes` — bestehend

### Active

- [ ] DB-Migration: `actual_break_start/end` → `actual_break_minutes INTEGER`
- [ ] Go-Model und Store: ein Feld statt zwei, automatische Umrechnung bestehender Daten
- [ ] UI: Zwei TimeSelect-Felder ersetzen durch Quick-Pick Buttons (0 / 15 / 30 / 45 / 60 min), vorausgefüllt mit Provider-Default
- [ ] Netto-Arbeitszeitberechnung anpassen: `(end − start) − break_minutes`
- [ ] PDF-Export: Pause als Dauer anzeigen ("30 min") statt Zeitbereich
- [ ] WorktimeTable und HistoryView: Pausenspalte als Dauer

### Out of Scope

- Exakte Pausenuhrzeit (Start/Ende) aufzeichnen — Dauer ist das Einzige was gesetzlich und abrechnungstechnisch relevant ist
- Mehrere Pausen pro Einsatz — eine Pause pro Einsatz ist ausreichend
- Automatische Pausen-Vorschläge basierend auf Arbeitszeit — bestehende Compliance-Anzeige im UI reicht

## Context

**Bestehende Codebase:**
- Backend: Go 1.25, chi router, SQLite (modernc — kein CGO)
- Frontend: Vue 3 + Pinia + Tailwind, axios mit Basic-Auth-Interceptor
- Pause heute: `actual_break_start TEXT` + `actual_break_end TEXT` in `assignments`
- Dauer-Konzept bereits vorhanden: `min_break_minutes INTEGER` auf `providers`
- `time.js` hat `breakMinutes()`, `netWorkMinutes()`, `legalMinBreakMinutes()` — alle arbeiten bereits auf Dauer-Basis
- `WorktimeChart.vue`, `WorktimeTable.vue`, `HistoryView.vue`, `WorktimeView.vue` verwenden Break-Felder
- `AssignmentForm.vue` hat zwei `<TimeSelect>` für Pause
- PDF (`internal/pdf/worktime.go`) zeigt heute "HH:MM–HH:MM" für Pausen

**Migrationsstrategie:**
- Migration v12: `ALTER TABLE assignments ADD COLUMN actual_break_minutes INTEGER NOT NULL DEFAULT 0`
- Data-Migration: `UPDATE assignments SET actual_break_minutes = <computed_duration>` aus bestehenden Start/End-Werten
- Danach: `actual_break_start`, `actual_break_end` können leer bleiben (rückwärtskompatibel) oder via folge-Migration entfernt werden

**Swiss ArG Pausen-Regeln (bereits implementiert in `legalMinBreakMinutes`):**
- >5.5h Arbeitszeit → mind. 15 min
- >7h → mind. 30 min
- >9h → mind. 60 min

## Constraints

- **Kompatibilität**: Migration muss bestehende Daten erhalten — `break_end − break_start` in Minuten umrechnen; leere Werte → 0
- **Kein CGO**: `modernc.org/sqlite` — bleibt so
- **Single-User**: Keine Multi-Tenancy-Überlegungen nötig
- **Rückwärtskompatibilität API**: `actual_break_start`/`actual_break_end` aus dem API-Request können ignoriert oder abgelehnt werden

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Dauer statt Zeitraum | Gesetzliche Anforderungen und Abrechnung brauchen nur Dauer; exakte Uhrzeit fügt nur UX-Reibung hinzu | — Pending |
| Quick-Pick Buttons (0/15/30/45/60) | Häufigste Werte mit einem Klick; Provider-Default vorausgefüllt | — Pending |
| PDF zeigt "30 min" statt Zeitbereich | Einfacher und klarer für Arbeitszeitnachweise | — Pending |
| Automatische Migration bestehender Daten | Bestehende break_start/end → Minuten umgerechnet; kein Datenverlust | — Pending |

## Evolution

Dieses Dokument entwickelt sich bei Phasenübergängen und Milestone-Abschlüssen.

**Nach jedem Phasenübergang** (via `/gsd-transition`):
1. Ungültig gewordene Requirements? → In Out of Scope verschieben mit Begründung
2. Validierte Requirements? → In Validated verschieben mit Phasen-Referenz
3. Neue Requirements aufgetaucht? → In Active ergänzen
4. Entscheidungen zu dokumentieren? → In Key Decisions eintragen
5. "What This Is" noch aktuell? → Anpassen wenn abgedriftet

**Nach jedem Milestone** (via `/gsd:complete-milestone`):
1. Vollständige Überprüfung aller Abschnitte
2. Core Value prüfen — noch die richtige Priorität?
3. Out of Scope auditieren — Begründungen noch valide?
4. Context mit aktuellem Stand aktualisieren

---
*Last updated: 2026-05-30 after initialization*
