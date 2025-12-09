package operations

import (
	"testing"
)

func TestGetStatusName(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "created",
			status: StatusCreated,
			want:   "Created",
		},
		{
			name:   "started",
			status: StatusStarted,
			want:   "In Progress",
		},
		{
			name:   "ready",
			status: StatusReady,
			want:   "Ready",
		},
		{
			name:   "delivered",
			status: StatusDelivered,
			want:   "Delivered",
		},
		{
			name:   "cancelled",
			status: StatusCancelled,
			want:   "Cancelled",
		},
		{
			name:   "unknownStatus",
			status: "unknown",
			want:   "Unknown",
		},
		{
			name:   "emptyStatus",
			status: "",
			want:   "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusName(tt.status)
			if got != tt.want {
				t.Errorf("getStatusName(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestStationViewFields(t *testing.T) {
	view := &StationView{
		Station:     "grill",
		StationName: "Grill Station",
		Columns:     make(map[string]*ColumnView),
		ColumnsList: []*ColumnView{},
	}

	if view.Station != "grill" {
		t.Errorf("Station = %q, want %q", view.Station, "grill")
	}
	if view.StationName != "Grill Station" {
		t.Errorf("StationName = %q, want %q", view.StationName, "Grill Station")
	}
	if view.Columns == nil {
		t.Error("Columns should not be nil")
	}
}

func TestColumnViewFields(t *testing.T) {
	view := &ColumnView{
		Status:     StatusCreated,
		StatusName: "Created",
		Tickets:    []*kitchenTicketResource{},
	}

	if view.Status != StatusCreated {
		t.Errorf("Status = %q, want %q", view.Status, StatusCreated)
	}
	if view.StatusName != "Created" {
		t.Errorf("StatusName = %q, want %q", view.StatusName, "Created")
	}
	if view.Tickets == nil {
		t.Error("Tickets should not be nil")
	}
}

func TestStatusConstants(t *testing.T) {
	// Verify status constants are defined correctly
	if StatusCreated != "created" {
		t.Errorf("StatusCreated = %q, want %q", StatusCreated, "created")
	}
	if StatusStarted != "started" {
		t.Errorf("StatusStarted = %q, want %q", StatusStarted, "started")
	}
	if StatusReady != "ready" {
		t.Errorf("StatusReady = %q, want %q", StatusReady, "ready")
	}
	if StatusDelivered != "delivered" {
		t.Errorf("StatusDelivered = %q, want %q", StatusDelivered, "delivered")
	}
	if StatusCancelled != "cancelled" {
		t.Errorf("StatusCancelled = %q, want %q", StatusCancelled, "cancelled")
	}
}
