package operations

import (
	"testing"
)

func TestFormatOrderItemStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "pending",
			status: "pending",
			want:   "Pending",
		},
		{
			name:   "preparing",
			status: "preparing",
			want:   "Preparing",
		},
		{
			name:   "ready",
			status: "ready",
			want:   "Ready",
		},
		{
			name:   "delivered",
			status: "delivered",
			want:   "Delivered",
		},
		{
			name:   "cancelled",
			status: "cancelled",
			want:   "Cancelled",
		},
		{
			name:   "unknownStatus",
			status: "custom-status",
			want:   "custom-status",
		},
		{
			name:   "emptyStatus",
			status: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOrderItemStatus(tt.status)
			if got != tt.want {
				t.Errorf("formatOrderItemStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestFormatOrderItemStatusClass(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "pending",
			status: "pending",
			want:   "status-pending",
		},
		{
			name:   "preparing",
			status: "preparing",
			want:   "status-preparing",
		},
		{
			name:   "ready",
			status: "ready",
			want:   "status-ready",
		},
		{
			name:   "delivered",
			status: "delivered",
			want:   "status-delivered",
		},
		{
			name:   "cancelled",
			status: "cancelled",
			want:   "status-cancelled",
		},
		{
			name:   "unknownStatus",
			status: "custom",
			want:   "",
		},
		{
			name:   "emptyStatus",
			status: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatOrderItemStatusClass(tt.status)
			if got != tt.want {
				t.Errorf("formatOrderItemStatusClass(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestNewOrderItemAdapter(t *testing.T) {
	adapter := NewOrderItemAdapter(nil)
	if adapter == nil {
		t.Error("NewOrderItemAdapter() returned nil")
	}
}
