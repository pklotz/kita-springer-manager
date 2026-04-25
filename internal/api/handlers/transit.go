package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/pak/kita-springer-manager/internal/store"
	"github.com/pak/kita-springer-manager/internal/transit"
)

func (h *Handler) GetConnections(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	date := q.Get("date")
	timeStr := q.Get("time")
	isArrival := q.Get("is_arrival") == "1"

	limit := 3
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	// Direct from/to query (no merging).
	if from != "" && to != "" {
		result, err := h.transit.GetConnections(from, to, date, timeStr, limit, isArrival)
		if err != nil {
			upstreamError(w, err, "ÖV-API nicht erreichbar")
			return
		}
		writeJSON(w, 200, map[string]any{
			"connections":                result.Connections,
			"walk_to_first_stop_minutes": h.walkToFirstStop(result.Connections),
		})
		return
	}

	// Assignment-based query: may fan out across multiple Kita stops.
	assignmentID := q.Get("assignment_id")
	if assignmentID == "" {
		writeError(w, 400, "from/to stops or assignment_id required")
		return
	}
	a, err := store.GetAssignment(h.db, assignmentID)
	if err != nil || a == nil {
		writeError(w, 400, "assignment not found")
		return
	}
	if a.Kita == nil {
		writeError(w, 422, "Keine Kita für diesen Einsatz hinterlegt")
		return
	}
	stops := a.Kita.Stops
	if len(stops) == 0 && a.Kita.StopName != "" {
		stops = []string{a.Kita.StopName}
	}
	if len(stops) == 0 {
		writeError(w, 422, "Keine ÖV-Haltestelle für diese Kita hinterlegt")
		return
	}

	settings, _ := store.GetSettings(h.db)
	if from == "" {
		from = settings.HomeStop
	}
	if date == "" {
		date = a.Date
	}
	if timeStr == "" {
		timeStr = a.StartTime
	}
	// Default to arrival time so the user arrives at the Kita on time.
	if q.Get("is_arrival") == "" {
		isArrival = true
	}

	// Per-stop fanout: fetch `limit` connections for each stop, then merge.
	merged := []transit.Connection{}
	for _, stop := range stops {
		res, err := h.transit.GetConnections(from, stop, date, timeStr, limit, isArrival)
		if err != nil {
			continue
		}
		merged = append(merged, res.Connections...)
	}
	if len(merged) == 0 {
		writeError(w, 502, "Keine Verbindungen gefunden")
		return
	}

	// Sort by departure time. Use arrival as a tiebreak so faster connections win.
	sort.SliceStable(merged, func(i, j int) bool {
		di, dj := depTime(merged[i]), depTime(merged[j])
		if di != dj {
			return di < dj
		}
		return arrTime(merged[i]) < arrTime(merged[j])
	})
	merged = dedupeConnections(merged)
	if len(merged) > limit {
		merged = merged[:limit]
	}

	writeJSON(w, 200, map[string]any{
		"connections":                merged,
		"walk_to_first_stop_minutes": h.walkToFirstStop(merged),
	})
}

// walkToFirstStop estimates walking minutes from the configured home to the first
// departure stop of the first connection. Returns 0 if home coords are missing.
func (h *Handler) walkToFirstStop(conns []transit.Connection) int {
	if len(conns) == 0 {
		return 0
	}
	settings, _ := store.GetSettings(h.db)
	if settings == nil || (settings.HomeLat == 0 && settings.HomeLng == 0) {
		return 0
	}
	c := conns[0]
	if c.From == nil || c.From.Station == nil || c.From.Station.Coordinate == nil {
		return 0
	}
	co := c.From.Station.Coordinate
	m := transit.HaversineMeters(settings.HomeLat, settings.HomeLng, co.X, co.Y)
	return transit.WalkingMinutes(m)
}

func depTime(c transit.Connection) string {
	if c.From == nil {
		return ""
	}
	return c.From.Departure
}

func arrTime(c transit.Connection) string {
	if c.To == nil {
		return ""
	}
	return c.To.Arrival
}

// dedupeConnections removes entries with identical (departure, arrival) pairs —
// stops that belong to the same physical station often yield duplicate journeys.
func dedupeConnections(conns []transit.Connection) []transit.Connection {
	seen := map[string]bool{}
	out := conns[:0]
	for _, c := range conns {
		key := depTime(c) + "|" + arrTime(c)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}

func (h *Handler) SearchStops(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, 400, "q parameter required")
		return
	}
	result, err := h.transit.SearchStops(query)
	if err != nil {
		upstreamError(w, err, "ÖV-API nicht erreichbar")
		return
	}
	writeJSON(w, 200, result)
}
