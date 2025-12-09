package order

import (
	"context"
	"testing"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/google/uuid"
)

func TestApplyDemoSeedsNilDB(t *testing.T) {
	repos := Repos{
		OrderRepo:      NewMockOrderRepo(),
		OrderItemRepo:  NewMockOrderItemRepo(),
		OrderGroupRepo: NewMockOrderGroupRepo(),
	}

	err := ApplyDemoSeeds(context.Background(), repos, nil, aqm.NewNoopLogger())
	if err == nil {
		t.Error("ApplyDemoSeeds() with nil db should return error")
	}

	expectedMsg := "database is required for demo seeding"
	if err.Error() != expectedMsg {
		t.Errorf("ApplyDemoSeeds() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestDemoSeedingFuncNilDB(t *testing.T) {
	repos := Repos{
		OrderRepo:      NewMockOrderRepo(),
		OrderItemRepo:  NewMockOrderItemRepo(),
		OrderGroupRepo: NewMockOrderGroupRepo(),
	}

	fn := DemoSeedingFunc(context.Background(), repos, nil, nil)
	if fn == nil {
		t.Fatal("DemoSeedingFunc() returned nil function")
	}

	// The function should return nil (the actual error happens in background goroutine)
	err := fn(context.Background())
	if err != nil {
		t.Errorf("DemoSeedingFunc() returned function should not return error, got: %v", err)
	}
}

func TestDemoSeedingFuncWithLogger(t *testing.T) {
	repos := Repos{
		OrderRepo:      NewMockOrderRepo(),
		OrderItemRepo:  NewMockOrderItemRepo(),
		OrderGroupRepo: NewMockOrderGroupRepo(),
	}

	fn := DemoSeedingFunc(context.Background(), repos, nil, aqm.NewNoopLogger())
	if fn == nil {
		t.Fatal("DemoSeedingFunc() returned nil function")
	}

	err := fn(context.Background())
	if err != nil {
		t.Errorf("DemoSeedingFunc() returned function should not return error, got: %v", err)
	}
}

func TestCreateItem(t *testing.T) {
	orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440400")
	groupID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440401")
	station := "grill"
	now := time.Now()

	tests := []struct {
		name               string
		orderID            uuid.UUID
		groupID            *uuid.UUID
		dishName           string
		category           string
		quantity           int
		price              float64
		status             string
		notes              string
		station            *string
		requiresProduction bool
	}{
		{
			name:               "fullItem",
			orderID:            orderID,
			groupID:            &groupID,
			dishName:           "Test Steak",
			category:           "Main",
			quantity:           2,
			price:              29.99,
			status:             "pending",
			notes:              "Medium rare",
			station:            &station,
			requiresProduction: true,
		},
		{
			name:               "itemWithoutGroup",
			orderID:            orderID,
			groupID:            nil,
			dishName:           "Soda",
			category:           "Beverage",
			quantity:           1,
			price:              3.50,
			status:             "pending",
			notes:              "",
			station:            nil,
			requiresProduction: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := createItem(tt.orderID, tt.groupID, tt.dishName, tt.category, tt.quantity, tt.price, tt.status, tt.notes, tt.station, tt.requiresProduction, now, now)

			if item == nil {
				t.Fatal("createItem() returned nil")
			}
			if item.OrderID != tt.orderID {
				t.Errorf("createItem() OrderID = %v, want %v", item.OrderID, tt.orderID)
			}
			if tt.groupID != nil && (item.GroupID == nil || *item.GroupID != *tt.groupID) {
				t.Errorf("createItem() GroupID = %v, want %v", item.GroupID, tt.groupID)
			}
			if tt.groupID == nil && item.GroupID != nil {
				t.Errorf("createItem() GroupID should be nil, got %v", item.GroupID)
			}
			if item.DishName != tt.dishName {
				t.Errorf("createItem() DishName = %q, want %q", item.DishName, tt.dishName)
			}
			if item.Category != tt.category {
				t.Errorf("createItem() Category = %q, want %q", item.Category, tt.category)
			}
			if item.Quantity != tt.quantity {
				t.Errorf("createItem() Quantity = %d, want %d", item.Quantity, tt.quantity)
			}
			if item.Price != tt.price {
				t.Errorf("createItem() Price = %f, want %f", item.Price, tt.price)
			}
			if item.Status != tt.status {
				t.Errorf("createItem() Status = %q, want %q", item.Status, tt.status)
			}
			if item.RequiresProduction != tt.requiresProduction {
				t.Errorf("createItem() RequiresProduction = %v, want %v", item.RequiresProduction, tt.requiresProduction)
			}
			if item.CreatedBy != "seed:demo" {
				t.Errorf("createItem() CreatedBy = %q, want %q", item.CreatedBy, "seed:demo")
			}
		})
	}
}

func TestBuildDemoOrderSeeds(t *testing.T) {
	repos := Repos{
		OrderRepo:      NewMockOrderRepo(),
		OrderItemRepo:  NewMockOrderItemRepo(),
		OrderGroupRepo: NewMockOrderGroupRepo(),
	}

	seeds := buildDemoOrderSeeds(repos, nil, aqm.NewNoopLogger())
	if len(seeds) == 0 {
		t.Error("buildDemoOrderSeeds() should return at least one seed")
	}

	// Verify seed has correct ID
	if seeds[0].ID != "2024-11-23_demo_orders_v1" {
		t.Errorf("buildDemoOrderSeeds() seed ID = %q, want %q", seeds[0].ID, "2024-11-23_demo_orders_v1")
	}
}
