package handlers

import (
	"net/http"
	"strconv"

	"github.com/pak/kita-springer-manager/internal/store"
)

func (h *Handler) GetConnections(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	date := q.Get("date")
	timeStr := q.Get("time")
	isArrival := q.Get("is_arrival") == "1"

	if from == "" || to == "" {
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
		settings, _ := store.GetSettings(h.db)
		from = settings.HomeStop
		to = a.Kita.StopName
		date = a.Date
		if timeStr == "" {
			timeStr = a.StartTime
		}
		// Default to arrival time so the user arrives at the Kita on time
		if q.Get("is_arrival") == "" {
			isArrival = true
		}
	}

	limit := 5
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	result, err := h.transit.GetConnections(from, to, date, timeStr, limit, isArrival)
	if err != nil {
		writeError(w, 502, "transit API error: "+err.Error())
		return
	}
	writeJSON(w, 200, result)
}

func (h *Handler) SearchStops(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, 400, "q parameter required")
		return
	}
	result, err := h.transit.SearchStops(query)
	if err != nil {
		writeError(w, 502, "transit API error: "+err.Error())
		return
	}
	writeJSON(w, 200, result)
}
