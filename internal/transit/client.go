package transit

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const baseURL = "https://transport.opendata.ch/v1"

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 10 * time.Second}}
}

type ConnectionsResponse struct {
	Connections []Connection `json:"connections"`
}

type Connection struct {
	From     *Checkpoint `json:"from"`
	To       *Checkpoint `json:"to"`
	Duration string      `json:"duration"`
	Sections []Section   `json:"sections"`
}

type Checkpoint struct {
	Station   *Station `json:"station"`
	Departure string   `json:"departure"`
	Arrival   string   `json:"arrival"`
	Platform  string   `json:"platform"`
}

type Station struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Coordinate *Coordinate `json:"coordinate,omitempty"`
}

// Coordinate uses opendata.ch's convention: x=latitude, y=longitude (WGS84).
type Coordinate struct {
	Type string  `json:"type,omitempty"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type Section struct {
	Journey   *Journey    `json:"journey"`
	Walk      *Walk       `json:"walk"`
	Departure *Checkpoint `json:"departure"`
	Arrival   *Checkpoint `json:"arrival"`
}

type Journey struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	CategoryCode string `json:"categoryCode,omitempty"`
	Number       string `json:"number"`
	Operator     string `json:"operator,omitempty"`
	To           string `json:"to,omitempty"`
}

type Walk struct {
	Duration int `json:"duration"`
}

type StationsResponse struct {
	Stations []Station `json:"stations"`
}

// GetConnections fetches the next connections between two stops.
// Set isArrivalTime=true to treat timeStr as desired arrival time (e.g. must arrive by shift start).
func (c *Client) GetConnections(from, to, date, timeStr string, limit int, isArrivalTime bool) (*ConnectionsResponse, error) {
	params := url.Values{
		"from":  {from},
		"to":    {to},
		"date":  {date},
		"time":  {timeStr},
		"limit": {fmt.Sprintf("%d", limit)},
	}
	if isArrivalTime {
		params.Set("isArrivalTime", "1")
	}
	resp, err := c.http.Get(baseURL + "/connections?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transit API returned %d", resp.StatusCode)
	}

	var result ConnectionsResponse
	return &result, json.NewDecoder(resp.Body).Decode(&result)
}

// HaversineMeters returns the great-circle distance between two WGS84 points.
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthR = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthR * math.Asin(math.Sqrt(a))
}

// WalkingMinutes estimates walking time for a straight-line distance.
// Applies a 1.3× detour factor to approximate real streets vs geodesic.
func WalkingMinutes(meters float64) int {
	const metersPerMin = 83.0 // ~5 km/h
	const detour = 1.3
	if meters <= 0 {
		return 0
	}
	return int(math.Round(meters * detour / metersPerMin))
}

// Geocode resolves a free-form address to lat/lng via Nominatim (OSM).
// Per Nominatim usage policy we set a descriptive User-Agent and limit results to 1.
func (c *Client) Geocode(address string) (lat, lng float64, err error) {
	params := url.Values{
		"q":      {address},
		"format": {"json"},
		"limit":  {"1"},
	}
	req, err := http.NewRequest("GET", "https://nominatim.openstreetmap.org/search?"+params.Encode(), nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "kita-springer-manager/1.0 (self-hosted)")
	req.Header.Set("Accept-Language", "de")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("nominatim returned %d", resp.StatusCode)
	}

	var hits []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		return 0, 0, err
	}
	if len(hits) == 0 {
		return 0, 0, fmt.Errorf("address not found")
	}
	lat, _ = strconv.ParseFloat(hits[0].Lat, 64)
	lng, _ = strconv.ParseFloat(hits[0].Lon, 64)
	return lat, lng, nil
}

func (c *Client) SearchStops(query string) (*StationsResponse, error) {
	params := url.Values{
		"query": {query},
		"type":  {"station"},
	}
	resp, err := c.http.Get(baseURL + "/locations?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transit API returned %d", resp.StatusCode)
	}

	var result StationsResponse
	return &result, json.NewDecoder(resp.Body).Decode(&result)
}

// StopsNear returns the nearest transit stations to a coordinate.
// opendata.ch /locations uses x=latitude, y=longitude.
func (c *Client) StopsNear(lat, lng float64) (*StationsResponse, error) {
	params := url.Values{
		"x":    {strconv.FormatFloat(lat, 'f', 6, 64)},
		"y":    {strconv.FormatFloat(lng, 'f', 6, 64)},
		"type": {"station"},
	}
	resp, err := c.http.Get(baseURL + "/locations?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transit API returned %d", resp.StatusCode)
	}
	var result StationsResponse
	return &result, json.NewDecoder(resp.Body).Decode(&result)
}
