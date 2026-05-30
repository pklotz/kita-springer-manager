# Requirements: Kita Springer Manager — Pausen-Umstellung

**Defined:** 2026-05-30
**Core Value:** Pausen korrekt in die Arbeitszeitberechnung einfliessen lassen — als Dauer, nicht als Zeitraum.

## v1 Requirements

### Datenbankschicht

- [ ] **DB-01**: DB-Migration v12 fügt `actual_break_minutes INTEGER NOT NULL DEFAULT 0` zur `assignments`-Tabelle hinzu
- [ ] **DB-02**: Migration rechnet bestehende `actual_break_start/end`-Werte automatisch in Minuten um (break_end − break_start); fehlende Werte ergeben 0

### Backend

- [ ] **BE-01**: Go-Model `Assignment` ersetzt `ActualBreakStart/End string` durch `ActualBreakMinutes int`
- [ ] **BE-02**: Store-Funktionen (`CreateAssignment`, `UpdateAssignment`, `scanAssignments`) verwenden das neue Feld
- [ ] **BE-03**: Handler-Validierung: `actual_break_minutes` als Integer (0–600 min) statt zwei Zeitfelder
- [ ] **BE-04**: PDF-Export zeigt Pause als Dauer ("30 min") statt Zeitbereich ("10:30–11:00")

### Frontend

- [ ] **FE-01**: `AssignmentForm.vue` ersetzt die zwei `<TimeSelect>`-Felder für Pause durch Quick-Pick Buttons (0 / 15 / 30 / 45 / 60 min)
- [ ] **FE-02**: Quick-Pick-Default ist automatisch der `min_break_minutes`-Wert des aktuellen Anbieters
- [ ] **FE-03**: `time.js` `netWorkMinutes()` und `breakMinutes()` arbeiten mit dem neuen Dauer-Feld (kein Start/End mehr)
- [ ] **FE-04**: `WorktimeTable.vue` zeigt Pausenspalte als Dauer ("30 min") statt Zeitbereich
- [ ] **FE-05**: `HistoryView.vue` und `WorktimeView.vue` verwenden neues Feld für Netto-Berechnung und Compliance-Check
- [ ] **FE-06**: `WorktimeChart.vue` verwendet neues Feld

## v2 Requirements

*(Keine — Scope ist abgeschlossen)*

## Out of Scope

| Feature | Reason |
|---------|--------|
| Exakte Pausenuhrzeit (Start/Ende) aufzeichnen | Nur Dauer ist gesetzlich und abrechnungstechnisch relevant |
| Mehrere Pausen pro Einsatz | Eine Pause reicht; kein identifiziertes Bedürfnis |
| Automatische Pausen-Vorschläge | Bestehende Compliance-Anzeige im UI ist ausreichend |
| Freies Zahlenfeld für Pausendauer | Quick-Picks (0/15/30/45/60) decken alle relevanten Werte ab |

## Traceability

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

**Coverage:**
- v1 requirements: 12 total
- Mapped to phases: 12
- Unmapped: 0 ✓

---
*Requirements defined: 2026-05-30*
*Last updated: 2026-05-30 after initial definition*
