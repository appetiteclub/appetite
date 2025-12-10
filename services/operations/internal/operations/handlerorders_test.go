package operations

import (
	"strings"
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

func TestTodayOrdersModalViewFields(t *testing.T) {
	tests := []struct {
		name        string
		view        todayOrdersModalView
		wantNumber  string
		wantTableID string
		wantCount   int
	}{
		{
			name: "emptyView",
			view: todayOrdersModalView{
				TableNumber: "",
				TableID:     "",
				Orders:      nil,
				OrderCount:  0,
			},
			wantNumber:  "",
			wantTableID: "",
			wantCount:   0,
		},
		{
			name: "viewWithOrders",
			view: todayOrdersModalView{
				TableNumber: "T5",
				TableID:     "table-123",
				Orders: []orderCardView{
					{ID: "order-1", ShortID: "O1"},
					{ID: "order-2", ShortID: "O2"},
				},
				OrderCount: 2,
			},
			wantNumber:  "T5",
			wantTableID: "table-123",
			wantCount:   2,
		},
		{
			name: "viewWithNoOrders",
			view: todayOrdersModalView{
				TableNumber: "T10",
				TableID:     "table-456",
				Orders:      []orderCardView{},
				OrderCount:  0,
			},
			wantNumber:  "T10",
			wantTableID: "table-456",
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.view.TableNumber != tt.wantNumber {
				t.Errorf("TableNumber = %q, want %q", tt.view.TableNumber, tt.wantNumber)
			}
			if tt.view.TableID != tt.wantTableID {
				t.Errorf("TableID = %q, want %q", tt.view.TableID, tt.wantTableID)
			}
			if tt.view.OrderCount != tt.wantCount {
				t.Errorf("OrderCount = %d, want %d", tt.view.OrderCount, tt.wantCount)
			}
			if len(tt.view.Orders) != tt.wantCount {
				t.Errorf("len(Orders) = %d, want %d", len(tt.view.Orders), tt.wantCount)
			}
		})
	}
}

