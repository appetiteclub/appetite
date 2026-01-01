package kitchen

import (
	"context"
	"testing"

	"github.com/appetiteclub/appetite/pkg/enums/kitchenstatus"
	"github.com/appetiteclub/apt"
)

func TestApplyDemoSeedsNilDB(t *testing.T) {
	repo := NewMockTicketRepository()
	cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
	logger := apt.NewNoopLogger()

	err := ApplyDemoSeeds(context.Background(), repo, cache, nil, logger)
	if err == nil {
		t.Error("ApplyDemoSeeds() with nil db should return error")
	}
	if err.Error() != "database is required for demo seeding" {
		t.Errorf("ApplyDemoSeeds() error = %v, want 'database is required for demo seeding'", err)
	}
}

func TestMapOrderItemStatusToKitchenStatus(t *testing.T) {
	tests := []struct {
		name        string
		orderStatus string
		wantStatus  string
	}{
		{
			name:        "pendingToCreated",
			orderStatus: "pending",
			wantStatus:  kitchenstatus.Statuses.Created.Code(),
		},
		{
			name:        "preparingToStarted",
			orderStatus: "preparing",
			wantStatus:  kitchenstatus.Statuses.Started.Code(),
		},
		{
			name:        "readyToReady",
			orderStatus: "ready",
			wantStatus:  kitchenstatus.Statuses.Ready.Code(),
		},
		{
			name:        "deliveredToDelivered",
			orderStatus: "delivered",
			wantStatus:  kitchenstatus.Statuses.Delivered.Code(),
		},
		{
			name:        "unknownToCreated",
			orderStatus: "unknown",
			wantStatus:  kitchenstatus.Statuses.Created.Code(),
		},
		{
			name:        "emptyToCreated",
			orderStatus: "",
			wantStatus:  kitchenstatus.Statuses.Created.Code(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapOrderItemStatusToKitchenStatus(tt.orderStatus)
			if got != tt.wantStatus {
				t.Errorf("mapOrderItemStatusToKitchenStatus(%q) = %v, want %v", tt.orderStatus, got, tt.wantStatus)
			}
		})
	}
}

func TestDemoSeedingFunc(t *testing.T) {
	tests := []struct {
		name   string
		logger apt.Logger
	}{
		{
			name:   "withLogger",
			logger: apt.NewNoopLogger(),
		},
		{
			name:   "withNilLogger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTicketRepository()
			cache := NewTicketStateCache(nil, nil, apt.NewNoopLogger())
			ctx := context.Background()

			fn := DemoSeedingFunc(ctx, repo, cache, nil, tt.logger)
			if fn == nil {
				t.Error("DemoSeedingFunc() returned nil")
			}

			// The function should not panic when called
			err := fn(ctx)
			if err != nil {
				t.Errorf("DemoSeedingFunc() returned function that errors: %v", err)
			}
		})
	}
}

func TestBuildDemoKitchenSeeds(t *testing.T) {
	repo := NewMockTicketRepository()
	logger := apt.NewNoopLogger()

	// With nil db, buildDemoKitchenSeeds should return a slice with one seed
	seeds := buildDemoKitchenSeeds(repo, nil, logger)

	if len(seeds) != 1 {
		t.Errorf("buildDemoKitchenSeeds() returned %d seeds, want 1", len(seeds))
	}

	if seeds[0].ID != "2024-11-23_demo_kitchen_tickets_v1" {
		t.Errorf("seed ID = %v, want '2024-11-23_demo_kitchen_tickets_v1'", seeds[0].ID)
	}
}

func TestOrderItemForTicketStruct(t *testing.T) {
	// Verify the struct has expected fields
	item := OrderItemForTicket{}

	// Just ensure the struct can be instantiated and fields exist
	_ = item.ID
	_ = item.OrderID
	_ = item.DishName
	_ = item.Quantity
	_ = item.Status
	_ = item.Notes
	_ = item.ProductionStation
	_ = item.RequiresProduction
	_ = item.CreatedAt
	_ = item.UpdatedAt
}
