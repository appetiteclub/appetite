package order

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/events"
	"github.com/appetiteclub/apt/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const MaxBodyBytes = 1 << 20

type Handler struct {
	logger         apt.Logger
	config         *apt.Config
	tlm            *telemetry.HTTP
	orderRepo      OrderRepo
	orderItemRepo  OrderItemRepo
	orderGroupRepo OrderGroupRepo
	tableClient    *apt.ServiceClient
	tableStates    *TableStateCache
	kitchenClient  *apt.ServiceClient
	publisher      events.Publisher
	streamServer   *OrderEventStreamServer
}

type HandlerDeps struct {
	Repos             Repos
	TableStatesCache  *TableStateCache
	KitchenClient     *apt.ServiceClient
	Publisher         events.Publisher
	OrderStreamServer *OrderEventStreamServer
}

type Repos struct {
	OrderRepo      OrderRepo
	OrderItemRepo  OrderItemRepo
	OrderGroupRepo OrderGroupRepo
}

func NewHandler(hd HandlerDeps, config *apt.Config, logger apt.Logger) *Handler {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}

	// Initialize table service client for querying table state
	tableURL, _ := config.GetString("services.table.url")
	tableClient := apt.NewServiceClient(tableURL)

	return &Handler{
		config:         config,
		logger:         logger,
		tlm:            telemetry.NewHTTP(),
		orderRepo:      hd.Repos.OrderRepo,
		orderItemRepo:  hd.Repos.OrderItemRepo,
		orderGroupRepo: hd.Repos.OrderGroupRepo,
		tableClient:    tableClient,
		tableStates:    hd.TableStatesCache,
		kitchenClient:  hd.KitchenClient,
		publisher:      hd.Publisher,
		streamServer:   hd.OrderStreamServer,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/orders", func(r chi.Router) {
		r.Post("/", h.CreateOrder)
		r.Get("/", h.ListOrders)
		r.Get("/{id}", h.GetOrder)
		r.Put("/{id}", h.UpdateOrderStatus)
		r.Delete("/{id}", h.DeleteOrder)
		r.Post("/{id}/close", h.CloseOrder)

		r.Route("/{orderID}/items", func(r chi.Router) {
			r.Post("/", h.CreateOrderItem)
			r.Get("/", h.ListOrderItems)
		})

		r.Route("/{orderID}/groups", func(r chi.Router) {
			r.Post("/", h.CreateOrderGroup)
			r.Get("/", h.ListOrderGroups)
		})
	})

	r.Route("/order-items", func(r chi.Router) {
		r.Get("/{id}", h.GetOrderItem)
		r.Put("/{id}", h.UpdateOrderItem)
		r.Delete("/{id}", h.DeleteOrderItem)
	})

	r.Route("/items", func(r chi.Router) {
		r.Patch("/{id}/deliver", h.MarkItemDelivered)
		r.Patch("/{id}/cancel", h.CancelItem)
	})
}

