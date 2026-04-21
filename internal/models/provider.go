package models

import "time"

type Provider struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	ColorHex    string      `json:"color_hex"`
	Notes       string      `json:"notes,omitempty"`
	ExcelConfig ExcelConfig `json:"excel_config"`
	CreatedAt   time.Time   `json:"created_at"`
}

// ExcelConfig describes how to parse a provider's Excel schedule.
type ExcelConfig struct {
	NameCol     string            `json:"name_col"`      // column with names, e.g. "A"
	HeaderRow   int               `json:"header_row"`    // row with weekday labels, e.g. 2
	KitaRow     int               `json:"kita_row"`      // row with group/kita abbreviation, e.g. 3
	FirstDayCol string            `json:"first_day_col"` // first day start column, e.g. "B"
	ColsPerDay  int               `json:"cols_per_day"`  // column pairs per day, e.g. 2
	DaysPerWeek int               `json:"days_per_week"` // e.g. 5 (Mon-Fri)
	KitaMapping map[string]string `json:"kita_mapping"`  // abbrev → kita_id
}

type RecurringAssignment struct {
	ID         string    `json:"id"`
	KitaID     string    `json:"kita_id"`
	Kita       *Kita     `json:"kita,omitempty"`
	ProviderID string    `json:"provider_id"`
	GroupName  string    `json:"group_name"`
	DayOfWeek  int       `json:"day_of_week"` // 0=Monday … 6=Sunday
	StartTime  string    `json:"start_time"`
	EndTime    string    `json:"end_time"`
	ValidFrom  string    `json:"valid_from"`
	ValidUntil string    `json:"valid_until"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
