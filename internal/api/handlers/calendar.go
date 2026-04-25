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
		writeICalProp(&sb, "SUMMARY", a.Kita.Name)
		fmt.Fprintf(&sb, "%s\r\n", dtstart)
		fmt.Fprintf(&sb, "%s\r\n", dtend)
		if a.Kita.Address != "" {
			writeICalProp(&sb, "LOCATION", a.Kita.Address)
		}
		var descParts []string
		if a.GroupName != "" {
			descParts = append(descParts, "Gruppe: "+a.GroupName)
		}
		if a.Notes != "" {
			descParts = append(descParts, a.Notes)
		}
		if len(descParts) > 0 {
			writeICalProp(&sb, "DESCRIPTION", strings.Join(descParts, "\n"))
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

// writeICalProp writes a property line with RFC 5545-compliant escaping and
// 75-octet line folding. Without this, user-supplied notes/addresses
// containing commas, semicolons, backslashes, or newlines could break the
// stream or inject extra calendar properties.
func writeICalProp(sb *strings.Builder, name, value string) {
	line := name + ":" + icalEscape(value)
	foldICalLine(sb, line)
}

// icalEscape escapes the four characters that have meaning in iCal TEXT
// values, per RFC 5545 §3.3.11. Order matters: backslash must come first.
func icalEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\r\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\n`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	return s
}

// foldICalLine writes a content line, folded at 75 octets per RFC 5545 §3.1.
// Continuation lines start with a single space.
func foldICalLine(sb *strings.Builder, line string) {
	const maxOctets = 75
	b := []byte(line)
	for len(b) > maxOctets {
		// Don't split inside a UTF-8 sequence: walk back to the previous
		// rune boundary if maxOctets lands mid-codepoint.
		split := maxOctets
		for split > 0 && b[split]&0xC0 == 0x80 {
			split--
		}
		sb.Write(b[:split])
		sb.WriteString("\r\n ")
		b = b[split:]
	}
	sb.Write(b)
	sb.WriteString("\r\n")
}
