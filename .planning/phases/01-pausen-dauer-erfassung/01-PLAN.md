---
wave: 1
depends_on: []
files_modified:
  - internal/db/db.go
  - internal/models/assignment.go
  - internal/store/assignments.go
  - internal/api/handlers/validate.go
  - internal/api/handlers/assignments.go
  - internal/pdf/worktime.go
  - frontend/src/utils/time.js
  - frontend/src/components/AssignmentForm.vue
  - frontend/src/components/WorktimeTable.vue
  - frontend/src/views/HistoryView.vue
  - frontend/src/views/WorktimeView.vue
  - frontend/src/components/WorktimeChart.vue
autonomous: true
requirements_addressed:
  - DB-01
  - DB-02
  - BE-01
  - BE-02
  - BE-03
  - BE-04
  - FE-01
  - FE-02
  - FE-03
  - FE-04
  - FE-05
  - FE-06
---

# Plan 01: Pausen-Umstellung auf Dauer-Erfassung

## Objective

Vollständige Umstellung der Pausen-Erfassung von Start+Ende-Zeitpaaren auf reine Dauer in Minuten — DB-Migration, Backend, PDF und Frontend als ein zusammenhängender vertikaler Schnitt.

## Must-Haves

```yaml
must_haves:
  goal: "Pausen korrekt als Dauer (Integer-Minuten) erfassen und verarbeiten"
  truths:
    - "actual_break_minutes INTEGER NOT NULL DEFAULT 0 existiert in assignments-Tabelle"
    - "Bestehende break_start/end-Daten wurden korrekt in Minuten migriert"
    - "AssignmentForm zeigt Quick-Pick Buttons (0/15/30/45/60), kein TimeSelect für Pause"
    - "netWorkMinutes() berechnet: diffMinutes(start,end) - actual_break_minutes"
    - "PDF zeigt Pause als '30 min' statt Zeitbereich"
    - "Swiss ArG Compliance-Check funktioniert weiterhin korrekt"
```

## Tasks

### Task 1: DB-Migration v12

<read_first>
- `internal/db/db.go` — migrations-Slice, aktuelle Länge prüfen (v11 ist letzter Eintrag), isDuplicateColumn Handling
</read_first>

<action>
Füge Migration v12 am Ende des migrations-Slice in `internal/db/db.go` hinzu:

```go
// v12: break duration in minutes instead of start/end times
{
    `ALTER TABLE assignments ADD COLUMN actual_break_minutes INTEGER NOT NULL DEFAULT 0`,
    // Migrate existing break data: compute duration from existing start/end times.
    // strftime converts HH:MM to minutes-since-midnight; difference = break duration.
    // Rows with missing start or end keep the default 0.
    `UPDATE assignments
     SET actual_break_minutes = (
         CAST(strftime('%H', actual_break_end) AS INTEGER) * 60 +
         CAST(strftime('%M', actual_break_end) AS INTEGER) -
         CAST(strftime('%H', actual_break_start) AS INTEGER) * 60 -
         CAST(strftime('%M', actual_break_start) AS INTEGER)
     )
     WHERE actual_break_start != '' AND actual_break_end != ''
       AND actual_break_end > actual_break_start`,
},
```
</action>

<acceptance_criteria>
- `internal/db/db.go` enthält eine migration mit `actual_break_minutes INTEGER NOT NULL DEFAULT 0`
- UPDATE-Statement referenziert `actual_break_start` und `actual_break_end` zur Berechnung
- Migration ist der letzte Eintrag im migrations-Slice (index 11, v12)
- `isDuplicateColumn` Fehler-Handling greift wenn Migration bereits gelaufen
</acceptance_criteria>

---

### Task 2: Go-Model anpassen

<read_first>
- `internal/models/assignment.go` — aktuelles Assignment-Struct, alle Felder
</read_first>

<action>
In `internal/models/assignment.go`: ersetze `ActualBreakStart string` und `ActualBreakEnd string` durch `ActualBreakMinutes int` mit JSON-Tag `json:"actual_break_minutes"`.

