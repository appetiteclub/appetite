package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Orders view state mirrors the tables view flash handling.
type ordersPageState struct {
	Error   string
	Success string
}

// orderCardView powers each collapsible card in the template.
type orderCardView struct {
	ID               string
	ShortID          string
	TableID          string
	TableNumber      string
	TableStatus      string
	TableStatusLabel string
	Status           string
	StatusLabel      string
	StatusClass      string
	ItemsCount       int
	Total            string
	TotalValue       float64
	UpdatedAt        string
	CreatedAgo       string
	PrepSummary      string
	NonPrepSummary   string
	Groups           []orderGroupView
	Ungrouped        orderGroupView
	HasUngrouped     bool
	Items            []orderItemView
	GroupTotals      []orderGroupTotalView
	Events           []orderEventView
}

// orderGroupView aggregates the items assigned to a billing group.
type orderGroupView struct {
	ID        string
	Name      string
	ItemCount int
	Total     string
	Items     []orderItemView
}

type orderGroupTotalView struct {
	Name      string
	ItemCount int
	Total     string
}

// orderItemView contains the item detail displayed inside the card.
type orderItemView struct {
	ID                 string
	DishName           string
	Quantity           int
	UnitPrice          string
	Total              string
	Status             string
	StatusLabel        string
	StatusClass        string
	Category           string
	GroupName          string
	Notes              string
	CreatedAt          string
	RequiresProduction bool
}

type orderSummaryView struct {
	ID          string
	Status      string
	StatusLabel string
	StatusClass string
	Total       string
	ItemsCount  int
	PrepSummary string
}

type tableOrderView struct {
	ID          string
	Number      string
	Status      string
	StatusLabel string
	GuestCount  int
	AssignedTo  string
	Bill        string
	UpdatedAt   string
	Disabled    bool
	HasOrder    bool
	Order       orderSummaryView
	LastEvent   string
	LastWhen    string
}

type orderEventView struct {
	Message    string
	Occurred   string
	OccurredAt time.Time
}

// Lightweight DTOs for decoding service responses.
type menuItemResource struct {
	ID        string              `json:"id"`
	ShortCode string              `json:"short_code"`
	Name      map[string]string   `json:"name"`
	Prices    []menuPriceResource `json:"prices"`
	Tags      []string            `json:"tags"`
}

type menuPriceResource struct {
	Amount       float64 `json:"amount"`
	CurrencyCode string  `json:"currency_code"`
}

// Order creation modal payload.
type orderFormModal struct {
	Title         string
	Action        string
	Tables        []tableOption
	SelectedTable string
	Error         string
}

type tableOption struct {
	ID       string
	Label    string
	Status   string
	Disabled bool
}

// Order item creation modal payload.
type orderItemFormModal struct {
	Title          string
	Action         string
	OrderID        string
	OrderLabel     string
	Groups         []orderGroupResource
	MenuItems      []menuItemOption
	SelectedMenu   string
	Quantity       string
	Notes          string
	GroupID        string
	Error          string
	DisplayPrice   string
	DisplayRouting string
	MenuQuery      string
}

type menuItemOption struct {
	ID        string
	Label     string
	Price     float64
	Currency  string
	Routing   string
	ShortCode string
}

// Order group creation modal payload.
type orderGroupFormModal struct {
	Title     string
	Action    string
	OrderID   string
	TableID   string
	GroupName string
	Error     string
}

var orderStatusLabels = map[string]string{
	"pending":   "Pending",
	"preparing": "Preparing",
	"ready":     "Ready",
	"delivered": "Delivered",
	"cancelled": "Cancelled",
	"closed":    "Closed",
}

var orderStatusClasses = map[string]string{
	"pending":   "status-pending",
	"preparing": "status-preparing",
	"ready":     "status-ready",
	"delivered": "status-delivered",
	"cancelled": "status-cancelled",
	"closed":    "status-closed",
}

var orderItemStatusClasses = map[string]string{
	"pending":   "item-status-pending",
	"preparing": "item-status-preparing",
	"ready":     "item-status-ready",
	"delivered": "item-status-delivered",
	"cancelled": "item-status-cancelled",
}

var nonProductionCategories = map[string]bool{
	"addon":    true,
	"retail":   true,
	"beverage": true,
	"drink":    true,
}

var allowedTableStatuses = map[string]bool{
	"available": true,
	"open":      true,
	"reserved":  true,
	"clearing":  true,
}

// Orders renders the management interface replicating the tables page experience.
func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.Orders")
	defer finish()

	state := ordersPageState{}
	q := r.URL.Query()
	switch {
	case q.Get("created") == "1":
		state.Success = "Order opened successfully."
	case q.Get("item_added") == "1":
		state.Success = "Item added to the order."
	case q.Get("group_created") == "1":
		state.Success = "Group created for the table."
	}

	h.renderOrdersPage(w, r, state)
}

