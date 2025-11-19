package operations

import (
	"testing"
	"time"
)

func TestBuildOrderCardTotalsAndEvents(t *testing.T) {
	handler := &Handler{}
	order := orderResource{ID: "order-1", TableID: "table-1", Status: "pending", CreatedAt: time.Now().Add(-2 * time.Hour), UpdatedAt: time.Now().Add(-time.Hour)}
	groups := []orderGroupResource{{ID: "group-main", Name: "Main"}}
	groupID := "group-main"
	items := []orderItemResource{
		{ID: "item-1", DishName: "Pasta", Quantity: 2, Price: 10, Status: "pending", CreatedAt: time.Now().Add(-90 * time.Minute), UpdatedAt: time.Now().Add(-80 * time.Minute), GroupID: &groupID},
		{ID: "item-2", DishName: "Water", Quantity: 1, Price: 5, Status: "pending", CreatedAt: time.Now().Add(-70 * time.Minute), UpdatedAt: time.Now().Add(-60 * time.Minute)},
	}

	card := handler.buildOrderCard(order, nil, items, groups, nil)

	if len(card.GroupTotals) != 2 {
		t.Fatalf("expected 2 group totals, got %d", len(card.GroupTotals))
	}

	if card.GroupTotals[0].Name == card.GroupTotals[1].Name {
		t.Fatalf("expected different group names in totals")
	}

	if len(card.Events) == 0 {
		t.Fatalf("expected events to be generated")
	}
}

func TestOrderItemFormEnsureGroupSelection(t *testing.T) {
	form := orderItemFormModal{
		Groups: []orderGroupResource{
			{ID: "g1", Name: "Main", IsDefault: true},
			{ID: "g2", Name: "Bar"},
		},
	}
	form.ensureGroupSelection()
	if form.GroupID != "g1" {
		t.Fatalf("expected default group g1, got %s", form.GroupID)
	}

	form2 := orderItemFormModal{
		Groups: []orderGroupResource{
			{ID: "g3", Name: "Alt"},
		},
	}
	form2.ensureGroupSelection()
	if form2.GroupID != "g3" {
		t.Fatalf("expected fallback to first group, got %s", form2.GroupID)
	}
}