Entferne die beiden alten Felder vollständig aus dem Struct. Das Struct behält alle anderen Felder (ActualStartTime, ActualEndTime, etc.) unverändert.
</action>

<acceptance_criteria>
- `models.Assignment` enthält `ActualBreakMinutes int` mit `json:"actual_break_minutes"`
- `ActualBreakStart` und `ActualBreakEnd` sind aus dem Struct entfernt
- Kein anderes Feld im Struct wurde verändert
</acceptance_criteria>

---

### Task 3: Store-Layer anpassen

<read_first>
- `internal/store/assignments.go` — `assignmentSelect` Konstante, `scanAssignments`, `CreateAssignment`, `UpdateAssignment` vollständig lesen
</read_first>

<action>
In `internal/store/assignments.go`:

1. `assignmentSelect`: ersetze `COALESCE(a.actual_break_start,''), COALESCE(a.actual_break_end,''),` durch `COALESCE(a.actual_break_minutes,0),`

2. `scanAssignments`: ersetze `&a.ActualBreakStart, &a.ActualBreakEnd,` durch `&a.ActualBreakMinutes,` in der Scan-Zeile

3. `CreateAssignment` INSERT-Statement: ersetze `actual_break_start, actual_break_end,` durch `actual_break_minutes,` in Spalten-Liste und Werte-Liste (`a.ActualBreakStart, a.ActualBreakEnd,` → `a.ActualBreakMinutes,`)

4. `UpdateAssignment` UPDATE-Statement: analog — ersetze `actual_break_start=?, actual_break_end=?,` durch `actual_break_minutes=?,` und entsprechend in der Args-Liste
</action>

<acceptance_criteria>
- `assignmentSelect` enthält `COALESCE(a.actual_break_minutes,0)` statt break_start/end
- `scanAssignments` scannt `&a.ActualBreakMinutes` (ein Feld, kein Paar)
- `CreateAssignment` INSERT enthält `actual_break_minutes` mit Wert `a.ActualBreakMinutes`
- `UpdateAssignment` SET enthält `actual_break_minutes=?` mit Wert `a.ActualBreakMinutes`
- `go build ./...` kompiliert ohne Fehler
</acceptance_criteria>

---

### Task 4: Handler-Validierung anpassen

<read_first>
- `internal/api/handlers/validate.go` — aktuelle Validierungslogik für Assignment, alle Felder
- `internal/validate/validate.go` — `IntRange` Funktion (bereits vorhanden)
</read_first>

<action>
In `internal/api/handlers/validate.go`:

1. Entferne die beiden Einträge für `actual_break_start` und `actual_break_end` aus der Zeit-Validierungsschleife (die Zeilen mit `{a.ActualBreakStart, "actual_break_start"}` und `{a.ActualBreakEnd, "actual_break_end"}`)

2. Füge stattdessen hinzu:
   `if err := validate.IntRange(a.ActualBreakMinutes, "actual_break_minutes", 0, 600); err != nil { return err }`
</action>

<acceptance_criteria>
- `validate.go` enthält `validate.IntRange(a.ActualBreakMinutes, "actual_break_minutes", 0, 600)`
- Keine Referenzen mehr auf `ActualBreakStart` oder `ActualBreakEnd` in `validate.go`
- `go build ./...` kompiliert ohne Fehler
</acceptance_criteria>

---

### Task 5: PDF-Export anpassen

<read_first>
- `internal/pdf/worktime.go` — vollständig lesen, insbesondere `breakDuration` oder analoge Funktion und wo Break-Zeiten verwendet werden (ca. Zeile 225, 290–291)
</read_first>

<action>
In `internal/pdf/worktime.go`:

1. Finde die Funktion/Logik die bisher `a.ActualBreakStart + "–" + a.ActualBreakEnd` zurückgibt (ca. Zeile 225–228) und `parseHM(a.ActualBreakStart/End)` aufruft (ca. Zeile 290–291)

