package handlers

import (
	"github.com/pak/kita-springer-manager/internal/models"
	"github.com/pak/kita-springer-manager/internal/validate"
)

// Field-length budgets. Conservative ceilings — if a real workflow exceeds
// any of them we'll bump the specific limit rather than removing the cap.
const (
	maxNameLen      = 200
	maxAddressLen   = 300
	maxStopNameLen  = 200
	maxNotesLen     = 5000
	maxLeitungLen   = 200
	maxGroupNameLen = 100
	maxGroupItems   = 50
)

// validStatus / validSource accept only the documented enum values to keep
// the DB and downstream logic (conflict detection, worktime export) sane.
var (
	validStatus = map[string]bool{"": true, "scheduled": true, "free": true}
	validSource = map[string]bool{"": true, "manual": true, "excel": true, "recurring": true}
)

func validateAssignment(a *models.Assignment) error {
	if a.KitaID == "" {
		return errMsg("kita_id ist erforderlich")
	}
	if err := validate.Date(a.Date, "date"); err != nil {
		return err
	}
	for _, p := range []struct{ s, f string }{
		{a.StartTime, "start_time"},
		{a.EndTime, "end_time"},
		{a.ActualStartTime, "actual_start_time"},
		{a.ActualEndTime, "actual_end_time"},
		{a.ActualBreakStart, "actual_break_start"},
		{a.ActualBreakEnd, "actual_break_end"},
	} {
		if err := validate.TimeHM(p.s, p.f); err != nil {
			return err
		}
	}
	if !validStatus[a.Status] {
		return errMsg("status: ungültiger Wert")
	}
	if !validSource[a.Source] {
		return errMsg("source: ungültiger Wert")
	}
	if err := validate.MaxLen(a.GroupName, "group_name", maxGroupNameLen); err != nil {
		return err
	}
	if err := validate.MaxLen(a.Notes, "notes", maxNotesLen); err != nil {
		return err
	}
	return nil
}

func validateKita(k *models.Kita) error {
	if k.Name == "" {
		return errMsg("name ist erforderlich")
	}
	if err := validate.MaxLen(k.Name, "name", maxNameLen); err != nil {
		return err
	}
	if err := validate.MaxLen(k.Address, "address", maxAddressLen); err != nil {
		return err
	}
	if err := validate.MaxLen(k.StopName, "stop_name", maxStopNameLen); err != nil {
		return err
	}
	if err := validate.PhoneOpt(k.Phone, "phone"); err != nil {
		return err
	}
	if err := validate.EmailOpt(k.Email, "email"); err != nil {
		return err
	}
	if err := validate.MaxLen(k.LeitungName, "leitung_name", maxLeitungLen); err != nil {
		return err
	}
	if err := validate.URLHTTPSOpt(k.PhotoURL, "photo_url"); err != nil {
		return err
	}
	if err := validate.MaxLen(k.Notes, "notes", maxNotesLen); err != nil {
		return err
	}
	if err := validate.StringSliceMax(k.Groups, "groups", maxGroupItems, maxGroupNameLen); err != nil {
		return err
	}
	if err := validate.StringSliceMax(k.Stops, "stops", 5, maxStopNameLen); err != nil {
		return err
	}
	return nil
}

func validateProvider(p *models.Provider) error {
	if p.Name == "" {
		return errMsg("name ist erforderlich")
	}
	if err := validate.MaxLen(p.Name, "name", maxNameLen); err != nil {
		return err
	}
	if err := validate.ColorHex(p.ColorHex, "color_hex"); err != nil {
		return err
	}
	if err := validate.MaxLen(p.Notes, "notes", maxNotesLen); err != nil {
		return err
	}
	if err := validate.IntRange(p.MinBreakMinutes, "min_break_minutes", 0, 600); err != nil {
		return err
	}
	return nil
}

func validateRecurring(r *models.RecurringAssignment) error {
	if r.KitaID == "" {
		return errMsg("kita_id ist erforderlich")
	}
	if err := validate.IntRange(r.DayOfWeek, "day_of_week", 0, 6); err != nil {
		return err
	}
	if err := validate.TimeHM(r.StartTime, "start_time"); err != nil {
		return err
	}
	if err := validate.TimeHM(r.EndTime, "end_time"); err != nil {
		return err
	}
	if err := validate.DateRange(r.ValidFrom, r.ValidUntil); err != nil {
		return err
	}
	if err := validate.MaxLen(r.GroupName, "group_name", maxGroupNameLen); err != nil {
		return err
	}
	if err := validate.MaxLen(r.Notes, "notes", maxNotesLen); err != nil {
		return err
	}
	return nil
}

func validateClosure(c *models.Closure) error {
	if err := validate.Date(c.Date, "date"); err != nil {
		return err
	}
	switch c.Type {
	case models.ClosureSpringerin, models.ClosureProvider, models.ClosureKita, models.ClosureHoliday:
	default:
		return errMsg("type: ungültiger Wert")
	}
	if err := validate.MaxLen(c.Note, "note", 500); err != nil {
		return err
	}
	return nil
}

func validateSettings(s *models.Settings) error {
	if err := validate.MaxLen(s.HomeAddress, "home_address", maxAddressLen); err != nil {
		return err
	}
	if err := validate.MaxLen(s.HomeStop, "home_stop", maxStopNameLen); err != nil {
		return err
	}
	if err := validate.MaxLen(s.UserName, "user_name", maxNameLen); err != nil {
		return err
	}
	if err := validate.Coord(s.HomeLat, s.HomeLng); err != nil {
		return err
	}
	// Canton is checked against the holiday list elsewhere.
	return nil
}

// errMsg is a tiny shim so handlers can emit short messages without having
// to import errors.New every time.
type validationErr string

func (e validationErr) Error() string { return string(e) }

func errMsg(s string) error { return validationErr(s) }
