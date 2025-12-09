package operations

import (
	"testing"
	"time"
)

func TestSummarizePrep(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]int
		want   string
	}{
		{
			name:   "allZeroCounts",
			counts: map[string]int{},
			want:   "No kitchen items",
		},
		{
			name:   "onlyPending",
			counts: map[string]int{"pending": 3},
			want:   "3 pending",
		},
		{
			name:   "onlyPreparing",
			counts: map[string]int{"preparing": 2},
			want:   "2 preparing",
		},
		{
			name:   "onlyReady",
			counts: map[string]int{"ready": 5},
			want:   "5 ready",
		},
		{
			name:   "pendingAndPreparing",
			counts: map[string]int{"pending": 1, "preparing": 2},
			want:   "1 pending • 2 preparing",
		},
		{
			name:   "allStatuses",
			counts: map[string]int{"pending": 1, "preparing": 2, "ready": 3},
			want:   "1 pending • 2 preparing • 3 ready",
		},
		{
			name:   "pendingAndReady",
			counts: map[string]int{"pending": 4, "ready": 1},
			want:   "4 pending • 1 ready",
		},
		{
			name:   "preparingAndReady",
			counts: map[string]int{"preparing": 2, "ready": 2},
			want:   "2 preparing • 2 ready",
		},
		{
			name:   "nilMap",
			counts: nil,
			want:   "No kitchen items",
		},
		{
			name:   "zeroPendingZeroPreparing",
			counts: map[string]int{"pending": 0, "preparing": 0, "ready": 0},
			want:   "No kitchen items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizePrep(tt.counts)
			if got != tt.want {
				t.Errorf("summarizePrep() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummarizeNonPrep(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  string
	}{
		{
			name:  "zeroCount",
			count: 0,
			want:  "",
		},
		{
			name:  "singleItem",
			count: 1,
			want:  "1 direct-to-check items",
		},
		{
			name:  "multipleItems",
			count: 5,
			want:  "5 direct-to-check items",
		},
		{
			name:  "largeCount",
			count: 100,
			want:  "100 direct-to-check items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeNonPrep(tt.count)
			if got != tt.want {
				t.Errorf("summarizeNonPrep(%d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestParseMoney(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  float64
	}{
		{
			name:  "emptyString",
			value: "",
			want:  0,
		},
		{
			name:  "simpleNumber",
			value: "10.50",
			want:  10.50,
		},
		{
			name:  "dollarPrefix",
			value: "$15.99",
			want:  15.99,
		},
		{
			name:  "integerValue",
			value: "100",
			want:  100,
		},
		{
			name:  "dollarPrefixInteger",
			value: "$50",
			want:  50,
		},
		{
			name:  "invalidValue",
			value: "not-a-number",
			want:  0,
		},
		{
			name:  "zeroValue",
			value: "0",
			want:  0,
		},
		{
			name:  "dollarZero",
			value: "$0.00",
			want:  0,
		},
		{
			name:  "negativeValue",
			value: "-5.50",
			want:  -5.50,
		},
		{
			name:  "dollarNegative",
			value: "$-10.00",
			want:  -10.00,
		},
		{
			name:  "multipleDecimals",
			value: "10.50.25",
			want:  0,
		},
		{
			name:  "trailingSpaces",
			value: "$25.00",
			want:  25.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMoney(tt.value)
			if got != tt.want {
				t.Errorf("parseMoney(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   string
	}{
		{
			name:   "zeroAmount",
			amount: 0,
			want:   "$0.00",
		},
		{
			name:   "positiveAmount",
			amount: 15.99,
			want:   "$15.99",
		},
		{
			name:   "wholeNumber",
			amount: 100.00,
			want:   "$100.00",
		},
		{
			name:   "negativeAmount",
			amount: -5.50,
			want:   "$-5.50",
		},
		{
			name:   "smallDecimal",
			amount: 0.01,
			want:   "$0.01",
		},
		{
			name:   "largeAmount",
			amount: 1234567.89,
			want:   "$1234567.89",
		},
		{
			name:   "roundedDecimal",
			amount: 10.999,
			want:   "$11.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMoney(tt.amount)
			if got != tt.want {
				t.Errorf("formatMoney(%v) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "emptyString",
			value: "",
			want:  "Unknown",
		},
		{
			name:  "whitespaceOnly",
			value: "   ",
			want:  "Unknown",
		},
		{
			name:  "lowercaseWord",
			value: "pending",
			want:  "Pending",
		},
		{
			name:  "uppercaseWord",
			value: "ACTIVE",
			want:  "Active",
		},
		{
			name:  "mixedCaseWord",
			value: "pEnDiNg",
			want:  "Pending",
		},
		{
			name:  "singleCharLower",
			value: "a",
			want:  "A",
		},
		{
			name:  "singleCharUpper",
			value: "X",
			want:  "X",
		},
		{
			name:  "multiWordWithSpaces",
			value: "hello world",
			want:  "Hello world",
		},
		{
			name:  "leadingWhitespace",
			value: "  test",
			want:  "Test",
		},
		{
			name:  "trailingWhitespace",
			value: "test  ",
			want:  "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titleCase(tt.value)
			if got != tt.want {
				t.Errorf("titleCase(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

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