// OrderModal renders the full-screen modal for a specific order with items/groups.
func (h *Handler) OrderModal(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.OrderModal")
	defer finish()

	if !h.requirePermission(w, r, "orders:read") {
		return
	}

	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Error(w, "missing order id", http.StatusBadRequest)
		return
	}

	order, err := h.fetchOrder(r.Context(), orderID)
	if err != nil || order == nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	items, err := h.fetchOrderItems(r.Context(), order.ID)
	if err != nil {
		h.log().Error("cannot load order items for modal", "order_id", order.ID, "error", err)
		http.Error(w, "Could not load order items", http.StatusInternalServerError)
		return
	}

	table, _ := h.fetchTable(r.Context(), order.TableID)
	groups, _ := h.fetchOrderGroups(r.Context(), order.ID, map[string][]orderGroupResource{})
	tickets, _ := h.fetchOrderTickets(r.Context(), order.ID)

	card := h.buildOrderCard(*order, table, items, groups, tickets)
	h.renderOrderModal(w, card)
}

func (h *Handler) renderOrdersPage(w http.ResponseWriter, r *http.Request, state ordersPageState) {
	if !h.requirePermission(w, r, "orders:read") {
		return
	}

	ctx := r.Context()
	tables, tableErr := h.fetchTableList(ctx)
	if tableErr != nil {
		h.log().Error("unable to load tables for orders view", "error", tableErr)
		if state.Error == "" {
			state.Error = "Could not load tables from the service."
		}
	}

	orderCards, orderErr := h.fetchOrderCards(ctx)
	if orderErr != nil {
		h.log().Error("unable to load orders", "error", orderErr)
		if state.Error == "" {
			state.Error = "Could not load orders from the service."
		}
	}

	orderByTable := map[string]*orderCardView{}
	for i := range orderCards {
		orderByTable[orderCards[i].TableID] = &orderCards[i]
	}

	views := make([]tableOrderView, 0, len(tables))
	for _, tbl := range tables {
		var card *orderCardView
		if existing, ok := orderByTable[tbl.ID]; ok {
			card = existing
		}
		views = append(views, h.buildTableOrderView(tbl, card))
	}

	data := map[string]interface{}{
		"Title":    "Orders",
		"Template": "orders",
		"User":     h.getUserFromSession(r),
		"Tables":   views,
		"Error":    state.Error,
		"Success":  state.Success,
	}

	h.renderTemplate(w, "orders.html", "base.html", data)
}

func (h *Handler) fetchOrderCards(ctx context.Context) ([]orderCardView, error) {
	orders, err := h.orderData.ListOrders(ctx)
	if err != nil {
		return nil, err
	}

	tableMap, err := h.fetchTableMap(ctx)
	if err != nil {
		return nil, err
	}

	groupCache := map[string][]orderGroupResource{}
	cards := make([]orderCardView, 0, len(orders))

	for _, order := range orders {
		// Skip closed orders unless the table is in "clearing" status (takeaway scenario)
		if strings.ToLower(order.Status) == "closed" {
			table := tableMap[order.TableID]
			if table == nil || strings.ToLower(table.Status) != "clearing" {
				continue
			}
		}
		items, itemErr := h.fetchOrderItems(ctx, order.ID)
		if itemErr != nil {
			h.log().Error("cannot load order items", "order_id", order.ID, "error", itemErr)
		}

		groups, groupErr := h.fetchOrderGroups(ctx, order.ID, groupCache)
		if groupErr != nil {
			h.log().Error("cannot load groups for table", "table_id", order.TableID, "error", groupErr)
		}

		card := h.buildOrderCard(order, tableMap[order.TableID], items, groups, nil)
		cards = append(cards, card)
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].TotalValue > cards[j].TotalValue
	})

	return cards, nil
}

func (h *Handler) fetchTableMap(ctx context.Context) (map[string]*tableResource, error) {
	tables, err := h.tableData.ListTables(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*tableResource, len(tables))
	for i := range tables {
		tbl := tables[i]
		result[tbl.ID] = &tbl
	}

	return result, nil
}

func (h *Handler) fetchTableList(ctx context.Context) ([]tableResource, error) {
	tables, err := h.tableData.ListTables(ctx)
	if err != nil {
		return nil, err
	}

	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Number < tables[j].Number
	})

	return tables, nil
}

func (h *Handler) fetchOrderItems(ctx context.Context, orderID string) ([]orderItemResource, error) {
	return h.orderData.ListOrderItems(ctx, orderID)
}

func (h *Handler) fetchOrderGroups(ctx context.Context, orderID string, cache map[string][]orderGroupResource) ([]orderGroupResource, error) {
	if orderID == "" {
		return nil, nil
	}
	if groups, ok := cache[orderID]; ok {
		return groups, nil
	}

	groups, err := h.orderData.ListOrderGroups(ctx, orderID)
	if err != nil {
		return nil, err
	}

	cache[orderID] = groups
	return groups, nil
}

func (h *Handler) fetchOrderTickets(ctx context.Context, orderID string) ([]kitchenTicketResource, error) {
	if orderID == "" || h.kitchenData == nil {
		return nil, nil
	}

	tickets, err := h.kitchenData.ListTicketsByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return tickets, nil
}