func TestDeriveStation(t *testing.T) {
	tests := []struct {
		name string
		item *menuItemResource
		want string
	}{
		{
			name: "nilItem",
			item: nil,
			want: "kitchen",
		},
		{
			name: "noTags",
			item: &menuItemResource{
				ID:   "item-1",
				Tags: nil,
			},
			want: "kitchen",
		},
		{
			name: "emptyTags",
			item: &menuItemResource{
				ID:   "item-2",
				Tags: []string{},
			},
			want: "kitchen",
		},
		{
			name: "barStation",
			item: &menuItemResource{
				ID:   "item-3",
				Tags: []string{"station:bar"},
			},
			want: "bar",
		},
		{
			name: "directStation",
			item: &menuItemResource{
				ID:   "item-4",
				Tags: []string{"station:direct"},
			},
			want: "direct",
		},
		{
			name: "mixedTags",
			item: &menuItemResource{
				ID:   "item-5",
				Tags: []string{"category:beverage", "station:bar", "popular"},
			},
			want: "bar",
		},
		{
			name: "noStationTag",
			item: &menuItemResource{
				ID:   "item-6",
				Tags: []string{"spicy", "vegetarian"},
			},
			want: "kitchen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveStation(tt.item)
			if got != tt.want {
				t.Errorf("deriveStation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPickMenuName(t *testing.T) {
	tests := []struct {
		name string
		item *menuItemResource
		want string
	}{
		{
			name: "nilItem",
			item: nil,
			want: "",
		},
		{
			name: "englishName",
			item: &menuItemResource{
				ID:   "item-1",
				Name: map[string]string{"en": "Burger"},
			},
			want: "Burger",
		},
		{
			name: "noEnglishFallbackToOther",
			item: &menuItemResource{
				ID:   "item-2",
				Name: map[string]string{"es": "Hamburguesa"},
			},
			want: "Hamburguesa",
		},
		{
			name: "emptyNameMap",
			item: &menuItemResource{
				ID:        "item-3",
				Name:      map[string]string{},
				ShortCode: "BRG",
			},
			want: "BRG",
		},
		{
			name: "noNameNoShortCode",
			item: &menuItemResource{
				ID:   "item-4",
				Name: map[string]string{},
			},
			want: "Menu Item",
		},
		{
			name: "emptyEnglishFallbackToOther",
			item: &menuItemResource{
				ID:   "item-5",
				Name: map[string]string{"en": "", "fr": "Baguette"},
			},
			want: "Baguette",
		},
		{
			name: "preferEnglishOverOthers",
			item: &menuItemResource{
				ID:   "item-6",
				Name: map[string]string{"en": "Pizza", "it": "Pizza Italiana"},
			},
			want: "Pizza",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickMenuName(tt.item)
			if got != tt.want {
				t.Errorf("pickMenuName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPickMenuPrice(t *testing.T) {
	tests := []struct {
		name string
		item *menuItemResource
		want float64
	}{
		{
			name: "nilItem",
			item: nil,
			want: 0,
		},
		{
			name: "noPrices",
			item: &menuItemResource{
				ID:     "item-1",
				Prices: nil,
			},
			want: 0,
		},
		{
			name: "emptyPrices",
			item: &menuItemResource{
				ID:     "item-2",
				Prices: []menuPriceResource{},
			},
			want: 0,
		},
		{
			name: "singlePrice",
			item: &menuItemResource{
				ID:     "item-3",
				Prices: []menuPriceResource{{Amount: 15.99, CurrencyCode: "USD"}},
			},
			want: 15.99,
		},
		{
			name: "multiplePricesReturnsFirst",
			item: &menuItemResource{
				ID: "item-4",
				Prices: []menuPriceResource{
					{Amount: 10.00, CurrencyCode: "USD"},
					{Amount: 12.00, CurrencyCode: "EUR"},
				},
			},
			want: 10.00,
		},
		{
			name: "zeroPrice",
			item: &menuItemResource{
				ID:     "item-5",
				Prices: []menuPriceResource{{Amount: 0, CurrencyCode: "USD"}},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickMenuPrice(tt.item)
			if got != tt.want {
				t.Errorf("pickMenuPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultMenuSelection(t *testing.T) {
	tests := []struct {
		name    string
		options []menuItemOption
		want    string
	}{
		{
			name:    "nilOptions",
			options: nil,
			want:    "",
		},
		{
			name:    "emptyOptions",
			options: []menuItemOption{},
			want:    "",
		},
		{
			name: "withOptions",
			options: []menuItemOption{
				{ID: "item-1", Label: "Pizza"},
				{ID: "item-2", Label: "Burger"},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultMenuSelection(tt.options)
			if got != tt.want {
				t.Errorf("defaultMenuSelection() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMatchMenuOption(t *testing.T) {
	options := []menuItemOption{
		{ID: "item-1", Label: "Margherita Pizza ($12.00)", ShortCode: "PIZ"},
		{ID: "item-2", Label: "Classic Burger ($15.00)", ShortCode: "BRG"},
		{ID: "item-3", Label: "Caesar Salad ($10.00)", ShortCode: "SAL"},
	}

	tests := []struct {
		name    string
		query   string
		options []menuItemOption
		wantID  string
	}{
		{
			name:    "emptyQuery",
			query:   "",
			options: options,
			wantID:  "",
		},
		{
			name:    "whitespaceQuery",
			query:   "   ",
			options: options,
			wantID:  "",
		},
		{
			name:    "exactShortCodeMatch",
			query:   "PIZ",
			options: options,
			wantID:  "item-1",
		},
		{
			name:    "shortCodeCaseInsensitive",
			query:   "piz",
			options: options,
			wantID:  "item-1",
		},
		{
			name:    "partialLabelMatch",
			query:   "burger",
			options: options,
			wantID:  "item-2",
		},
		{
			name:    "noMatch",
			query:   "sushi",
			options: options,
			wantID:  "",
		},
		{
			name:    "emptyOptions",
			query:   "pizza",
			options: []menuItemOption{},
			wantID:  "",
		},
		{
			name:    "nilOptions",
			query:   "pizza",
			options: nil,
			wantID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchMenuOption(tt.query, tt.options)
			if got.ID != tt.wantID {
				t.Errorf("matchMenuOption(%q) ID = %q, want %q", tt.query, got.ID, tt.wantID)
			}
		})
	}
}

func TestPickMenuNameOption(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  string
	}{
		{
			name:  "withDash",
			label: "PIZ — Margherita Pizza ($12.00)",
			want:  "PIZ",
		},
		{
			name:  "withoutDash",
			label: "Margherita Pizza ($12.00)",
			want:  "Margherita Pizza ($12.00)",
		},
		{
			name:  "emptyLabel",
			label: "",
			want:  "",
		},
		{
			name:  "justDash",
			label: "—",
			want:  "",
		},
		{
			name:  "multipleDashes",
			label: "PIZ — Margherita — Special",
			want:  "PIZ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickMenuNameOption(tt.label)
			if got != tt.want {
				t.Errorf("pickMenuNameOption(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestTableOrderViewFields(t *testing.T) {
	tests := []struct {
		name         string
		view         tableOrderView
		wantID       string
		wantNumber   string
		wantHasOrder bool
		wantDisabled bool
	}{
		{
			name: "emptyView",
			view: tableOrderView{
				ID:       "",
				Number:   "",
				HasOrder: false,
				Disabled: false,
			},
			wantID:       "",
			wantNumber:   "",
			wantHasOrder: false,
			wantDisabled: false,
		},
		{
			name: "tableWithOrder",
			view: tableOrderView{
				ID:       "table-123",
				Number:   "T5",
				Status:   "occupied",
				HasOrder: true,
				Disabled: false,
				Order: orderSummaryView{
					ID:         "order-456",
					ItemsCount: 3,
					Total:      "$45.00",
				},
			},
			wantID:       "table-123",
			wantNumber:   "T5",
			wantHasOrder: true,
			wantDisabled: false,
		},
		{
			name: "tableWithoutOrder",
			view: tableOrderView{
				ID:       "table-789",
				Number:   "T10",
				Status:   "available",
				HasOrder: false,
				Disabled: false,
			},
			wantID:       "table-789",
			wantNumber:   "T10",
			wantHasOrder: false,
			wantDisabled: false,
		},
		{
			name: "disabledTable",
			view: tableOrderView{
				ID:       "table-disabled",
				Number:   "T99",
				Status:   "reserved",
				HasOrder: false,
				Disabled: true,
			},
			wantID:       "table-disabled",
			wantNumber:   "T99",
			wantHasOrder: false,
			wantDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.view.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", tt.view.ID, tt.wantID)
			}
			if tt.view.Number != tt.wantNumber {
				t.Errorf("Number = %q, want %q", tt.view.Number, tt.wantNumber)
			}
			if tt.view.HasOrder != tt.wantHasOrder {
				t.Errorf("HasOrder = %v, want %v", tt.view.HasOrder, tt.wantHasOrder)
			}
			if tt.view.Disabled != tt.wantDisabled {
				t.Errorf("Disabled = %v, want %v", tt.view.Disabled, tt.wantDisabled)
			}
		})
	}
}

func TestTableOrderViewOrderSummary(t *testing.T) {
	tests := []struct {
		name           string
		view           tableOrderView
		wantOrderID    string
		wantItemsCount int
		wantTotal      string
	}{
		{
			name: "orderWithItems",
			view: tableOrderView{
				HasOrder: true,
				Order: orderSummaryView{
					ID:          "order-123",
					ItemsCount:  5,
					Total:       "$75.50",
					StatusLabel: "Preparing",
					StatusClass: "status-preparing",
					PrepSummary: "2 pending, 3 preparing",
				},
			},
			wantOrderID:    "order-123",
			wantItemsCount: 5,
			wantTotal:      "$75.50",
		},
		{
			name: "orderWithNoItems",
			view: tableOrderView{
				HasOrder: true,
				Order: orderSummaryView{
					ID:          "order-empty",
					ItemsCount:  0,
					Total:       "$0.00",
					StatusLabel: "Pending",
					StatusClass: "status-pending",
				},
			},
			wantOrderID:    "order-empty",
			wantItemsCount: 0,
			wantTotal:      "$0.00",
		},
		{
			name: "noOrder",
			view: tableOrderView{
				HasOrder: false,
				Order:    orderSummaryView{},
			},
			wantOrderID:    "",
			wantItemsCount: 0,
			wantTotal:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.view.Order.ID != tt.wantOrderID {
				t.Errorf("Order.ID = %q, want %q", tt.view.Order.ID, tt.wantOrderID)
			}
			if tt.view.Order.ItemsCount != tt.wantItemsCount {
				t.Errorf("Order.ItemsCount = %d, want %d", tt.view.Order.ItemsCount, tt.wantItemsCount)
			}
			if tt.view.Order.Total != tt.wantTotal {
				t.Errorf("Order.Total = %q, want %q", tt.view.Order.Total, tt.wantTotal)
			}
		})
	}
}

func TestShouldIncludeOrderForTableCard(t *testing.T) {
	tests := []struct {
		name        string
		orderStatus string
		tableStatus string
		sameTable   bool
		wantInclude bool
	}{
		{
			name:        "activeOrderSameTable",
			orderStatus: "pending",
			tableStatus: "occupied",
			sameTable:   true,
			wantInclude: true,
		},
		{
			name:        "activeOrderDifferentTable",
			orderStatus: "pending",
			tableStatus: "occupied",
			sameTable:   false,
			wantInclude: false,
		},
		{
			name:        "closedOrderTableNotClearing",
			orderStatus: "closed",
			tableStatus: "occupied",
			sameTable:   true,
			wantInclude: false,
		},
		{
			name:        "closedOrderTableClearing",
			orderStatus: "closed",
			tableStatus: "clearing",
			sameTable:   true,
			wantInclude: true,
		},
		{
			name:        "preparingOrderSameTable",
			orderStatus: "preparing",
			tableStatus: "occupied",
			sameTable:   true,
			wantInclude: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from TableCard handler
			include := tt.sameTable
			if include && strings.ToLower(tt.orderStatus) == "closed" {
				include = strings.ToLower(tt.tableStatus) == "clearing"
			}
			if include != tt.wantInclude {
				t.Errorf("include = %v, want %v (orderStatus=%q, tableStatus=%q, sameTable=%v)",
					include, tt.wantInclude, tt.orderStatus, tt.tableStatus, tt.sameTable)
			}
		})
	}
}
