# Phase 1: Pausen-Dauer-Erfassung — Context

**Gathered:** 2026-05-30
**Status:** Ready for planning
**Source:** /gsd:new-project discussion

<domain>
## Phase Boundary

Pausen-Erfassung von Start+Ende-Zeitpaaren (`actual_break_start/end`) auf reine Dauereingabe in Minuten (`actual_break_minutes`) umstellen — vollständiger vertikaler Schnitt: DB-Migration, Go-Backend, PDF-Export und Vue-Frontend.

Was NICHT in Scope ist: Mehrere Pausen, exakte Pausenuhrzeiten, automatische Pausen-Vorschläge.
</domain>

<decisions>
## Implementation Decisions

### Datenbankschicht
- Migration v12: `ALTER TABLE assignments ADD COLUMN actual_break_minutes INTEGER NOT NULL DEFAULT 0`
- Data-Migration: bestehende `actual_break_start/end` → Minuten (break_end − break_start); leere Werte → 0
- `actual_break_start`, `actual_break_end` bleiben als leere Felder in DB (rückwärtskompatibel)

### Go-Backend
- `models.Assignment`: `ActualBreakStart string`, `ActualBreakEnd string` → `ActualBreakMinutes int`
- `store/assignments.go`: `assignmentSelect`, `scanAssignments`, `CreateAssignment`, `UpdateAssignment` anpassen
- Handler-Validierung (`handlers/validate.go`): `validate.IntRange(a.ActualBreakMinutes, "actual_break_minutes", 0, 600)`
- API-Request: `actual_break_start/end` werden ignoriert oder als 400 abgelehnt

### PDF-Export
- `internal/pdf/worktime.go`: Break-Anzeige als "30 min" statt "10:30–11:00"
- `breakDuration(a)` Hilfsfunktion: `fmt.Sprintf("%d min", a.ActualBreakMinutes)` wenn > 0, sonst "–"

### Frontend — Eingabe
- `AssignmentForm.vue`: zwei `<TimeSelect>` für Pause → Quick-Pick Buttons (0 / 15 / 30 / 45 / 60 min)
- Default-Wert: `currentProvider.value?.min_break_minutes || 0` vorausgewählt beim Laden/Wechsel des Providers
- Formular-State: `actual_break_minutes: 0` statt `actual_break_start/end: ''`

### Frontend — Berechnung
- `time.js` `breakMinutes()`: Parameter wird zu `breakMinutes(breakMin)` → gibt `breakMin` direkt zurück
- `time.js` `netWorkMinutes()`: `(start, breakMin, end)` → `diffMinutes(start, end) - breakMin`
- Alle Aufrufer (`WorktimeTable`, `HistoryView`, `WorktimeView`, `WorktimeChart`) anpassen
- `requiredBreakMinutes()` und `legalMinBreakMinutes()` bleiben unverändert

### Frontend — Anzeige
- `WorktimeTable.vue`: Pausenspalte zeigt `formatHm(a.actual_break_minutes)` statt Zeitbereich
- `HistoryView.vue`, `WorktimeView.vue`: `breakMin = a.actual_break_minutes` direkt
- Compliance-Check: `a.actual_break_minutes < requiredBreakMinutes(gross, providerMin)`

### Claude's Discretion
- Genaue Button-Styles der Quick-Picks (Tailwind-Klassen, active state)
- Ob `actual_break_start/end` in der API als 400 abgelehnt oder still ignoriert werden
- Formatierung "30 min" vs "0:30" im PDF (Empfehlung: "30 min" ist lesbarer)
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Datenbankschicht
- `internal/db/db.go` — migrations-Slice (1-based): Migration v12 hier anfügen
- `internal/models/assignment.go` — Assignment-Struct: ActualBreakStart/End → ActualBreakMinutes
- `internal/store/assignments.go` — assignmentSelect SQL, scanAssignments, Create/Update

### Backend
- `internal/api/handlers/validate.go` — Validierungslogik für Assignment-Felder
- `internal/api/handlers/assignments.go` — CreateAssignment, UpdateAssignment Handler
- `internal/pdf/worktime.go` — Break-Anzeige in PDF-Report

### Frontend
- `frontend/src/components/AssignmentForm.vue` — Hauptformular mit Break-Feldern
- `frontend/src/utils/time.js` — breakMinutes(), netWorkMinutes(), requiredBreakMinutes()
- `frontend/src/components/WorktimeTable.vue` — Break-Spalte
- `frontend/src/views/HistoryView.vue` — Break-Berechnung
- `frontend/src/views/WorktimeView.vue` — Break-Berechnung + Compliance
- `frontend/src/components/WorktimeChart.vue` — netWorkMinutes Aufruf
- `frontend/src/api/index.js` — API-Payload-Format
</canonical_refs>

<specifics>
## Specific Ideas

- Quick-Pick Button-Werte: 0, 15, 30, 45, 60 Minuten
- Provider-Default (`min_break_minutes`) als vorausgewählter Wert
- Swiss ArG Art. 15 Logik bleibt unverändert in `legalMinBreakMinutes()`
- Migration muss idempotent sein (isDuplicateColumn Fehler-Handling existiert bereits)
</specifics>

<deferred>
## Deferred Ideas

- Freies Zahlenfeld für Pausendauer — Quick-Picks reichen
- Mehrere Pausen pro Einsatz — out of scope
- Automatische Pausen-Vorschläge — out of scope
</deferred>

---
*Phase: 01-pausen-dauer-erfassung*
*Context gathered: 2026-05-30 via /gsd:new-project discussion*