func (h *Handler) buildOrderCard(order orderResource, table *tableResource, items []orderItemResource, groups []orderGroupResource, tickets []kitchenTicketResource) orderCardView {
	tableNumber := "Unassigned"
	tableStatus := ""
	tableStatusLabel := ""
	if table != nil {
		tableNumber = table.Number
		tableStatus = table.Status
		tableStatusLabel = humanizeStatus(table.Status)
	}

	shortID := shortOrderID(order.ID)
	statusKey := strings.ToLower(order.Status)
	statusLabel := orderStatusLabels[statusKey]
	if statusLabel == "" {
		statusLabel = titleCase(order.Status)
	}

	statusClass := orderStatusClasses[statusKey]
	if statusClass == "" {
		statusClass = "status-pending"
	}

	counts := map[string]int{}
	nonPrep := 0
	total := 0.0

	groupLookups := map[string]*orderGroupView{}
	for _, group := range groups {
		groupCopy := orderGroupView{ID: group.ID, Name: group.Name}
		groupLookups[group.ID] = &groupCopy
	}

	itemViews := make([]orderItemView, 0, len(items))
	for _, item := range items {
		statusKey := strings.ToLower(item.Status)
		counts[statusKey]++
		total += float64(item.Quantity) * item.Price

		requiresProduction := !nonProductionCategories[strings.ToLower(strings.TrimSpace(item.Category))]
		if !requiresProduction {
			nonPrep += item.Quantity
		}

		groupName := ""
		if item.GroupID != nil && groupLookups[*item.GroupID] != nil {
			groupName = groupLookups[*item.GroupID].Name
		}

		itemView := orderItemView{
			ID:                 item.ID,
			DishName:           item.DishName,
			Quantity:           item.Quantity,
			UnitPrice:          formatMoney(item.Price),
			Total:              formatMoney(float64(item.Quantity) * item.Price),
			Status:             statusKey,
			StatusLabel:        orderStatusLabels[statusKey],
			StatusClass:        orderItemStatusClasses[statusKey],
			Category:           item.Category,
			GroupName:          groupName,
			Notes:              item.Notes,
			CreatedAt:          relativeTimeSince(item.CreatedAt),
			RequiresProduction: requiresProduction,
		}

		if itemView.StatusLabel == "" {
			itemView.StatusLabel = titleCase(item.Status)
		}
		if itemView.StatusClass == "" {
			itemView.StatusClass = "item-status-pending"
		}

		itemViews = append(itemViews, itemView)
	}

	groupSections := make([]orderGroupView, 0, len(groupLookups))
	for _, section := range groupLookups {
		section.Items = []orderItemView{}
		groupSections = append(groupSections, *section)
	}

	ungrouped := orderGroupView{Name: "Ungrouped"}
	for _, item := range itemViews {
		if item.GroupName == "" {
			ungrouped.Items = append(ungrouped.Items, item)
			ungrouped.ItemCount += item.Quantity
			subtotal, _ := strconv.ParseFloat(strings.TrimPrefix(item.Total, "$"), 64)
			ungroupedTotal := parseMoney(ungrouped.Total)
			ungrouped.Total = formatMoney(ungroupedTotal + subtotal)
			continue
		}
		for idx := range groupSections {
			if groupSections[idx].Name == item.GroupName {
				groupSections[idx].Items = append(groupSections[idx].Items, item)
				groupSections[idx].ItemCount += item.Quantity
				subtotal, _ := strconv.ParseFloat(strings.TrimPrefix(item.Total, "$"), 64)
				sectionTotal := parseMoney(groupSections[idx].Total)
				groupSections[idx].Total = formatMoney(sectionTotal + subtotal)
				break
			}
		}
	}

	sort.Slice(groupSections, func(i, j int) bool {
		return groupSections[i].Name < groupSections[j].Name
	})

	card := orderCardView{
		ID:               order.ID,
		ShortID:          shortID,
		TableID:          order.TableID,
		TableNumber:      tableNumber,
		TableStatus:      tableStatus,
		TableStatusLabel: tableStatusLabel,
		Status:           order.Status,
		StatusLabel:      statusLabel,
		StatusClass:      statusClass,
		ItemsCount:       len(items),
		Total:            formatMoney(total),
		TotalValue:       total,
		UpdatedAt:        relativeTimeSince(order.UpdatedAt),
		CreatedAgo:       relativeTimeSince(order.CreatedAt),
		PrepSummary:      summarizePrep(counts),
		NonPrepSummary:   summarizeNonPrep(nonPrep),
		Groups:           groupSections,
		Ungrouped:        ungrouped,
		HasUngrouped:     len(ungrouped.Items) > 0,
		Items:            itemViews,
	}

	if len(groupSections) > 0 || ungrouped.ItemCount > 0 {
		groupTotals := make([]orderGroupTotalView, 0, len(groupSections)+1)
		for _, section := range groupSections {
			if section.ItemCount == 0 {
				continue
			}
			groupTotals = append(groupTotals, orderGroupTotalView{
				Name:      section.Name,
				ItemCount: section.ItemCount,
				Total:     section.Total,
			})
		}
		if ungrouped.ItemCount > 0 {
			groupTotals = append(groupTotals, orderGroupTotalView{
				Name:      ungrouped.Name,
				ItemCount: ungrouped.ItemCount,
				Total:     ungrouped.Total,
			})
		}
		card.GroupTotals = groupTotals
	}

	card.Events = h.buildOrderEvents(order, table, items, tickets, statusLabel)

	return card
}

