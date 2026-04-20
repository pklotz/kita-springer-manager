package transit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	ID   string `json:"id"`
	Name string `json:"name"`
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
