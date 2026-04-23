package models

import "time"

type Kita struct {
	ID          string    `json:"id"`
	ProviderID  string    `json:"provider_id,omitempty"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
	StopName    string    `json:"stop_name"` // primary stop — mirrors Stops[0]
	Stops       []string  `json:"stops"`     // up to 2 nearby public-transport stops
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
	LeitungName string    `json:"leitung_name,omitempty"`
	PhotoURL    string    `json:"photo_url,omitempty"`
	Groups      []string  `json:"groups"`
	Lat         *float64  `json:"lat,omitempty"`
	Lng         *float64  `json:"lng,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