func (h *Handler) buildOrderEvents(order orderResource, table *tableResource, items []orderItemResource, tickets []kitchenTicketResource, statusLabel string) []orderEventView {
	type timelineEvent struct {
		message  string
		occurred time.Time
	}

	events := make([]timelineEvent, 0, len(items)*2+4)
	addEvent := func(message string, ts time.Time) {
		if ts.IsZero() {
			return
		}
		trimmed := strings.TrimSpace(message)
		if trimmed == "" {
			return
		}
		events = append(events, timelineEvent{message: trimmed, occurred: ts})
	}

	if statusLabel == "" {
		statusLabel = titleCase(order.Status)
	}
	addEvent(fmt.Sprintf("Order %s", statusLabel), order.UpdatedAt)
	addEvent("Order created", order.CreatedAt)

	if table != nil {
		addEvent(fmt.Sprintf("Table %s", humanizeStatus(table.Status)), table.UpdatedAt)
	}

	itemLookup := make(map[string]orderItemResource, len(items))
	for _, item := range items {
		itemLookup[item.ID] = item
		dish := item.DishName
		if dish == "" {
			dish = "Menu item"
		}
		addEvent(fmt.Sprintf("Added %s ×%d", dish, item.Quantity), item.CreatedAt)

		itemStatusLabel := orderStatusLabels[strings.ToLower(item.Status)]
		if itemStatusLabel == "" {
			itemStatusLabel = titleCase(item.Status)
		}
		statusKey := strings.ToLower(item.Status)
		if statusKey != "pending" || !item.UpdatedAt.Equal(item.CreatedAt) {
			addEvent(fmt.Sprintf("%s marked %s", dish, itemStatusLabel), item.UpdatedAt)
		}
	}

	if len(tickets) > 0 {
		for _, ticket := range tickets {
			item, ok := itemLookup[ticket.OrderItemID]
			dish := item.DishName
			if !ok || dish == "" {
				dish = fmt.Sprintf("Item %s", truncateID(ticket.OrderItemID))
			}
			quantity := ticket.Quantity
			if quantity == 0 && ok {
				quantity = item.Quantity
			}
			prepLabel := routingLabel(item.Category)
			if prepLabel == "" {
				prepLabel = "Kitchen"
			}
			lineItem := fmt.Sprintf("%s ×%d", dish, quantity)
			addEvent(fmt.Sprintf("%s ticket queued · %s", prepLabel, lineItem), ticket.CreatedAt)
			if ticket.StartedAt != nil {
				addEvent(fmt.Sprintf("Prep started · %s", lineItem), *ticket.StartedAt)
			}
			if ticket.FinishedAt != nil {
				addEvent(fmt.Sprintf("Ready for pickup · %s", lineItem), *ticket.FinishedAt)
			}
			if ticket.DeliveredAt != nil {
				addEvent(fmt.Sprintf("Delivered · %s", lineItem), *ticket.DeliveredAt)
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].occurred.Equal(events[j].occurred) {
			return events[i].message < events[j].message
		}
		return events[i].occurred.After(events[j].occurred)
	})

	views := make([]orderEventView, 0, len(events))
	for _, event := range events {
		views = append(views, orderEventView{
			Message:    event.message,
			Occurred:   relativeTimeSince(event.occurred),
			OccurredAt: event.occurred,
		})
	}

	return views
}

func (h *Handler) buildTableOrderView(table tableResource, order *orderCardView) tableOrderView {
	assigned := "-"
	if table.AssignedTo != nil && *table.AssignedTo != "" {
		assigned = truncateID(*table.AssignedTo)
	}

	view := tableOrderView{
		ID:          table.ID,
		Number:      table.Number,
		Status:      table.Status,
		StatusLabel: humanizeStatus(table.Status),
		GuestCount:  table.GuestCount,
		AssignedTo:  assigned,
		Bill:        formatBill(table.CurrentBill),
		UpdatedAt:   relativeTimeSince(table.UpdatedAt),
		Disabled:    !allowedTableStatuses[strings.ToLower(table.Status)],
	}

	if order != nil {
		view.HasOrder = true
		view.Order = orderSummaryView{
			ID:          order.ID,
			Status:      order.Status,
			StatusLabel: order.StatusLabel,
			StatusClass: order.StatusClass,
			Total:       order.Total,
			ItemsCount:  order.ItemsCount,
			PrepSummary: order.PrepSummary,
		}
		if len(order.Events) > 0 {
			view.LastEvent = order.Events[0].Message
			view.LastWhen = order.Events[0].Occurred
		} else {
			view.LastEvent = fmt.Sprintf("%s · %s", order.StatusLabel, order.PrepSummary)
			view.LastWhen = order.UpdatedAt
		}
	}

	if view.LastEvent == "" {
		view.LastEvent = fmt.Sprintf("Table %s", view.StatusLabel)
		view.LastWhen = view.UpdatedAt
	}

	return view
}

func summarizePrep(counts map[string]int) string {
	parts := []string{}
	if counts["pending"] > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", counts["pending"]))
	}
	if counts["preparing"] > 0 {
		parts = append(parts, fmt.Sprintf("%d preparing", counts["preparing"]))
	}
	if counts["ready"] > 0 {
		parts = append(parts, fmt.Sprintf("%d ready", counts["ready"]))
	}
	if len(parts) == 0 {
		return "No kitchen items"
	}
	return strings.Join(parts, " • ")
}

func summarizeNonPrep(count int) string {
	if count == 0 {
		return ""
	}
	return fmt.Sprintf("%d direct-to-check items", count)
}

