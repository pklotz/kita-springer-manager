// Package validate offers small, dependency-free validation primitives for
// the request-decoded models. All errors are German strings safe to surface
// to the client (so we don't need an i18n layer in the API).
package validate

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	dateRe   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRe   = regexp.MustCompile(`^\d{1,2}:\d{2}$`)
	colorRe  = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	// Pragmatic email check: one @, no whitespace, at least one dot in the
	// host part. Matches what users expect; full RFC 5322 is overkill here.
	emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

// Date requires a non-empty YYYY-MM-DD string that parses cleanly.
func Date(s, field string) error {
	if s == "" {
		return fmt.Errorf("%s: Datum fehlt", field)
	}
	if !dateRe.MatchString(s) {
		return fmt.Errorf("%s: Datum muss im Format YYYY-MM-DD sein", field)
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("%s: ungültiges Datum %q", field, s)
	}
	return nil
}

// DateOpt accepts an empty string; otherwise validates as Date.
func DateOpt(s, field string) error {
	if s == "" {
		return nil
	}
	return Date(s, field)
}

// DateRange requires from <= until (lexical comparison works because both
// pass Date validation and use ISO-8601 ordering).
func DateRange(from, until string) error {
	if err := Date(from, "valid_from"); err != nil {
		return err
	}
	if err := Date(until, "valid_until"); err != nil {
		return err
	}
	if from > until {
		return fmt.Errorf("valid_from darf nicht nach valid_until liegen")
	}
	return nil
}

// TimeHM accepts empty (= unset) or HH:MM with the hour 0–23 and minute 0–59.
func TimeHM(s, field string) error {
	if s == "" {
		return nil
	}
	if !timeRe.MatchString(s) {
		return fmt.Errorf("%s: Zeit muss im Format HH:MM sein", field)
	}
	parts := strings.SplitN(s, ":", 2)
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return fmt.Errorf("%s: ungültige Uhrzeit %q", field, s)
	}
	return nil
}

// ColorHex accepts empty or "#RRGGBB" (case-insensitive).
func ColorHex(s, field string) error {
	if s == "" {
		return nil
	}
	if !colorRe.MatchString(s) {
		return fmt.Errorf("%s: Farbe muss im Format #RRGGBB sein", field)
	}
	return nil
}

// EmailOpt accepts empty or a basic-shape email.
func EmailOpt(s, field string) error {
	if s == "" {
		return nil
	}
	if utf8.RuneCountInString(s) > 254 {
		return fmt.Errorf("%s: zu lang", field)
	}
	if !emailRe.MatchString(s) {
		return fmt.Errorf("%s: keine gültige E-Mail-Adresse", field)
	}
	return nil
}

// URLHTTPSOpt accepts empty or an https:// URL with a non-empty host.
// Plain http is rejected — these URLs are rendered as <img src> and we don't
// want to mix-content downgrade or expose tracking pixels over HTTP.
func URLHTTPSOpt(s, field string) error {
	if s == "" {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%s: keine gültige URL", field)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("%s: URL muss mit https:// beginnen", field)
	}
	return nil
}

// MaxLen guards against unbounded user strings by counting runes (so that
// emoji and combined characters don't sneak past byte-length limits).
func MaxLen(s, field string, max int) error {
	if utf8.RuneCountInString(s) > max {
		return fmt.Errorf("%s: zu lang (max. %d Zeichen)", field, max)
	}
	return nil
}

// IntRange checks that v is within [min, max] inclusive.
func IntRange(v int, field string, min, max int) error {
	if v < min || v > max {
		return fmt.Errorf("%s: muss zwischen %d und %d liegen", field, min, max)
	}
	return nil
}

// Coord validates a WGS84 lat/lng pair. The (0,0) sentinel means "not set"
// and passes — geocoding writes real values or leaves both at zero.
func Coord(lat, lng float64) error {
	if lat == 0 && lng == 0 {
		return nil
	}
	if lat < -90 || lat > 90 {
		return fmt.Errorf("home_lat: Breitengrad muss zwischen -90 und 90 liegen")
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("home_lng: Längengrad muss zwischen -180 und 180 liegen")
	}
	return nil
}

// PhoneOpt accepts empty or a permissive phone string: digits, spaces,
// "+", "-", "/", "(", ")", up to 32 characters. Strict E.164 would reject
// many real-world Swiss formats users actually paste in.
func PhoneOpt(s, field string) error {
	if s == "" {
		return nil
	}
	if utf8.RuneCountInString(s) > 32 {
		return fmt.Errorf("%s: zu lang (max. 32 Zeichen)", field)
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r == ' ' || r == '+' || r == '-' || r == '/' || r == '(' || r == ')':
		default:
			return fmt.Errorf("%s: enthält ungültiges Zeichen %q", field, r)
		}
	}
	return nil
}

// StringSliceMax checks that the slice has at most maxItems entries and
// each entry is at most maxLen runes long.
func StringSliceMax(items []string, field string, maxItems, maxLen int) error {
	if len(items) > maxItems {
		return fmt.Errorf("%s: zu viele Einträge (max. %d)", field, maxItems)
	}
	for i, v := range items {
		if utf8.RuneCountInString(v) > maxLen {
			return fmt.Errorf("%s[%d]: zu lang (max. %d Zeichen)", field, i, maxLen)
		}
	}
	return nil
}