// Order Handlers

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateOrder")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	req, ok := h.decodeOrderCreatePayload(w, r, log)
	if !ok {
		return
	}

	if req.TableID == uuid.Nil {
		log.Debug("missing table id in create order request")
		apt.RespondError(w, http.StatusBadRequest, "table_id is required")
		return
	}

	status, err := h.ensureTableAllowsOrdering(r.Context(), req.TableID)
	if err != nil {
		log.Info("table cannot accept orders", "table_id", req.TableID.String(), "status", status, "error", err)
		h.publishOrderTableRejection(r.Context(), req.TableID, nil, "create_order", err.Error(), status)
		apt.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	order := NewOrder()
	order.TableID = req.TableID
	order.Status = "pending"
	order.BeforeCreate()

	if err := h.orderRepo.Create(ctx, order); err != nil {
		log.Error("cannot create order", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not create order")
		return
	}

	defaultGroup := NewOrderGroup(order.ID, "Main")
	defaultGroup.MarkDefault()
	if err := h.orderGroupRepo.Create(ctx, defaultGroup); err != nil {
		log.Error("cannot create default order group", "error", err)
		// best effort: do not fail the whole request
	}

	links := apt.RESTfulLinksFor(order)
	w.WriteHeader(http.StatusCreated)
	apt.RespondSuccess(w, order, links...)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetOrder")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	order, err := h.orderRepo.Get(ctx, id)
	if err != nil {
		log.Error("error loading order", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusNotFound, "Order not found")
		return
	}

	if order == nil {
		apt.RespondError(w, http.StatusNotFound, "Order not found")
		return
	}

	links := apt.RESTfulLinksFor(order)
	apt.RespondSuccess(w, order, links...)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListOrders")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	// Support filtering by table_id and status via query params
	tableIDStr := r.URL.Query().Get("table_id")
	status := r.URL.Query().Get("status")

	var orders []*Order
	var err error

	if tableIDStr != "" {
		tableID, parseErr := uuid.Parse(tableIDStr)
		if parseErr != nil {
			log.Debug("invalid table_id parameter", "table_id", tableIDStr)
			apt.RespondError(w, http.StatusBadRequest, "Invalid table_id parameter")
			return
		}
		orders, err = h.orderRepo.ListByTable(ctx, tableID)
	} else if status != "" {
		orders, err = h.orderRepo.ListByStatus(ctx, status)
	} else {
		orders, err = h.orderRepo.List(ctx)
	}

	if err != nil {
		log.Error("error retrieving orders", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not retrieve orders")
		return
	}

	apt.RespondCollection(w, orders, "order")
}

func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.UpdateOrderStatus")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	order, err := h.orderRepo.Get(ctx, id)
	if err != nil || order == nil {
		log.Error("order not found", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusNotFound, "Order not found")
		return
	}

	req, ok := h.decodeOrderUpdatePayload(w, r, log)
	if !ok {
		return
	}

	// Update status based on request
	switch req.Status {
	case "preparing":
		order.MarkAsPreparing()
	case "ready":
		order.MarkAsReady()
	case "delivered":
		order.MarkAsDelivered()
	case "cancelled":
		order.Cancel()
	default:
		log.Debug("invalid status", "status", req.Status)
		apt.RespondError(w, http.StatusBadRequest, "Invalid status")
		return
	}

	if err := h.orderRepo.Save(ctx, order); err != nil {
		log.Error("cannot update order", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not update order")
		return
	}

	links := apt.RESTfulLinksFor(order)
	apt.RespondSuccess(w, order, links...)
}

func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeleteOrder")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	if err := h.orderRepo.Delete(ctx, id); err != nil {
		log.Error("cannot delete order", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not delete order")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// OrderItem Handlers

func (h *Handler) CreateOrderItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateOrderItem")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		log.Debug("invalid order ID", "orderID", orderIDStr)
		apt.RespondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	req, ok := h.decodeOrderItemCreatePayload(w, r, log)
	if !ok {
		return
	}

	parentOrder, err := h.orderRepo.Get(ctx, orderID)
	if err != nil || parentOrder == nil {
		log.Error("order not found for item create", "error", err, "order_id", orderID.String())
		apt.RespondError(w, http.StatusNotFound, "Order not found")
		return
	}

	status, guardErr := h.ensureTableAllowsOrdering(ctx, parentOrder.TableID)
	if guardErr != nil {
		log.Info("table cannot accept order items", "table_id", parentOrder.TableID.String(), "status", status, "error", guardErr)
		h.publishOrderTableRejection(ctx, parentOrder.TableID, &parentOrder.ID, "add_item", guardErr.Error(), status)
		apt.RespondError(w, http.StatusBadRequest, guardErr.Error())
		return
	}

	item := NewOrderItem()
	item.OrderID = orderID
	item.GroupID = req.GroupID
	item.DishName = req.DishName
	item.Category = req.Category
	item.Quantity = req.Quantity
	item.Price = req.Price
	item.Notes = req.Notes
	item.MenuItemID = req.MenuItemID
	item.ProductionStation = req.ProductionStation
	item.RequiresProduction = req.RequiresProduction

	// Direct service items (no production required) start as ready for immediate delivery
	// NOTE: Future enhancement may involve stock service integration for availability checks
	if !req.RequiresProduction {
		item.Status = "ready"
	}

	item.BeforeCreate()

	if err := h.orderItemRepo.Create(ctx, item); err != nil {
		log.Error("cannot create order item", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not create order item")
		return
	}

	// Publish event to NATS if item requires production
	if item.RequiresProduction && h.publisher != nil {
		h.publishOrderItemCreated(ctx, item, parentOrder)
	}

	links := apt.RESTfulLinksFor(item)
	w.WriteHeader(http.StatusCreated)
	apt.RespondSuccess(w, item, links...)
}

func (h *Handler) GetOrderItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetOrderItem")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	item, err := h.orderItemRepo.Get(ctx, id)
	if err != nil {
		log.Error("error loading order item", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusNotFound, "Order item not found")
		return
	}

	if item == nil {
		apt.RespondError(w, http.StatusNotFound, "Order item not found")
		return
	}

	links := apt.RESTfulLinksFor(item)
	apt.RespondSuccess(w, item, links...)
}

func (h *Handler) ListOrderItems(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListOrderItems")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		log.Debug("invalid order ID", "orderID", orderIDStr)
		apt.RespondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	items, err := h.orderItemRepo.ListByOrder(ctx, orderID)
	if err != nil {
		log.Error("error retrieving order items", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not retrieve order items")
		return
	}

	apt.RespondCollection(w, items, "order-item")
}

func (h *Handler) UpdateOrderItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.UpdateOrderItem")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	item, err := h.orderItemRepo.Get(ctx, id)
	if err != nil || item == nil {
		log.Error("order item not found", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusNotFound, "Order item not found")
		return
	}

	req, ok := h.decodeOrderItemUpdatePayload(w, r, log)
	if !ok {
		return
	}

	// Update fields
	if req.Quantity != nil {
		item.Quantity = *req.Quantity
	}
	if req.Notes != nil {
		item.Notes = *req.Notes
	}
	if req.Status != nil {
		switch *req.Status {
		case "preparing":
			item.MarkAsPreparing()
		case "ready":
			item.MarkAsReady()
		case "delivered":
			item.MarkAsDelivered()
		case "cancelled":
			item.Cancel()
		default:
			log.Debug("invalid status", "status", *req.Status)
			apt.RespondError(w, http.StatusBadRequest, "Invalid status")
			return
		}
	} else {
		item.BeforeUpdate()
	}

	if err := h.orderItemRepo.Save(ctx, item); err != nil {
		log.Error("cannot update order item", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not update order item")
		return
	}

	links := apt.RESTfulLinksFor(item)
	apt.RespondSuccess(w, item, links...)
}

func (h *Handler) DeleteOrderItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeleteOrderItem")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	if err := h.orderItemRepo.Delete(ctx, id); err != nil {
		log.Error("cannot delete order item", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not delete order item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateOrderGroup(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateOrderGroup")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		log.Debug("invalid order ID", "order_id", orderIDStr)
		apt.RespondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	orderEntity, err := h.orderRepo.Get(ctx, orderID)
	if err != nil || orderEntity == nil {
		log.Debug("order not found for group create", "order_id", orderID.String())
		apt.RespondError(w, http.StatusNotFound, "Order not found")
		return
	}

	req, ok := h.decodeOrderGroupCreatePayload(w, r, log)
	if !ok {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		apt.RespondError(w, http.StatusBadRequest, "name is required")
		return
	}

	group := NewOrderGroup(orderID, name)
	if err := h.orderGroupRepo.Create(ctx, group); err != nil {
		log.Error("cannot create order group", "error", err, "order_id", orderID.String())
		apt.RespondError(w, http.StatusInternalServerError, "Could not create order group")
		return
	}

	links := apt.RESTfulLinksFor(group)
	w.WriteHeader(http.StatusCreated)
	apt.RespondSuccess(w, group, links...)
}

func (h *Handler) ListOrderGroups(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListOrderGroups")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	orderIDStr := chi.URLParam(r, "orderID")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		log.Debug("invalid order ID", "order_id", orderIDStr)
		apt.RespondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	groups, err := h.orderGroupRepo.ListByOrder(ctx, orderID)
	if err != nil {
		log.Error("cannot list order groups", "error", err, "order_id", orderID.String())
		apt.RespondError(w, http.StatusInternalServerError, "Could not retrieve order groups")
		return
	}

	apt.RespondCollection(w, groups, "order_group")
}

// Helper methods
func (h *Handler) parseIDParam(w http.ResponseWriter, r *http.Request, log apt.Logger) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		log.Debug("missing id parameter")
		apt.RespondError(w, http.StatusBadRequest, "Missing id parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Debug("invalid id parameter", "id", idStr)
		apt.RespondError(w, http.StatusBadRequest, "Invalid id parameter")
		return uuid.Nil, false
	}

	return id, true
}

// Payload decoders

type OrderCreateRequest struct {
	TableID uuid.UUID `json:"table_id"`
}

type OrderUpdateRequest struct {
	Status string `json:"status"`
}

type OrderItemCreateRequest struct {
	GroupID            *uuid.UUID `json:"group_id,omitempty"`
	DishName           string     `json:"dish_name"`
	Category           string     `json:"category"`
	Quantity           int        `json:"quantity"`
	Price              float64    `json:"price"`
	Notes              string     `json:"notes,omitempty"`
	MenuItemID         *uuid.UUID `json:"menu_item_id,omitempty"`
	ProductionStation  *string    `json:"production_station,omitempty"`
	RequiresProduction bool       `json:"requires_production"`
}

type OrderItemUpdateRequest struct {
	Quantity *int    `json:"quantity,omitempty"`
	Status   *string `json:"status,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

type OrderGroupCreateRequest struct {
	Name string `json:"name"`
}

func (h *Handler) decodeOrderCreatePayload(w http.ResponseWriter, r *http.Request, log apt.Logger) (OrderCreateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("failed to read request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Failed to read request body")
		return OrderCreateRequest{}, false
	}

	var req OrderCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("failed to decode request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON in request body")
		return OrderCreateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeOrderUpdatePayload(w http.ResponseWriter, r *http.Request, log apt.Logger) (OrderUpdateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("failed to read request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Failed to read request body")
		return OrderUpdateRequest{}, false
	}

	var req OrderUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("failed to decode request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON in request body")
		return OrderUpdateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeOrderItemCreatePayload(w http.ResponseWriter, r *http.Request, log apt.Logger) (OrderItemCreateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("failed to read request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Failed to read request body")
		return OrderItemCreateRequest{}, false
	}

	var req OrderItemCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("failed to decode request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON in request body")
		return OrderItemCreateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeOrderItemUpdatePayload(w http.ResponseWriter, r *http.Request, log apt.Logger) (OrderItemUpdateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("failed to read request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Failed to read request body")
		return OrderItemUpdateRequest{}, false
	}

	var req OrderItemUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("failed to decode request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON in request body")
		return OrderItemUpdateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeOrderGroupCreatePayload(w http.ResponseWriter, r *http.Request, log apt.Logger) (OrderGroupCreateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("failed to read request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Failed to read request body")
		return OrderGroupCreateRequest{}, false
	}

	var req OrderGroupCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("failed to decode request body", "error", err)
		apt.RespondError(w, http.StatusBadRequest, "Invalid JSON in request body")
		return OrderGroupCreateRequest{}, false
	}

	return req, true
}

func (h *Handler) ensureTableAllowsOrdering(ctx context.Context, tableID uuid.UUID) (string, error) {
	if tableID == uuid.Nil {
		return "", fmt.Errorf("table_id is required")
	}
	if h.tableStates == nil {
		return "", nil
	}
	status, err := h.tableStates.Ensure(ctx, tableID)
	if err != nil {
		return status, err
	}
	if status == "" {
		return status, fmt.Errorf("table status unavailable")
	}
	switch status {
	case "available", "open", "reserved":
		return status, nil
	default:
		return status, fmt.Errorf("table is %s", status)
	}
}

func (h *Handler) publishOrderTableRejection(ctx context.Context, tableID uuid.UUID, orderID *uuid.UUID, action, reason, status string) {
	if h.publisher == nil {
		return
	}
	event := pkg.OrderTableRejectionEvent{
		EventType:  pkg.EventOrderTableRejected,
		TableID:    tableID.String(),
		Action:     action,
		Reason:     reason,
		Status:     status,
		OccurredAt: time.Now().UTC(),
	}
	if orderID != nil {
		event.OrderID = orderID.String()
	}
	payload, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("cannot marshal order table rejection", "error", err, "table_id", tableID.String())
		return
	}
	if err := h.publisher.Publish(ctx, pkg.OrderTableTopic, payload); err != nil {
		h.logger.Error("cannot publish order table rejection", "error", err, "table_id", tableID.String())
	}
}

func (h *Handler) fetchTableInfo(ctx context.Context, tableID uuid.UUID) (*TableInfo, error) {
	if h.tableClient == nil {
		return nil, fmt.Errorf("table client not available")
	}

	resp, err := h.tableClient.Get(ctx, "tables", tableID.String())
	if err != nil {
		return nil, err
	}

	var table TableInfo
	if err := decodeSuccessResponse(resp, &table); err != nil {
		return nil, err
	}

	return &table, nil
}

type TableInfo struct {
	Number string `json:"number"`
}

func decodeSuccessResponse(resp *apt.SuccessResponse, target interface{}) error {
	if resp == nil {
		return fmt.Errorf("nil success response")
	}

	raw, err := json.Marshal(resp.Data)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}

	return nil
}

func (h *Handler) publishOrderItemCreated(ctx context.Context, item *OrderItem, parentOrder *Order) {
	if h.publisher == nil {
		return
	}

	// Get table number for enrichment
	tableNumber := ""
	if h.tableClient != nil {
		table, err := h.fetchTableInfo(ctx, parentOrder.TableID)
		if err == nil && table != nil {
			tableNumber = table.Number
		}
	}

	// Get station name for enrichment (if production_station is set)
	stationName := ""
	if item.ProductionStation != nil {
		// TODO: Fetch from Dictionary service when available
		// For now, station name will be empty, Operations can fetch it
	}

	evt := event.OrderItemEvent{
		EventType:          event.EventOrderItemCreated,
		OccurredAt:         time.Now().UTC(),
		OrderID:            item.OrderID.String(),
		OrderItemID:        item.ID.String(),
		Quantity:           item.Quantity,
		Notes:              item.Notes,
		RequiresProduction: item.RequiresProduction,
		MenuItemName:       item.DishName, // Use DishName as menu_item_name
		TableNumber:        tableNumber,
		StationName:        stationName,
	}

	if item.MenuItemID != nil {
		evt.MenuItemID = item.MenuItemID.String()
	}
	if item.ProductionStation != nil {
		evt.ProductionStation = *item.ProductionStation
	}
	if parentOrder != nil {
		evt.TableID = parentOrder.TableID.String()
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		h.logger.Error("cannot marshal order item created event", "error", err)
		return
	}
	if err := h.publisher.Publish(ctx, event.OrderItemsTopic, payload); err != nil {
		h.logger.Error("cannot publish order item created event", "error", err)
	} else {
		h.logger.Info("published order item created event", "order_item_id", item.ID.String())
	}
}

// MarkItemDelivered marks an order item as delivered
func (h *Handler) MarkItemDelivered(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.MarkItemDelivered")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	item, err := h.orderItemRepo.Get(ctx, itemID)
	if err != nil {
		log.Error("cannot get order item", "error", err)
		apt.RespondError(w, http.StatusNotFound, "Item not found")
		return
	}

	previousStatus := item.Status
	item.MarkAsDelivered()

	if err := h.orderItemRepo.Save(ctx, item); err != nil {
		log.Error("cannot update order item", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not mark item as delivered")
		return
	}

	// Broadcast the status change to gRPC stream subscribers (operations service, etc.)
	if h.streamServer != nil {
		h.streamServer.BroadcastOrderItemEvent(item, "order.item.status_changed", previousStatus)
	}

	// Publish NATS event for kitchen service to update ticket status
	if item.RequiresProduction {
		h.publishOrderItemStatusChange(ctx, item, previousStatus)
	}

	log.Info("order item marked as delivered", "item_id", itemID)
	apt.Respond(w, http.StatusOK, item, nil)
}

func (h *Handler) publishOrderItemStatusChange(ctx context.Context, item *OrderItem, previousStatus string) {
	evt := event.OrderItemEvent{
		EventType:          "order.item.status_changed",
		OccurredAt:         time.Now().UTC(),
		OrderID:            item.OrderID.String(),
		OrderItemID:        item.ID.String(),
		Status:             item.Status,
		PreviousStatus:     previousStatus,
		RequiresProduction: item.RequiresProduction,
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		h.logger.Error("cannot marshal order item status change event", "error", err)
		return
	}

	if err := h.publisher.Publish(ctx, event.OrderItemsTopic, payload); err != nil {
		h.logger.Error("cannot publish order item status change event", "error", err)
	} else {
		h.logger.Info("published order item status change event", "order_item_id", item.ID.String(), "status", item.Status)
	}
}

// CancelItem cancels an order item
func (h *Handler) CancelItem(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CancelItem")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	itemID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apt.RespondError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	item, err := h.orderItemRepo.Get(ctx, itemID)
	if err != nil {
		log.Error("cannot get order item", "error", err)
		apt.RespondError(w, http.StatusNotFound, "Item not found")
		return
	}

	item.Cancel()

	if err := h.orderItemRepo.Save(ctx, item); err != nil {
		log.Error("cannot cancel order item", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not cancel item")
		return
	}

	log.Info("order item cancelled", "item_id", itemID)
	apt.Respond(w, http.StatusOK, item, nil)
}

// updateKitchenTicketStatus updates the kitchen ticket status via Kitchen service
// This is called when the waiter manually marks an item as delivered
func (h *Handler) updateKitchenTicketStatus(ctx context.Context, orderItemID uuid.UUID, statusID string, log apt.Logger) {
	if h.kitchenClient == nil {
		return
	}

	// Find ticket by order_item_id query
	path := fmt.Sprintf("/tickets?order_item_id=%s", orderItemID.String())
	resp, err := h.kitchenClient.Request(ctx, "GET", path, nil)
	if err != nil {
		log.Info("cannot find kitchen ticket for order item", "order_item_id", orderItemID, "error", err)
		return
	}

	// Parse tickets response from Data field
	if data, ok := resp.Data.(map[string]interface{}); ok {
		if tickets, ok := data["tickets"].([]interface{}); ok && len(tickets) > 0 {
			if ticket, ok := tickets[0].(map[string]interface{}); ok {
				if ticketID, ok := ticket["id"].(string); ok {
					// Update ticket status
					updatePath := fmt.Sprintf("/tickets/%s/status", ticketID)
					body := map[string]string{"status_id": statusID}
					_, err := h.kitchenClient.Request(ctx, "PATCH", updatePath, body)
					if err != nil {
						log.Info("cannot update kitchen ticket status", "ticket_id", ticketID, "error", err)
					} else {
						log.Info("kitchen ticket status updated", "ticket_id", ticketID, "status_id", statusID)
					}
				}
			}
		}
	}
}

// CloseOrder closes an order, optionally auto-processing pending/ready/preparing items
// Query params:
//   - force=true: auto-process pending (cancel) and ready (deliver) items
//   - takeaway=true: treat preparing items as takeaway (mark delivered, table goes to clearing)
func (h *Handler) CloseOrder(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CloseOrder")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	order, err := h.orderRepo.Get(ctx, id)
	if err != nil || order == nil {
		log.Error("order not found", "error", err, "id", id.String())
		apt.RespondError(w, http.StatusNotFound, "Order not found")
		return
	}

	if order.Status == "closed" {
		apt.RespondError(w, http.StatusBadRequest, "Order is already closed")
		return
	}

	// Get all items for this order
	items, err := h.orderItemRepo.ListByOrder(ctx, id)
	if err != nil {
		log.Error("cannot list order items", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not retrieve order items")
		return
	}

	// Check for flags
	force := r.URL.Query().Get("force") == "true"
	takeaway := r.URL.Query().Get("takeaway") == "true"

	// Categorize items by status
	var pendingItems, readyItems, preparingItems []*OrderItem
	for _, item := range items {
		switch item.Status {
		case "pending":
			pendingItems = append(pendingItems, item)
		case "ready":
			readyItems = append(readyItems, item)
		case "preparing", "started":
			preparingItems = append(preparingItems, item)
		// delivered, cancelled are ok
		}
	}

	// If there are items being prepared and no takeaway flag, check what to do
	if len(preparingItems) > 0 && !takeaway {
		// Return info for confirmation with preparing items
		response := map[string]interface{}{
			"requires_confirmation": true,
			"pending_count":         len(pendingItems),
			"ready_count":           len(readyItems),
			"preparing_count":       len(preparingItems),
			"message":               fmt.Sprintf("Order has %d preparing, %d pending and %d ready items", len(preparingItems), len(pendingItems), len(readyItems)),
		}
		apt.Respond(w, http.StatusOK, response, nil)
		return
	}

	// If not forcing and there are pending/ready items (but no preparing), return info for confirmation
	if !force && (len(pendingItems) > 0 || len(readyItems) > 0) {
		response := map[string]interface{}{
			"requires_confirmation": true,
			"pending_count":         len(pendingItems),
			"ready_count":           len(readyItems),
			"preparing_count":       0,
			"message":               fmt.Sprintf("Order has %d pending and %d ready items", len(pendingItems), len(readyItems)),
		}
		apt.Respond(w, http.StatusOK, response, nil)
		return
	}

	// Auto-cancel pending items
	for _, item := range pendingItems {
		item.Cancel()
		if err := h.orderItemRepo.Save(ctx, item); err != nil {
			log.Error("cannot cancel pending item", "error", err, "item_id", item.ID)
		}
	}

	// Auto-deliver ready items
	for _, item := range readyItems {
		item.MarkAsDelivered()
		if err := h.orderItemRepo.Save(ctx, item); err != nil {
			log.Error("cannot deliver ready item", "error", err, "item_id", item.ID)
		}
	}

	// Handle preparing items as takeaway (mark as delivered)
	hasTakeaway := false
	if takeaway && len(preparingItems) > 0 {
		hasTakeaway = true
		for _, item := range preparingItems {
			item.MarkAsDelivered()
			if err := h.orderItemRepo.Save(ctx, item); err != nil {
				log.Error("cannot mark preparing item as delivered (takeaway)", "error", err, "item_id", item.ID)
			}
		}
	}

	// Close the order
	order.Close()
	if err := h.orderRepo.Save(ctx, order); err != nil {
		log.Error("cannot close order", "error", err)
		apt.RespondError(w, http.StatusInternalServerError, "Could not close order")
		return
	}

	log.Info("order closed", "order_id", id, "cancelled_items", len(pendingItems), "delivered_items", len(readyItems), "takeaway_items", len(preparingItems))

	response := map[string]interface{}{
		"order":           order,
		"cancelled_count": len(pendingItems),
		"delivered_count": len(readyItems),
		"takeaway_count":  len(preparingItems),
		"has_takeaway":    hasTakeaway,
		"table_id":        order.TableID,
	}
	apt.Respond(w, http.StatusOK, response, nil)
}

func (h *Handler) log(r *http.Request) apt.Logger {
	return h.logger.With("request_id", r.Context().Value("request_id"))
}