func parseMoney(value string) float64 {
	if value == "" {
		return 0
	}
	trimmed := strings.TrimPrefix(value, "$")
	result, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return result
}

func formatMoney(amount float64) string {
	return fmt.Sprintf("$%.2f", amount)
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Unknown"
	}
	lower := strings.ToLower(value)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

// ----- Order creation -----

func (h *Handler) NewOrderForm(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "orders:write") {
		return
	}

	options, err := h.collectTableOptions(r.Context())
	if err != nil {
		h.log().Error("cannot load table options", "error", err)
		http.Error(w, "Unable to load tables", http.StatusInternalServerError)
		return
	}

	selected := strings.TrimSpace(r.URL.Query().Get("table_id"))
	form := orderFormModal{
		Title:         "Open Order",
		Action:        "/add-order",
		Tables:        options,
		SelectedTable: selected,
	}

	h.renderOrderForm(w, form)
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.CreateOrder")
	defer finish()

	if !h.requirePermission(w, r, "orders:write") {
		return
	}

	isHTMX := aqm.IsHTMX(r)
	if err := r.ParseForm(); err != nil {
		if isHTMX {
			http.Error(w, "Could not read the submitted form", http.StatusBadRequest)
		} else {
			h.handleOrderFormError(w, orderFormModal{Title: "Open Order", Action: "/add-order"}, "Could not read the submitted form.")
		}
		return
	}

	tableID := strings.TrimSpace(r.FormValue("table_id"))
	options, _ := h.collectTableOptions(r.Context())
	form := orderFormModal{
		Title:         "Open Order",
		Action:        "/add-order",
		Tables:        options,
		SelectedTable: tableID,
	}

	if tableID == "" {
		if isHTMX {
			http.Error(w, "table_id is required", http.StatusBadRequest)
		} else {
			h.handleOrderFormError(w, form, "Please select a table before opening an order.")
		}
		return
	}

	if !h.tableAllowsOrdering(r.Context(), tableID) {
		if isHTMX {
			http.Error(w, "This table cannot accept orders in its current status.", http.StatusBadRequest)
		} else {
			h.handleOrderFormError(w, form, "This table cannot accept orders in its current status.")
		}
		return
	}

	order, err := h.orderData.CreateOrder(r.Context(), CreateOrderRequest{TableID: tableID})
	if err != nil {
		h.log().Error("order service create failed", "table_id", tableID, "error", err)
		if isHTMX {
			http.Error(w, "Could not open the order right now.", http.StatusInternalServerError)
		} else {
			h.handleOrderFormError(w, form, "Could not open the order right now.")
		}
		return
	}

	if isHTMX {
		items := []orderItemResource{}
		table, _ := h.fetchTable(r.Context(), order.TableID)
		groups, _ := h.fetchOrderGroups(r.Context(), order.ID, map[string][]orderGroupResource{})
		card := h.buildOrderCard(*order, table, items, groups, nil)
		h.renderOrderModal(w, card)
		return
	}

	aqm.RedirectOrHeader(w, r, "/orders?created=1")
}

