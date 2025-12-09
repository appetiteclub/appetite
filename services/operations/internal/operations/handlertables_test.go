package operations

import (
	"testing"
	"time"
)

func TestHumanizeStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "emptyString",
			status: "",
			want:   "Unknown",
		},
		{
			name:   "whitespaceOnly",
			status: "   ",
			want:   "Unknown",
		},
		{
			name:   "availableStatus",
			status: "available",
			want:   "Available",
		},
		{
			name:   "openStatus",
			status: "open",
			want:   "Open",
		},
		{
			name:   "occupiedStatus",
			status: "occupied",
			want:   "Occupied",
		},
		{
			name:   "reservedStatus",
			status: "reserved",
			want:   "Reserved",
		},
		{
			name:   "cleaningStatus",
			status: "cleaning",
			want:   "Cleaning",
		},
		{
			name:   "clearingStatus",
			status: "clearing",
			want:   "Clearing",
		},
		{
			name:   "outOfServiceStatus",
			status: "out_of_service",
			want:   "Out of Service",
		},
		{
			name:   "uppercaseAvailable",
			status: "AVAILABLE",
			want:   "Available",
		},
		{
			name:   "mixedCaseOpen",
			status: "OpEn",
			want:   "Open",
		},
		{
			name:   "unknownStatusFallback",
			status: "custom_status",
			want:   "Custom_status",
		},
		{
			name:   "singleChar",
			status: "a",
			want:   "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanizeStatus(tt.status)
			if got != tt.want {
				t.Errorf("humanizeStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestFormatBill(t *testing.T) {
	tests := []struct {
		name string
		bill *tableBillResource
		want string
	}{
		{
			name: "nilBill",
			bill: nil,
			want: "-",
		},
		{
			name: "zeroBill",
			bill: &tableBillResource{Total: 0},
			want: "$0.00",
		},
		{
			name: "positiveBill",
			bill: &tableBillResource{Total: 125.50},
			want: "$125.50",
		},
		{
			name: "largeBill",
			bill: &tableBillResource{Total: 9999.99},
			want: "$9999.99",
		},
		{
			name: "smallBill",
			bill: &tableBillResource{Total: 0.01},
			want: "$0.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBill(tt.bill)
			if got != tt.want {
				t.Errorf("formatBill() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRelativeTimeSince(t *testing.T) {
	tests := []struct {
		name string
		ts   time.Time
		want string
	}{
		{
			name: "zeroTime",
			ts:   time.Time{},
			want: "-",
		},
		{
			name: "justNow",
			ts:   time.Now().Add(-30 * time.Second),
			want: "just now",
		},
		{
			name: "fiveMinutesAgo",
			ts:   time.Now().Add(-5 * time.Minute),
			want: "5m ago",
		},
		{
			name: "thirtyMinutesAgo",
			ts:   time.Now().Add(-30 * time.Minute),
			want: "30m ago",
		},
		{
			name: "oneHourAgo",
			ts:   time.Now().Add(-1 * time.Hour),
			want: "1h ago",
		},
		{
			name: "threeHoursAgo",
			ts:   time.Now().Add(-3 * time.Hour),
			want: "3h ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTimeSince(tt.ts)
			if got != tt.want {
				t.Errorf("relativeTimeSince() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRelativeTimeSinceOldDate(t *testing.T) {
	// Test for dates more than 24 hours ago (returns formatted date)
	ts := time.Now().Add(-48 * time.Hour)
	got := relativeTimeSince(ts)
	expected := ts.Format("02 Jan 15:04")
	if got != expected {
		t.Errorf("relativeTimeSince() = %q, want %q", got, expected)
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "emptyString",
			value: "",
			want:  "",
		},
		{
			name:  "shortID",
			value: "abc123",
			want:  "abc123",
		},
		{
			name:  "exactlyEightChars",
			value: "12345678",
			want:  "12345678",
		},
		{
			name:  "longID",
			value: "123456789abcdef",
			want:  "12345678...",
		},
		{
			name:  "uuid",
			value: "550e8400-e29b-41d4-a716-446655440000",
			want:  "550e8400...",
		},
		{
			name:  "singleChar",
			value: "a",
			want:  "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateID(tt.value)
			if got != tt.want {
				t.Errorf("truncateID(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestShortOrderID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "emptyString",
			value: "",
			want:  "ORDER",
		},
		{
			name:  "whitespaceOnly",
			value: "   ",
			want:  "ORDER",
		},
		{
			name:  "simpleID",
			value: "abc123",
			want:  "ABC123",
		},
		{
			name:  "hyphenatedID",
			value: "order-12345",
			want:  "ORDER",
		},
		{
			name:  "multipleHyphens",
			value: "prefix-middle-suffix",
			want:  "PREFIX",
		},
		{
			name:  "onlyHyphens",
			value: "---",
			want:  "ORDER",
		},
		{
			name:  "longFirstPart",
			value: "verylongprefix-123",
			want:  "VERYLONGPREFIX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortOrderID(tt.value)
			if got != tt.want {
				t.Errorf("shortOrderID(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestRoutingLabel(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "barCategory",
			value: "bar",
			want:  "Bar station",
		},
		{
			name:  "barUppercase",
			value: "BAR",
			want:  "Bar station",
		},
		{
			name:  "directCategory",
			value: "direct",
			want:  "Direct to bill",
		},
		{
			name:  "directMixedCase",
			value: "Direct",
			want:  "Direct to bill",
		},
		{
			name:  "kitchenCategory",
			value: "kitchen",
			want:  "Kitchen",
		},
		{
			name:  "emptyString",
			value: "",
			want:  "Kitchen",
		},
		{
			name:  "unknownCategory",
			value: "dessert",
			want:  "Kitchen",
		},
		{
			name:  "whitespaceCategory",
			value: "  bar  ",
			want:  "Bar station",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := routingLabel(tt.value)
			if got != tt.want {
				t.Errorf("routingLabel(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestTableResourceFields(t *testing.T) {
	assignedTo := "user-123"
	tr := tableResource{
		ID:          "table-1",
		Number:      "5",
		Status:      "available",
		GuestCount:  4,
		AssignedTo:  &assignedTo,
		CurrentBill: &tableBillResource{Total: 50.00},
		UpdatedAt:   time.Now(),
		CreatedAt:   time.Now(),
	}

	if tr.ID != "table-1" {
		t.Errorf("ID = %q, want %q", tr.ID, "table-1")
	}
	if tr.Number != "5" {
		t.Errorf("Number = %q, want %q", tr.Number, "5")
	}
	if tr.Status != "available" {
		t.Errorf("Status = %q, want %q", tr.Status, "available")
	}
	if tr.GuestCount != 4 {
		t.Errorf("GuestCount = %d, want %d", tr.GuestCount, 4)
	}
	if *tr.AssignedTo != "user-123" {
		t.Errorf("AssignedTo = %q, want %q", *tr.AssignedTo, "user-123")
	}
	if tr.CurrentBill.Total != 50.00 {
		t.Errorf("CurrentBill.Total = %v, want %v", tr.CurrentBill.Total, 50.00)
	}
}

func TestTableBillResourceFields(t *testing.T) {
	br := tableBillResource{
		Total: 100.50,
	}

	if br.Total != 100.50 {
		t.Errorf("Total = %v, want %v", br.Total, 100.50)
	}
}
