package operations

import (
	"net/http"
	"sort"
)

// Status IDs (match Kitchen service constants)
const (
	StatusCreated   = "00000000-0000-0000-0000-000000000001"
	StatusAccepted  = "00000000-0000-0000-0000-000000000002"
	StatusStarted   = "00000000-0000-0000-0000-000000000003"
	StatusReady     = "00000000-0000-0000-0000-000000000004"
	StatusDelivered = "00000000-0000-0000-0000-000000000005"
	StatusCancelled = "00000000-0000-0000-0000-000000000010"
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

	// Group tickets by station
	stationMap := make(map[string]*StationView)
	for i := range allTickets {
		ticket := &allTickets[i]
		stationID := ticket.StationID
		if _, exists := stationMap[stationID]; !exists {
			stationMap[stationID] = &StationView{
				StationID:   stationID,
				StationName: ticket.StationName,
				Columns:     make(map[string]*ColumnView),
			}
		}

		station := stationMap[stationID]

		// Create columns for each status
		statusID := ticket.StatusID
		if _, exists := station.Columns[statusID]; !exists {
			station.Columns[statusID] = &ColumnView{
				StatusID:   statusID,
				StatusName: getStatusName(statusID),
				Tickets:    []*kitchenTicketResource{},
			}
		}

		station.Columns[statusID].Tickets = append(station.Columns[statusID].Tickets, ticket)
	}

	// Convert map to slice and sort stations
	// Always show all columns even if empty
	statusOrder := []struct {
		ID   string
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
			if col, exists := station.Columns[status.ID]; exists {
				col.StatusName = status.Name // Override with proper name
				orderedColumns = append(orderedColumns, col)
			} else {
				// Create empty column
				orderedColumns = append(orderedColumns, &ColumnView{
					StatusID:   status.ID,
					StatusName: status.Name,
					Tickets:    []*kitchenTicketResource{},
				})
			}
		}
		station.ColumnsList = orderedColumns
		stations = append(stations, station)
	}

	// If no stations exist, create a default one to show the empty columns
	if len(stations) == 0 {
		defaultStation := &StationView{
			StationID:   "default",
			StationName: "Kitchen",
			Columns:     make(map[string]*ColumnView),
			ColumnsList: make([]*ColumnView, 0, len(statusOrder)),
		}
		for _, status := range statusOrder {
			defaultStation.ColumnsList = append(defaultStation.ColumnsList, &ColumnView{
				StatusID:   status.ID,
				StatusName: status.Name,
				Tickets:    []*kitchenTicketResource{},
			})
		}
		stations = append(stations, defaultStation)
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
	StationID   string
	StationName string
	Columns     map[string]*ColumnView
	ColumnsList []*ColumnView // Ordered list for rendering
}

// ColumnView represents a Kanban column (status)
type ColumnView struct {
	StatusID   string
	StatusName string
	Tickets    []*kitchenTicketResource
}

func getStatusName(statusID string) string {
	switch statusID {
	case StatusCreated:
		return "Created"
	case StatusAccepted:
		return "Accepted"
	case StatusStarted:
		return "In Progress"
	case StatusReady:
		return "Ready"
	case StatusDelivered:
		return "Delivered"
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
