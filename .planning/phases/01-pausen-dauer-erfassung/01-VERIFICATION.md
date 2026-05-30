---
phase: "01"
status: passed
verified: 2026-05-30
must_haves_score: 6/6
---

# Verification: Phase 01 — Pausen-Dauer-Erfassung

## Must-Haves Verification

| # | Must-Have | Status |
|---|-----------|--------|
| 1 | `actual_break_minutes INTEGER NOT NULL DEFAULT 0` existiert in assignments-Tabelle (DB-Migration v12, letzter Eintrag Index 11) | PASS |
| 2 | Bestehende break_start/end-Daten wurden korrekt in Minuten migriert (strftime-basiertes UPDATE in v12) | PASS |
| 3 | `AssignmentForm` zeigt Quick-Pick Buttons (0/15/30/45/60), kein TimeSelect für Pause | PASS |
| 4 | `netWorkMinutes()` berechnet: `diffMinutes(start,end) - actual_break_minutes` | PASS |
| 5 | PDF zeigt Pause als "30 min" statt Zeitbereich (`pauseLabel` → `fmt.Sprintf("%d min", ...)`) | PASS |
| 6 | Swiss ArG Compliance-Check funktioniert weiterhin (`requiredBreakMinutes` / `legalMinBreakMinutes` unverändert) | PASS |

## Requirements Coverage

| Req-ID | Beschreibung | Status | Fundstelle |
|--------|-------------|--------|-----------|
| DB-01 | Migration v12 fügt `actual_break_minutes INTEGER NOT NULL DEFAULT 0` hinzu | PASS | `internal/db/db.go` Z. 153 — letzter Eintrag im migrations-Slice (Index 11) |
| DB-02 | Migration rechnet bestehende break_start/end → Minuten um; fehlende Werte = 0 | PASS | `internal/db/db.go` Z. 154–162 — strftime-basiertes UPDATE mit WHERE-Guard |
| BE-01 | `Assignment`-Struct enthält `ActualBreakMinutes int` statt Start/End | PASS | `internal/models/assignment.go` Z. 25 — `ActualBreakMinutes int json:"actual_break_minutes"` |
| BE-02 | Store-Funktionen (`Create`, `Update`, `scan`) verwenden das neue Feld | PASS | `internal/store/assignments.go` — `assignmentSelect` Z. 17, `scanAssignments` Z. 74, `CreateAssignment` Z. 173, `UpdateAssignment` Z. 189 |
| BE-03 | Validierung: `actual_break_minutes` als Integer 0–600 | PASS | `internal/api/handlers/validate.go` Z. 44 — `validate.IntRange(a.ActualBreakMinutes, "actual_break_minutes", 0, 600)` |
| BE-04 | PDF zeigt Pause als Dauer ("30 min") statt Zeitbereich | PASS | `internal/pdf/worktime.go` Z. 224–229 — `pauseLabel` gibt `fmt.Sprintf("%d min", a.ActualBreakMinutes)` zurück |
| FE-01 | `AssignmentForm.vue` zeigt Quick-Pick Buttons (0/15/30/45/60 min), kein TimeSelect für Pause | PASS | `frontend/src/components/AssignmentForm.vue` Z. 72–81 |
| FE-02 | Quick-Pick-Default ist `min_break_minutes` des aktuellen Anbieters | PASS | `AssignmentForm.vue` Z. 147 — `copyPlanToActual` setzt `currentProvider.value?.min_break_minutes \|\| 0` |
| FE-03 | `time.js` `netWorkMinutes(start, breakMin, end)` und `breakMinutes(breakMin)` mit neuen Signaturen | PASS | `frontend/src/utils/time.js` Z. 29–37 — ein-Parameter-`breakMinutes`, drei-Parameter-`netWorkMinutes` |
| FE-04 | `WorktimeTable.vue` zeigt Pause als Dauer statt Zeitbereich | PASS | `frontend/src/components/WorktimeTable.vue` Z. 105–107 — `breakLabel` gibt `formatHm(breakMin(a))` zurück |
| FE-05 | `HistoryView.vue` und `WorktimeView.vue` verwenden neues Feld für Netto-Berechnung und Compliance | PASS | `HistoryView.vue` Z. 138–147; `WorktimeView.vue` Z. 186–193 |
| FE-06 | `WorktimeChart.vue` verwendet neues Feld | PASS | `frontend/src/components/WorktimeChart.vue` Z. 121 — `netWorkMinutes(a.actual_start_time, a.actual_break_minutes, a.actual_end_time)` |

## Issues Found

Keine. Die einzigen verbleibenden Vorkommen von `actual_break_start`/`actual_break_end` befinden sich ausschliesslich in `internal/db/db.go`:

- **v8-Migration** (Z. 128–129): fügt die alten Spalten hinzu — korrekt und notwendig für Rückwärtskompatibilität der DB-Struktur.
- **v12-Migration** (Z. 156–162): liest die alten Werte zur Datenmigration — korrekt und beabsichtigt gemäss Plan.

Es gibt keine Referenzen auf `ActualBreakStart`/`ActualBreakEnd` im Go-Applikationscode (Models, Store, Handler, PDF) und keine Referenzen auf `actual_break_start`/`actual_break_end` im Frontend-Code.

`go build ./...` kompiliert ohne Fehler oder Warnungen.

## Verification Complete

**Gesamtergebnis: PASSED** — Alle 12 Requirements erfüllt, alle 6 Must-Haves bestätigt. Die Phase ist vollständig und korrekt implementiert.