2. Ersetze durch:
   - Anzeige: wenn `a.ActualBreakMinutes > 0`, gib `fmt.Sprintf("%d min", a.ActualBreakMinutes)` zurück, sonst `"–"`
   - Dauer-Berechnung für Netto-Zeit: verwende `a.ActualBreakMinutes` direkt (in Minuten) statt break_end − break_start

3. Entferne alle `parseHM`-Aufrufe für Break-Felder (falls keine anderen Caller)
</action>

<acceptance_criteria>
- PDF-Pause wird als "30 min" (oder "–" wenn 0) angezeigt, nicht als Zeitbereich
- Netto-Arbeitszeit im PDF rechnet korrekt: `(end − start) − actual_break_minutes`
- Keine Referenzen auf `ActualBreakStart` oder `ActualBreakEnd` in `worktime.go`
- `go build ./...` kompiliert ohne Fehler
</acceptance_criteria>

---

### Task 6: Frontend — time.js anpassen

<read_first>
- `frontend/src/utils/time.js` — vollständig lesen (alle Funktionen)
</read_first>

<action>
In `frontend/src/utils/time.js`:

1. `breakMinutes(breakStart, breakEnd)` → umbenennen zu `breakMinutes(breakMin)` und Körper ersetzen durch: `return breakMin || 0`

2. `netWorkMinutes(start, breakStart, breakEnd, end)` → Signatur zu `netWorkMinutes(start, breakMin, end)` und Körper ersetzen durch:
   ```js
   if (!start || !end) return 0
   return Math.max(0, diffMinutes(start, end) - (breakMin || 0))
   ```

3. `grossWorkMinutes` bleibt unverändert. `legalMinBreakMinutes`, `requiredBreakMinutes`, `formatHm`, `formatHours`, `diffMinutes` bleiben unverändert.
</action>

<acceptance_criteria>
- `breakMinutes(breakMin)` gibt `breakMin || 0` zurück (ein Parameter)
- `netWorkMinutes(start, breakMin, end)` gibt `diffMinutes(start,end) - breakMin` zurück (drei Parameter)
- `grossWorkMinutes` ist unverändert
- Datei exportiert alle bisherigen Funktionsnamen (kein Breaking Change für andere Imports)
</acceptance_criteria>

---

### Task 7: Frontend — AssignmentForm.vue anpassen

<read_first>
- `frontend/src/components/AssignmentForm.vue` — vollständig lesen (alle Felder, Computed Properties, Template)
- `frontend/src/components/TimeSelect.vue` — zur Referenz (wird entfernt, nicht mehr gebraucht für Pause)
</read_first>

<action>
In `frontend/src/components/AssignmentForm.vue`:

1. **Formular-State**: ersetze `actual_break_start: '', actual_break_end: ''` durch `actual_break_minutes: 0`

2. **Template**: ersetze die zwei `<TimeSelect v-model="form.actual_break_start" />` und `<TimeSelect v-model="form.actual_break_end" />` Blöcke durch Quick-Pick Buttons:
   ```html
   <div class="flex gap-2 flex-wrap">
     <button v-for="min in [0, 15, 30, 45, 60]" :key="min"
       type="button"
       @click="form.actual_break_minutes = min"
       :class="form.actual_break_minutes === min
         ? 'bg-blue-600 text-white'
         : 'bg-gray-100 text-gray-700 hover:bg-gray-200'"
       class="px-3 py-1 rounded text-sm font-medium">
       {{ min === 0 ? 'Keine' : min + ' min' }}
     </button>
   </div>
   ```

3. **Computed Properties**: ersetze alle Aufrufe von `breakMinutes(form.actual_break_start, form.actual_break_end)` durch `breakMinutes(form.actual_break_minutes)`, und `netWorkMinutes(start, form.actual_break_start, form.actual_break_end, end)` durch `netWorkMinutes(start, form.actual_break_minutes, end)`

4. **Provider-Wechsel**: wenn Provider wechselt (watch auf `currentProvider`), setze `form.actual_break_minutes = currentProvider.value?.min_break_minutes || 0` statt bisherigem Reset auf ''

5. **Initialisierung beim Laden**: wenn Assignment geladen wird, setze `actual_break_minutes: a.actual_break_minutes || 0` statt `actual_break_start/end`

