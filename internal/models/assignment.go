package models

import "time"

const (
	StatusScheduled = "scheduled"
	StatusFree      = "free"

	SourceManual    = "manual"
	SourceExcel     = "excel"
	SourceRecurring = "recurring"
)

type Assignment struct {
	ID              string    `json:"id"`
	KitaID          string    `json:"kita_id,omitempty"`
	Kita            *Kita     `json:"kita,omitempty"`
	ProviderID      string    `json:"provider_id,omitempty"`
	Provider        *Provider `json:"provider,omitempty"`
	GroupName       string    `json:"group_name,omitempty"`
	Date            string    `json:"date"`
	StartTime       string    `json:"start_time"`
	EndTime         string    `json:"end_time"`
	ActualStartTime  string    `json:"actual_start_time"`
	ActualBreakStart string    `json:"actual_break_start"`
	ActualBreakEnd   string    `json:"actual_break_end"`
	ActualEndTime    string    `json:"actual_end_time"`
	Status          string    `json:"status"` // scheduled | free
	Source          string    `json:"source"` // manual | excel | recurring
	ImportHash      string    `json:"import_hash,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}
