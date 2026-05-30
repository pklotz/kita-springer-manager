# Roadmap: Kita Springer Manager — Pausen-Umstellung

**Project:** Pausen-Erfassung von Start+Ende-Zeiten auf reine Dauer umstellen
**Granularity:** Coarse
**Created:** 2026-05-30

---

## Phases

### Phase 1: Pausen-Dauer-Erfassung

**Goal:** Die gesamte Pausen-Erfassung auf reine Dauer (Minuten) umstellen — DB-Migration, Backend, PDF und Frontend als vollständiger vertikaler Schnitt.

**Mode:** mvp

**Requirements:** DB-01, DB-02, BE-01, BE-02, BE-03, BE-04, FE-01, FE-02, FE-03, FE-04, FE-05, FE-06

**Success Criteria:**
1. Benutzer kann eine Pause über Quick-Pick Buttons (0/15/30/45/60 min) erfassen — kein Start/Ende-Zeitpaar mehr sichtbar
2. Provider-Default (`min_break_minutes`) ist beim Öffnen des Formulars automatisch vorausgewählt
3. Netto-Arbeitszeit berechnet sich korrekt als `(end − start) − break_minutes`
4. Swiss-ArG-Compliance-Prüfung funktioniert weiterhin korrekt
5. PDF-Arbeitszeitnachweis zeigt Pause als "30 min" (oder entsprechende Dauer)
6. Bestehende Einsätze mit `actual_break_start/end`-Daten werden korrekt migriert (Dauer erhalten)

**Canonical refs:**
- `internal/db/db.go` — Migration v12 hier anfügen (migrations-Slice, 1-based Index)
- `internal/models/assignment.go` — `ActualBreakStart/End` → `ActualBreakMinutes`
- `internal/store/assignments.go` — `assignmentSelect`, `CreateAssignment`, `UpdateAssignment`, `scanAssignments`
- `internal/api/handlers/validate.go` — Break-Validierung anpassen
- `internal/pdf/worktime.go` — Break-Anzeige im PDF
- `frontend/src/components/AssignmentForm.vue` — zwei TimeSelect → Quick-Pick Buttons
- `frontend/src/utils/time.js` — `breakMinutes()`, `netWorkMinutes()` anpassen
- `frontend/src/components/WorktimeTable.vue` — Pausenspalte
- `frontend/src/views/HistoryView.vue` — Break-Berechnung
- `frontend/src/views/WorktimeView.vue` — Break-Berechnung + Compliance
- `frontend/src/components/WorktimeChart.vue` — Break-Berechnung

---

## Requirement Coverage

| Requirement | Phase | Status |
|-------------|-------|--------|
| DB-01 | Phase 1 | Pending |
| DB-02 | Phase 1 | Pending |
| BE-01 | Phase 1 | Pending |
| BE-02 | Phase 1 | Pending |
| BE-03 | Phase 1 | Pending |
| BE-04 | Phase 1 | Pending |
| FE-01 | Phase 1 | Pending |
| FE-02 | Phase 1 | Pending |
| FE-03 | Phase 1 | Pending |
| FE-04 | Phase 1 | Pending |
| FE-05 | Phase 1 | Pending |
| FE-06 | Phase 1 | Pending |

**Coverage:** 12/12 v1 requirements mapped ✓

---
*Roadmap created: 2026-05-30*
