package operations

import (
	"net/http"
	"sort"

	"github.com/appetiteclub/appetite/pkg/enums/station"
)

// Status codes (match Kitchen service status enum)
const (
	StatusCreated   = "created"
	StatusStarted   = "started"
	StatusReady     = "ready"
	StatusDelivered = "delivered"
	StatusCancelled = "cancelled"
)

// KitchenKanban displays the kitchen dashboard with tabs by station and columns by status.
func (h *Handler) KitchenKanban(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.KitchenKanban")
	defer finish()

	log := h.log()
	ctx := r.Context()

	// Get all tickets from Kitchen service via HTTP
	var allTickets []kitchenTicketResource
	if h.kitchenData == nil {
		log.Info("Kitchen service not configured, showing empty dashboard")
		// Empty tickets - will show empty Kanban board
	} else {
		tickets, err := h.kitchenData.ListTickets(ctx)
		if err != nil {
			log.Errorf("cannot fetch tickets from kitchen: %v", err)
			// Don't fail - just show empty board
			log.Info("Failed to fetch tickets, showing empty dashboard")
		} else {
			allTickets = tickets
		}
	}

	// Pre-create all stations from enum (so they always show, even without tickets)
	stationMap := make(map[string]*StationView)
	for _, s := range station.All {
		stationMap[s.Code()] = &StationView{
			Station:     s.Code(),
			StationName: s.Label(),
			Columns:     make(map[string]*ColumnView),
		}
	}

	// Group tickets by station
	for i := range allTickets {
		ticket := &allTickets[i]
		stationCode := ticket.Station

		// Get station (should always exist from pre-creation above)
		st, exists := stationMap[stationCode]
		if !exists {
			// Fallback for unknown station codes (shouldn't happen with valid data)
			st = &StationView{
				Station:     stationCode,
				StationName: stationCode,
				Columns:     make(map[string]*ColumnView),
			}
			stationMap[stationCode] = st
		}

		// Create columns for each status
		status := ticket.Status
		if _, exists := st.Columns[status]; !exists {
			st.Columns[status] = &ColumnView{
				Status:     status,
				StatusName: getStatusName(status),
				Tickets:    []*kitchenTicketResource{},
			}
		}

		st.Columns[status].Tickets = append(st.Columns[status].Tickets, ticket)
	}

	// Convert map to slice and sort stations
	// Always show all columns even if empty
	statusOrder := []struct {
		Code string
		Name string
	}{
		{StatusCreated, "Received"},
		{StatusStarted, "In Preparation"},
		{StatusReady, "Ready for Delivery"},
		{StatusDelivered, "Delivered"},
		{StatusCancelled, "Rejected"},
	}

	stations := make([]*StationView, 0, len(stationMap))
	for _, station := range stationMap {
		// Create all columns (even empty ones)
		orderedColumns := make([]*ColumnView, 0, len(statusOrder))
		for _, status := range statusOrder {
			if col, exists := station.Columns[status.Code]; exists {
				col.StatusName = status.Name // Override with proper name
				orderedColumns = append(orderedColumns, col)
			} else {
				// Create empty column
				orderedColumns = append(orderedColumns, &ColumnView{
					Status:     status.Code,
					StatusName: status.Name,
					Tickets:    []*kitchenTicketResource{},
				})
			}
		}
		station.ColumnsList = orderedColumns
		stations = append(stations, station)
	}

	// Sort stations by name
	sort.Slice(stations, func(i, j int) bool {
		return stations[i].StationName < stations[j].StationName
	})

	data := map[string]interface{}{
		"Title":    "Kitchen Dashboard",
		"Template": "kitchen",
		"User":     h.getUserFromSession(r),
		"stations": stations,
	}

	h.renderTemplate(w, "kitchen.html", "base.html", data)
}

// StationView represents a station with its columns
type StationView struct {
	Station     string
	StationName string
	Columns     map[string]*ColumnView
	ColumnsList []*ColumnView // Ordered list for rendering
}

// ColumnView represents a Kanban column (status)
type ColumnView struct {
	Status     string
	StatusName string
	Tickets    []*kitchenTicketResource
}

func getStatusName(status string) string {
	switch status {
	case StatusCreated:
		return "Created"
	case StatusStarted:
		return "In Progress"
	case StatusReady:
		return "Ready"
	case StatusDelivered:
		return "Delivered"
	case StatusCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// ProxyKitchenTicketStatus proxies PATCH /api/kitchen/tickets/:id/status to Kitchen service
func (h *Handler) ProxyKitchenTicketStatus(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.ProxyKitchenTicketStatus")
	defer finish()

	if h.kitchenData == nil {
		http.Error(w, "Kitchen service not configured", http.StatusServiceUnavailable)
		return
	}

	// Forward the request to Kitchen service
	err := h.kitchenData.UpdateTicketStatus(r.Context(), r)
	if err != nil {
		h.log().Errorf("failed to update ticket status: %v", err)
		http.Error(w, "Failed to update ticket status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
