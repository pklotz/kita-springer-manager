package models

type Settings struct {
	HomeAddress  string       `json:"home_address"`
	HomeStop     string       `json:"home_stop"`
	TransitPrefs TransitPrefs `json:"transit_prefs"`
}

type TransitPrefs struct {
	ExcludeTypes []string `json:"exclude_types"` // e.g. "ir", "ic"
	WalkingSpeed string   `json:"walking_speed"` // slow, normal, fast
}