6. **Import**: entferne `breakMinutes, netWorkMinutes` Import-Parameter (werden neu importiert mit neuer Signatur) — überprüfe alle Imports aus `../utils/time`
</action>

<acceptance_criteria>
- Template zeigt 5 Buttons (Keine/15/30/45/60 min), kein TimeSelect für Pause
- Aktiver Button (=`form.actual_break_minutes`) hat `bg-blue-600 text-white` Klasse
- Provider-Default füllt `actual_break_minutes` beim Laden und Wechsel vor
- `netWorkMinutes` und `breakMinutes` werden mit neuen Signaturen aufgerufen
- Kein Verweis auf `actual_break_start` oder `actual_break_end` im Script oder Template
</acceptance_criteria>

---

### Task 8: Frontend — WorktimeTable, HistoryView, WorktimeView, WorktimeChart anpassen

<read_first>
- `frontend/src/components/WorktimeTable.vue` — Break-Spalte und Berechnungen (ca. Zeile 102–122)
- `frontend/src/views/HistoryView.vue` — breakMin, netMin Computed (ca. Zeile 138–145)
- `frontend/src/views/WorktimeView.vue` — netWorkMinutes, breakMinutes Aufrufe (ca. Zeile 186–192)
- `frontend/src/components/WorktimeChart.vue` — netWorkMinutes Aufruf (ca. Zeile 121)
</read_first>

<action>
In allen vier Dateien:

**WorktimeTable.vue:**
- `breakMin(a)`: ersetze `breakMinutes(a.actual_break_start, a.actual_break_end)` durch `breakMinutes(a.actual_break_minutes)`
- `netMin(a)`: ersetze `netWorkMinutes(a.actual_start_time, a.actual_break_start, a.actual_break_end, a.actual_end_time)` durch `netWorkMinutes(a.actual_start_time, a.actual_break_minutes, a.actual_end_time)`
- Template-Anzeige der Pause-Spalte: zeige `formatHm(breakMin(a))` statt Zeitbereich. Entferne die `end`-Berechnung die `actual_break_start/end` referenziert (ca. Zeile 102–107)

**HistoryView.vue:**
- analog: `breakMin(a)` und `netMin(a)` mit neuen Signaturen
- Compliance-Check: `breakMinutes(a.actual_break_minutes) < requiredBreakMinutes(...)` → `a.actual_break_minutes < requiredBreakMinutes(...)`

**WorktimeView.vue:**
- `net += netWorkMinutes(a.actual_start_time, a.actual_break_minutes, a.actual_end_time)`
- `brk += a.actual_break_minutes` (statt breakMinutes Aufruf)
- Compliance-Check: `a.actual_break_minutes < requiredBreakMinutes(...)`

**WorktimeChart.vue:**
- `netWorkMinutes(a.actual_start_time, a.actual_break_minutes, a.actual_end_time)`
</action>

<acceptance_criteria>
- Keine Datei referenziert `actual_break_start` oder `actual_break_end` mehr
- `breakMinutes(a.actual_break_minutes)` wird mit einem Argument aufgerufen
- `netWorkMinutes(start, breakMin, end)` wird mit drei Argumenten aufgerufen
- `vite build` kompiliert ohne Fehler
- WorktimeTable zeigt Pausendauer als "0:30" (formatHm) oder "–" wenn 0
</acceptance_criteria>

---

## Verification

```yaml
verification:
  commands:
    - go build ./...
    - cd frontend && npm run build
  checks:
    - "assignments-Tabelle hat actual_break_minutes Spalte (SQLite PRAGMA table_info)"
    - "Bestehende break_start/end-Daten wurden in Minuten konvertiert"
    - "AssignmentForm zeigt Quick-Pick Buttons ohne TimeSelect für Pause"
    - "Netto-Arbeitszeit berechnet sich als (end−start) − break_minutes"
    - "PDF zeigt '30 min' statt Zeitbereich"
    - "Swiss ArG Compliance-Anzeige funktioniert weiterhin"
```
