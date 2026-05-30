---
phase: "01"
plan: "01"
subsystem: "break-duration"
tags: [db-migration, go-backend, pdf, vue-frontend]
key-files:
  created: []
  modified:
    - internal/db/db.go
    - internal/models/assignment.go
    - internal/store/assignments.go
    - internal/api/handlers/validate.go
    - internal/pdf/worktime.go
    - frontend/src/utils/time.js
    - frontend/src/components/AssignmentForm.vue
    - frontend/src/components/WorktimeTable.vue
    - frontend/src/views/HistoryView.vue
    - frontend/src/views/WorktimeView.vue
    - frontend/src/components/WorktimeChart.vue
metrics:
  tasks_completed: 8
  tasks_total: 8
  commits: 8
---

# Plan 01-01 Summary: Pausen-Dauer-Erfassung

## Objective

Vollständige Umstellung der Pausen-Erfassung von Start+Ende-Zeitpaaren auf reine Dauer in Minuten — DB-Migration, Backend, PDF und Frontend als vertikaler Schnitt.

## Commits

| Task | Commit | Beschreibung |
|------|--------|-------------|
| 1 | fd41320 | DB-Migration v12 — actual_break_minutes Spalte |
| 2 | edbbcf0 | Assignment-Model — ActualBreakMinutes statt Start/End |
| 3 | cf80cb6 | Store — assignmentSelect/scan/Create/Update auf actual_break_minutes |
| 4 | 7fbde40 | Validierung — IntRange für actual_break_minutes (0–600 min) |
| 5 | fda8a1c | PDF — Pause als '30 min' statt Zeitbereich |
| 6 | a315cf8 | time.js — breakMinutes/netWorkMinutes auf Dauer-Basis |
| 7 | 1ec56cc | AssignmentForm — Quick-Pick Buttons für Pausendauer |
| 8 | c7bf3d4 | Consumer-Views auf actual_break_minutes umgestellt |

## What Was Built

- **DB v12**: `actual_break_minutes INTEGER NOT NULL DEFAULT 0` + Datenmigration via strftime
- **Go-Model**: `ActualBreakMinutes int` ersetzt `ActualBreakStart/End string`
- **Store**: Alle SQL-Abfragen und Scan-Aufrufe angepasst
- **Validierung**: `validate.IntRange(0–600)` statt zwei TimeHM-Felder
- **PDF**: Pause wird als "30 min" angezeigt (statt "10:30–11:00")
- **time.js**: `breakMinutes(min)` / `netWorkMinutes(start, min, end)` — neue Signaturen
- **AssignmentForm**: 5 Quick-Pick-Buttons (Keine/15/30/45/60 min), kein TimeSelect mehr für Pause
- **Alle Consumer**: WorktimeTable, HistoryView, WorktimeView, WorktimeChart auf neue Signaturen umgestellt

## Deviations

- **morningRange** in WorktimeTable: Da keine separate Pause-Zeitangabe mehr vorhanden, zeigt die Spalte nur noch Start–Ende (kein Vormittag/Nachmittag-Split mehr). Die `afternoonRange`-Funktion gibt jetzt immer `''` zurück — konsistent mit dem neuen Modell.
- **copyPlanToActual**: Setzt nun den Provider-Default (`min_break_minutes`) als Pause-Vorgabe statt `''`.

## Self-Check

- [x] `go build ./...` — OK
- [x] `cd frontend && npm run build` — OK (✓ built in 1.11s)
- [x] Keine Referenzen mehr auf `actual_break_start`/`actual_break_end` im Frontend
- [x] Keine Referenzen mehr auf `ActualBreakStart`/`ActualBreakEnd` im Go-Code
- [x] `validate.IntRange(a.ActualBreakMinutes, "actual_break_minutes", 0, 600)` vorhanden
- [x] DB-Migration v12 als letzter Eintrag im migrations-Slice (index 11)

## Self-Check: PASSED
