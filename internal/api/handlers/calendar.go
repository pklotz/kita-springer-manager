package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pak/kita-springer-manager/internal/store"
)

func (h *Handler) GetCalendar(w http.ResponseWriter, r *http.Request) {
	assignments, err := store.ListAssignments(h.db, "", "")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//KitaSpringer//Einsatzkalender//DE\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString("X-WR-CALNAME:Kita Einsätze\r\n")
	sb.WriteString("X-WR-TIMEZONE:Europe/Zurich\r\n")

	dtstamp := time.Now().UTC().Format("20060102T150405Z")

	for _, a := range assignments {
		var dtstart, dtend string
		if a.StartTime != "" {
			dtstart = icalDT("DTSTART", a.Date, a.StartTime)
			endTime := a.EndTime
			if endTime == "" {
				endTime = a.StartTime
			}
			dtend = icalDT("DTEND", a.Date, endTime)
		} else {
			d := strings.ReplaceAll(a.Date, "-", "")
			dtstart = "DTSTART;VALUE=DATE:" + d
			dtend = "DTEND;VALUE=DATE:" + d
		}

		sb.WriteString("BEGIN:VEVENT\r\n")
		fmt.Fprintf(&sb, "UID:%s@kita-springer\r\n", a.ID)
		fmt.Fprintf(&sb, "SUMMARY:%s\r\n", a.Kita.Name)
		fmt.Fprintf(&sb, "%s\r\n", dtstart)
		fmt.Fprintf(&sb, "%s\r\n", dtend)
		if a.Kita.Address != "" {
			fmt.Fprintf(&sb, "LOCATION:%s\r\n", a.Kita.Address)
		}
		var descParts []string
		if a.GroupName != "" {
			descParts = append(descParts, "Gruppe: "+a.GroupName)
		}
		if a.Notes != "" {
			descParts = append(descParts, a.Notes)
		}
		if len(descParts) > 0 {
			fmt.Fprintf(&sb, "DESCRIPTION:%s\r\n", strings.Join(descParts, `\n`))
		}
		fmt.Fprintf(&sb, "DTSTAMP:%s\r\n", dtstamp)
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=kita-einsaetze.ics")
	fmt.Fprint(w, sb.String())
}

func icalDT(prop, date, t string) string {
	dateStr := strings.ReplaceAll(date, "-", "")
	timeStr := strings.ReplaceAll(t, ":", "") + "00"
	return prop + ";TZID=Europe/Zurich:" + dateStr + "T" + timeStr
}
