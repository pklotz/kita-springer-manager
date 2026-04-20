package models

import "time"

const (
	ClosureHoliday    = "holiday"
	ClosureSpringerin = "springerin"
	ClosureProvider   = "provider"
	ClosureKita       = "kita"
)

type Closure struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	ReferenceID string    `json:"reference_id,omitempty"`
	Date        string    `json:"date"`
	Note        string    `json:"note,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
