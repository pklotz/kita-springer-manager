package models

type Settings struct {
	HomeAddress  string       `json:"home_address"`
	HomeStop     string       `json:"home_stop"`
	HomeLat      float64      `json:"home_lat"`
	HomeLng      float64      `json:"home_lng"`
	UserName     string       `json:"user_name"`
	Canton       string       `json:"canton"` // ISO 3166-2:CH code, e.g. "BE"
	TransitPrefs TransitPrefs `json:"transit_prefs"`
}

type TransitPrefs struct {
	ExcludeTypes []string `json:"exclude_types"` // e.g. "ir", "ic"
	WalkingSpeed string   `json:"walking_speed"` // slow, normal, fast
}