func (h *Handler) collectTableOptions(ctx context.Context) ([]tableOption, error) {
	tables, err := h.tableData.ListTables(ctx)
	if err != nil {
		return nil, err
	}

	options := make([]tableOption, 0, len(tables))
	for _, tbl := range tables {
		disabled := !allowedTableStatuses[strings.ToLower(tbl.Status)]
		label := fmt.Sprintf("%s (%s)", tbl.Number, humanizeStatus(tbl.Status))
		options = append(options, tableOption{ID: tbl.ID, Label: label, Status: tbl.Status, Disabled: disabled})
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	return options, nil
}

func (h *Handler) tableAllowsOrdering(ctx context.Context, tableID string) bool {
	if tableID == "" {
		return false
	}
	table, err := h.fetchTable(ctx, tableID)
	if err != nil || table == nil {
		return false
	}
	return allowedTableStatuses[strings.ToLower(table.Status)]
}

func (h *Handler) renderOrderForm(w http.ResponseWriter, data orderFormModal) {
	tmpl, err := h.tmplMgr.Get("orders_form.html")
	if err != nil {
		h.log().Error("error loading order form template", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "orders_form.html", data); err != nil {
		h.log().Error("error rendering order form", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handler) handleOrderFormError(w http.ResponseWriter, data orderFormModal, message string) {
	data.Error = message
	h.renderOrderForm(w, data)
}

// ----- Order item creation -----

func (h *Handler) NewOrderItemForm(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "orders:write") {
		return
	}

	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Error(w, "Missing order ID", http.StatusBadRequest)
		return
	}

	order, err := h.fetchOrder(r.Context(), orderID)
	if err != nil || order == nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	menuItems, err := h.fetchMenuOptions(r.Context())
	if err != nil {
		h.log().Error("cannot load menu items", "error", err)
		http.Error(w, "Unable to load menu items", http.StatusInternalServerError)
		return
	}

	groups, _ := h.fetchOrderGroups(r.Context(), order.ID, map[string][]orderGroupResource{})

	form := orderItemFormModal{
		Title:      fmt.Sprintf("Add Item to %s", shortOrderID(order.ID)),
		Action:     fmt.Sprintf("/orders/%s/items", order.ID),
		OrderID:    order.ID,
		OrderLabel: shortOrderID(order.ID),
		MenuItems:  menuItems,
		Groups:     groups,
		Quantity:   "1",
		MenuQuery:  "",
	}
	form.ensureGroupSelection()

	form.SelectedMenu = defaultMenuSelection(menuItems)
	form.populateOrderItemPreview()
	h.renderOrderItemForm(w, form)
}

func (h *Handler) CreateOrderItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.CreateOrderItem")
	defer finish()

	if !h.requirePermission(w, r, "orders:write") {
		return
	}

	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		http.Error(w, "Missing order ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.handleOrderItemFormError(w, orderItemFormModal{}, "Could not read the submitted form.")
		return
	}

	menuItemID := strings.TrimSpace(r.FormValue("menu_item_id"))
	quantityStr := strings.TrimSpace(r.FormValue("quantity"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	groupIDStr := strings.TrimSpace(r.FormValue("group_id"))
	menuQuery := strings.TrimSpace(r.FormValue("menu_item_query"))

	form := orderItemFormModal{
		Title:        fmt.Sprintf("Add Item to %s", shortOrderID(orderID)),
		Action:       fmt.Sprintf("/orders/%s/items", orderID),
		OrderID:      orderID,
		MenuItems:    []menuItemOption{},
		Quantity:     quantityStr,
		Notes:        notes,
		GroupID:      groupIDStr,
		SelectedMenu: menuItemID,
		MenuQuery:    menuQuery,
	}

	if menuItemID == "" {
		h.handleOrderItemFormError(w, form, "Choose an item from the menu.")
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		h.handleOrderItemFormError(w, form, "Quantity must be a positive number.")
		return
	}

	menuItem, err := h.fetchMenuItem(r.Context(), menuItemID)
	if err != nil {
		h.log().Error("cannot load menu item", "menu_item_id", menuItemID, "error", err)
		h.handleOrderItemFormError(w, form, "Could not load the selected menu item.")
		return
	}

	price := pickMenuPrice(menuItem)
	routing := deriveStation(menuItem)
	form.DisplayPrice = formatMoney(price)
	form.DisplayRouting = routingLabel(routing)

	defaultGroup := h.defaultOrderGroupID(r.Context(), orderID)
	if groupIDStr == "" && defaultGroup != "" {
		groupIDStr = defaultGroup
	}

	dishName := pickMenuName(menuItem)
	requiresProduction := routing != "direct" && routing != ""

	payload := map[string]interface{}{
		"dish_name":          dishName,
		"category":           routing,
		"quantity":           quantity,
		"price":              price,
		"menu_item_id":       menuItemID,
		"requires_production": requiresProduction,
	}

	if notes != "" {
		payload["notes"] = notes
	}

	if groupIDStr != "" {
		if _, err := uuid.Parse(groupIDStr); err == nil {
			payload["group_id"] = groupIDStr
		}
	}

	// Get production station ID from dictionary if needed
	if requiresProduction {
		if stationID, err := h.getStationIDByName(r.Context(), routing); err == nil && stationID != "" {
			payload["production_station"] = stationID
		}
	}

	path := fmt.Sprintf("/orders/%s/items", orderID)
	if _, err := h.orderClient.Request(r.Context(), "POST", path, payload); err != nil {
		h.log().Error("order item creation failed", "order_id", orderID, "error", err)
		h.handleOrderItemFormError(w, form, "Could not add the item right now.")
		return
	}

	if aqm.IsHTMX(r) {
		h.renderOrderModalFor(r.Context(), orderID, w, r)
		return
	}

	aqm.RedirectOrHeader(w, r, "/orders?item_added=1")
}

func (h *Handler) renderOrderItemForm(w http.ResponseWriter, data orderItemFormModal) {
	h.enrichOrderItemForm(context.Background(), &data)

	tmpl, err := h.tmplMgr.Get("order_items_form.html")
	if err != nil {
		h.log().Error("error loading order item form template", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "order_items_form.html", data); err != nil {
		h.log().Error("error rendering order item form", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handler) handleOrderItemFormError(w http.ResponseWriter, data orderItemFormModal, message string) {
	data.Error = message
	h.enrichOrderItemForm(context.Background(), &data)
	h.renderOrderItemForm(w, data)
}

func (h *Handler) fetchMenuOptions(ctx context.Context) ([]menuItemOption, error) {
	resp, err := h.menuClient.Request(ctx, "GET", "/menu/items?active=true", nil)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, err
	}

	var items []menuItemResource
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}

	options := make([]menuItemOption, 0, len(items))
	for _, item := range items {
		price := 0.0
		currency := "USD"
		if len(item.Prices) > 0 {
			price = item.Prices[0].Amount
			if item.Prices[0].CurrencyCode != "" {
				currency = item.Prices[0].CurrencyCode
			}
		}
		name := pickMenuName(&item)
		label := fmt.Sprintf("%s (%s)", name, formatMoney(price))
		if item.ShortCode != "" {
			label = fmt.Sprintf("%s — %s (%s)", item.ShortCode, name, formatMoney(price))
		}
		routing := deriveStation(&item)
		options = append(options, menuItemOption{
			ID:        item.ID,
			Label:     label,
			Price:     price,
			Currency:  currency,
			Routing:   routing,
			ShortCode: item.ShortCode,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	return options, nil
}

func (h *Handler) fetchMenuItem(ctx context.Context, id string) (*menuItemResource, error) {
	path := fmt.Sprintf("/menu/items/%s", id)
	resp, err := h.menuClient.Request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, err
	}

	var item menuItemResource
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

func pickMenuName(item *menuItemResource) string {
	if item == nil {
		return ""
	}
	if name, ok := item.Name["en"]; ok && name != "" {
		return name
	}
	for _, value := range item.Name {
		if value != "" {
			return value
		}
	}
	if item.ShortCode != "" {
		return item.ShortCode
	}
	return "Menu Item"
}

func pickMenuPrice(item *menuItemResource) float64 {
	if item == nil || len(item.Prices) == 0 {
		return 0
	}
	return item.Prices[0].Amount
}

func deriveStation(item *menuItemResource) string {
	if item == nil {
		return "kitchen"
	}
	for _, tag := range item.Tags {
		if strings.HasPrefix(tag, "station:") {
			return strings.TrimPrefix(tag, "station:")
		}
	}
	return "kitchen"
}

func routingLabel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bar":
		return "Bar station"
	case "direct":
		return "Direct to bill"
	default:
		return "Kitchen"
	}
}

func (h *Handler) getStationIDByName(ctx context.Context, stationName string) (string, error) {
	stationName = strings.ToLower(strings.TrimSpace(stationName))

	// Known stations from enum
	knownStations := map[string]bool{
		"kitchen": true,
		"dessert": true,
		"bar":     true,
		"coffee":  true,
		"other":   true,
	}

	// If station is known, return it
	if knownStations[stationName] {
		return stationName, nil
	}

	// Map unknown/invalid stations to "other"
	return "other", nil
}

func defaultMenuSelection(options []menuItemOption) string {
	// Do not pre-select any menu item - user must explicitly choose
	return ""
}

func matchMenuOption(query string, options []menuItemOption) menuItemOption {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	if trimmed == "" {
		// Do not return a default item when query is empty - user must type to search
		return menuItemOption{}
	}
	for _, opt := range options {
		if strings.EqualFold(opt.ShortCode, trimmed) {
			return opt
		}
	}
	for _, opt := range options {
		if strings.Contains(strings.ToLower(opt.Label), trimmed) {
			return opt
		}
	}
	return menuItemOption{}
}

func (h *Handler) enrichOrderItemForm(ctx context.Context, form *orderItemFormModal) {
	if len(form.MenuItems) == 0 {
		menuItems, _ := h.fetchMenuOptions(ctx)
		form.MenuItems = menuItems
	}
	if form.Groups == nil && form.OrderID != "" {
		order, _ := h.fetchOrder(ctx, form.OrderID)
		if order != nil {
			groups, _ := h.fetchOrderGroups(ctx, order.ID, map[string][]orderGroupResource{})
			form.Groups = groups
		}
	}
	form.ensureGroupSelection()
	if form.Quantity == "" {
		form.Quantity = "1"
	}
	form.populateOrderItemPreview()
	if form.MenuQuery == "" && form.SelectedMenu != "" {
		if opt, ok := form.findOption(form.SelectedMenu); ok {
			if opt.ShortCode != "" {
				form.MenuQuery = opt.ShortCode
			} else {
				form.MenuQuery = pickMenuNameOption(opt.Label)
			}
		}
	}
}

func (form *orderItemFormModal) populateOrderItemPreview() {
	if form.SelectedMenu == "" {
		form.SelectedMenu = defaultMenuSelection(form.MenuItems)
	}
	for _, opt := range form.MenuItems {
		if opt.ID == form.SelectedMenu {
			form.DisplayPrice = formatMoney(opt.Price)
			form.DisplayRouting = routingLabel(opt.Routing)
			return
		}
	}
	if len(form.MenuItems) > 0 {
		form.DisplayPrice = formatMoney(form.MenuItems[0].Price)
		form.DisplayRouting = routingLabel(form.MenuItems[0].Routing)
	}
}

func (form *orderItemFormModal) findOption(id string) (menuItemOption, bool) {
	for _, opt := range form.MenuItems {
		if opt.ID == id {
			return opt, true
		}
	}
	return menuItemOption{}, false
}

func (form *orderItemFormModal) ensureGroupSelection() {
	if form.GroupID != "" {
		return
	}
	for _, group := range form.Groups {
		if group.IsDefault {
			form.GroupID = group.ID
			return
		}
	}
	if len(form.Groups) > 0 {
		form.GroupID = form.Groups[0].ID
	}
}

func pickMenuNameOption(label string) string {
	parts := strings.SplitN(label, "—", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0])
	}
	return label
}

func (h *Handler) fetchOrder(ctx context.Context, id string) (*orderResource, error) {
	return h.orderData.GetOrder(ctx, id)
}

// ----- Order group creation -----

func (h *Handler) NewOrderGroupForm(w http.ResponseWriter, r *http.Request) {
	if !h.requirePermission(w, r, "orders:write") {
		return
	}

	orderID := chi.URLParam(r, "id")
	order, err := h.fetchOrder(r.Context(), orderID)
	if err != nil || order == nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	form := orderGroupFormModal{
		Title:   fmt.Sprintf("New Group for %s", shortOrderID(order.ID)),
		Action:  fmt.Sprintf("/orders/%s/groups", order.ID),
		OrderID: order.ID,
		TableID: order.TableID,
	}

	h.renderOrderGroupForm(w, form)
}

func (h *Handler) CreateOrderGroup(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.CreateOrderGroup")
	defer finish()

	if !h.requirePermission(w, r, "orders:write") {
		return
	}

	orderID := chi.URLParam(r, "id")
	order, err := h.fetchOrder(r.Context(), orderID)
	if err != nil || order == nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.handleOrderGroupFormError(w, orderGroupFormModal{OrderID: orderID, TableID: order.TableID}, "Could not read the submitted form.")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	form := orderGroupFormModal{
		Title:     fmt.Sprintf("New Group for %s", shortOrderID(order.ID)),
		Action:    fmt.Sprintf("/orders/%s/groups", order.ID),
		OrderID:   order.ID,
		TableID:   order.TableID,
		GroupName: name,
	}

	if name == "" {
		h.handleOrderGroupFormError(w, form, "Group name is required.")
		return
	}

	body := map[string]interface{}{
		"name": name,
	}

	path := fmt.Sprintf("/orders/%s/groups", order.ID)
	if _, err := h.orderClient.Request(r.Context(), "POST", path, body); err != nil {
		h.log().Error("order service group create failed", "order_id", order.ID, "error", err)
		h.handleOrderGroupFormError(w, form, "Could not create the group right now.")
		return
	}

	if aqm.IsHTMX(r) {
		h.renderOrderModalFor(r.Context(), order.ID, w, r)
		return
	}

	aqm.RedirectOrHeader(w, r, "/orders?group_created=1")
}

func (h *Handler) renderOrderGroupForm(w http.ResponseWriter, data orderGroupFormModal) {
	tmpl, err := h.tmplMgr.Get("order_groups_form.html")
	if err != nil {
		h.log().Error("error loading order group form template", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "order_groups_form.html", data); err != nil {
		h.log().Error("error rendering order group form", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handler) handleOrderGroupFormError(w http.ResponseWriter, data orderGroupFormModal, message string) {
	data.Error = message
	h.renderOrderGroupForm(w, data)
}

// Menu helpers for HTMX search/preview

func (h *Handler) OrderMenuMatch(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.http.Start(w, r, "Handler.OrderMenuMatch")
	defer finish()

	if !h.requirePermission(w, r, "orders:write") {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("menu_item_query"))
	options, err := h.fetchMenuOptions(r.Context())
	if err != nil {
		http.Error(w, "Unable to load menu", http.StatusInternalServerError)
		return
	}

	match := matchMenuOption(query, options)
	form := orderItemFormModal{
		MenuItems:    options,
		SelectedMenu: match.ID,
		MenuQuery:    query,
	}
	if match.ID != "" {
		form.DisplayPrice = formatMoney(match.Price)
		form.DisplayRouting = routingLabel(match.Routing)
		if match.ShortCode != "" {
			form.MenuQuery = match.ShortCode
		}
	}
	form.populateOrderItemPreview()
	h.renderOrderItemPreview(w, form)
	if match.ID != "" {
		fmt.Fprintf(w, "<div id=\"menu-item-hidden\" hx-swap-oob=\"true\"><input type=\"hidden\" name=\"menu_item_id\" value=\"%s\"></div>", match.ID)
	} else {
		fmt.Fprintf(w, "<div id=\"menu-item-hidden\" hx-swap-oob=\"true\"><input type=\"hidden\" name=\"menu_item_id\" value=\"\"></div>")
	}
}

func (h *Handler) renderOrderItemPreview(w http.ResponseWriter, data orderItemFormModal) {
	tmpl, err := h.tmplMgr.Get("order_item_preview.html")
	if err != nil {
		h.log().Error("error loading preview template", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "order_item_preview.html", data); err != nil {
		h.log().Error("error rendering preview", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handler) renderOrderModal(w http.ResponseWriter, order orderCardView) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	tmpl, err := h.tmplMgr.Get("order_modal.html")
	if err != nil {
		h.log().Error("error loading order modal template", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Order": order,
	}

	if err := tmpl.ExecuteTemplate(w, "order_modal.html", data); err != nil {
		h.log().Error("error rendering order modal", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *Handler) renderOrderModalFor(ctx context.Context, orderID string, w http.ResponseWriter, r *http.Request) {
	order, err := h.orderData.GetOrder(ctx, orderID)
	if err != nil || order == nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}
	items, _ := h.fetchOrderItems(ctx, orderID)
	table, _ := h.fetchTable(ctx, order.TableID)
	groups, _ := h.fetchOrderGroups(ctx, order.ID, map[string][]orderGroupResource{})
	tickets, _ := h.fetchOrderTickets(ctx, order.ID)
	card := h.buildOrderCard(*order, table, items, groups, tickets)
	h.renderOrderModal(w, card)
}

func (h *Handler) defaultOrderGroupID(ctx context.Context, orderID string) string {
	groups, err := h.orderData.ListOrderGroups(ctx, orderID)
	if err != nil {
		return ""
	}
	for _, group := range groups {
		if group.IsDefault {
			return group.ID
		}
	}
	if len(groups) > 0 {
		return groups[0].ID
	}
	return ""
}
