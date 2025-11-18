package tables

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/aquamarinepk/aqm/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const MaxBodyBytes = 1 << 20

type Handler struct {
	tableRepo       TableRepo
	groupRepo       GroupRepo
	orderRepo       OrderRepo
	orderItemRepo   OrderItemRepo
	reservationRepo ReservationRepo
	logger          aqm.Logger
	config          *aqm.Config
	tlm             *telemetry.HTTP
	publisher       events.Publisher
}

const tableEventSource = "table-service"

func NewHandler(
	tableRepo TableRepo,
	groupRepo GroupRepo,
	orderRepo OrderRepo,
	orderItemRepo OrderItemRepo,
	reservationRepo ReservationRepo,
	logger aqm.Logger,
	config *aqm.Config,
	publisher events.Publisher,
) *Handler {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &Handler{
		tableRepo:       tableRepo,
		groupRepo:       groupRepo,
		orderRepo:       orderRepo,
		orderItemRepo:   orderItemRepo,
		reservationRepo: reservationRepo,
		logger:          logger,
		config:          config,
		tlm:             telemetry.NewHTTP(),
		publisher:       publisher,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/tables", func(r chi.Router) {
		r.Post("/", h.CreateTable)
		r.Get("/", h.ListTables)
		r.Get("/{id}", h.GetTable)
		r.Patch("/{id}", h.UpdateTable)
		r.Delete("/{id}", h.DeleteTable)

		r.Post("/{id}/open", h.OpenTable)
		r.Post("/{id}/close", h.CloseTable)

		r.Route("/{tableID}/groups", func(r chi.Router) {
			r.Post("/", h.CreateGroup)
			r.Get("/", h.ListGroups)
		})

		r.Route("/{tableID}/orders", func(r chi.Router) {
			r.Post("/", h.CreateOrder)
			r.Get("/", h.ListOrders)
		})
	})

	r.Route("/orders", func(r chi.Router) {
		r.Get("/{id}", h.GetOrder)
		r.Delete("/{id}", h.DeleteOrder)

		r.Route("/{orderID}/items", func(r chi.Router) {
			r.Post("/", h.CreateOrderItem)
			r.Get("/", h.ListOrderItems)
		})
	})

	r.Route("/order-items", func(r chi.Router) {
		r.Get("/{id}", h.GetOrderItem)
		r.Patch("/{id}", h.UpdateOrderItem)
		r.Delete("/{id}", h.DeleteOrderItem)
	})

	r.Route("/groups", func(r chi.Router) {
		r.Delete("/{id}", h.DeleteGroup)
	})

	r.Route("/reservations", func(r chi.Router) {
		r.Post("/", h.CreateReservation)
		r.Get("/", h.ListReservations)
		r.Get("/{id}", h.GetReservation)
		r.Patch("/{id}", h.UpdateReservation)
		r.Delete("/{id}", h.DeleteReservation)
	})
}

// Table Handlers

func (h *Handler) CreateTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateTable")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	req, ok := h.decodeTableCreatePayload(w, r, log)
	if !ok {
		return
	}

	validationErrors := ValidateTableCreate(ctx, req)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		aqm.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	table := NewTable()
	table.Number = req.Number
	table.GuestCount = req.GuestCount
	table.AssignedTo = req.AssignedTo
	table.BeforeCreate()

	if err := h.tableRepo.Create(ctx, table); err != nil {
		log.Error("cannot create table", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not create table")
		return
	}

	h.publishTableStatusChanged(ctx, table, "", "table.created")

	links := aqm.RESTfulLinksFor(table)
	w.WriteHeader(http.StatusCreated)
	aqm.RespondSuccess(w, table, links...)
}

func (h *Handler) GetTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetTable")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	table, err := h.tableRepo.Get(ctx, id)
	if err != nil {
		log.Error("error loading table", "error", err, "id", id.String())
		aqm.RespondError(w, http.StatusNotFound, "Table not found")
		return
	}

	if table == nil {
		aqm.RespondError(w, http.StatusNotFound, "Table not found")
		return
	}

	links := aqm.RESTfulLinksFor(table)
	aqm.RespondSuccess(w, table, links...)
}

func (h *Handler) ListTables(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListTables")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	status := r.URL.Query().Get("status")

	var tables []*Table
	var err error

	if status != "" {
		tables, err = h.tableRepo.ListByStatus(ctx, status)
	} else {
		tables, err = h.tableRepo.List(ctx)
	}

	if err != nil {
		log.Error("error retrieving tables", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not retrieve tables")
		return
	}

	aqm.RespondCollection(w, tables, "table")
}

func (h *Handler) UpdateTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.UpdateTable")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	req, ok := h.decodeTableUpdatePayload(w, r, log)
	if !ok {
		return
	}

	validationErrors := ValidateTableUpdate(ctx, id, req)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		aqm.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	table, err := h.tableRepo.Get(ctx, id)
	if err != nil || table == nil {
		log.Error("table not found", "error", err, "id", id.String())
		aqm.RespondError(w, http.StatusNotFound, "Table not found")
		return
	}

	previousStatus := table.Status
	statusChanged := false

	if req.Number != "" {
		table.Number = req.Number
	}
	if req.Status != "" {
		if table.Status != req.Status {
			statusChanged = true
		}
		table.Status = req.Status
	}
	if req.GuestCount > 0 {
		table.GuestCount = req.GuestCount
	}
	if req.AssignedTo != nil {
		table.AssignedTo = req.AssignedTo
	}

	table.BeforeUpdate()

	if err := h.tableRepo.Save(ctx, table); err != nil {
		log.Error("cannot update table", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update table")
		return
	}

	if statusChanged {
		h.publishTableStatusChanged(ctx, table, previousStatus, "table.updated")
	}

	links := aqm.RESTfulLinksFor(table)
	aqm.RespondSuccess(w, table, links...)
}

func (h *Handler) DeleteTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeleteTable")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	if err := h.tableRepo.Delete(ctx, id); err != nil {
		log.Error("cannot delete table", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not delete table")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) OpenTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.OpenTable")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	req, ok := h.decodeTableOpenPayload(w, r, log)
	if !ok {
		return
	}

	table, err := h.tableRepo.Get(ctx, id)
	if err != nil || table == nil {
		log.Error("table not found", "error", err, "id", id.String())
		aqm.RespondError(w, http.StatusNotFound, "Table not found")
		return
	}

	previousStatus := table.Status
	table.Open(req.GuestCount, req.AssignedTo)

	if err := h.tableRepo.Save(ctx, table); err != nil {
		log.Error("cannot open table", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not open table")
		return
	}

	h.publishTableStatusChanged(ctx, table, previousStatus, "table.opened")

	links := aqm.RESTfulLinksFor(table)
	aqm.RespondSuccess(w, table, links...)
}

func (h *Handler) CloseTable(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CloseTable")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	table, err := h.tableRepo.Get(ctx, id)
	if err != nil || table == nil {
		log.Error("table not found", "error", err, "id", id.String())
		aqm.RespondError(w, http.StatusNotFound, "Table not found")
		return
	}

	previousStatus := table.Status
	table.Close()

	if err := h.tableRepo.Save(ctx, table); err != nil {
		log.Error("cannot close table", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not close table")
		return
	}

	h.publishTableStatusChanged(ctx, table, previousStatus, "table.closed")

	links := aqm.RESTfulLinksFor(table)
	aqm.RespondSuccess(w, table, links...)
}

func (h *Handler) publishTableStatusChanged(ctx context.Context, table *Table, previousStatus, reason string) {
	if h.publisher == nil || table == nil {
		return
	}

	event := pkg.TableStatusEvent{
		EventType:      pkg.EventTableStatusChanged,
		TableID:        table.ID.String(),
		Status:         table.Status,
		PreviousStatus: previousStatus,
		Reason:         reason,
		Source:         tableEventSource,
		OccurredAt:     time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("cannot marshal table status event", "error", err, "table_id", table.ID.String())
		return
	}

	if err := h.publisher.Publish(ctx, pkg.TableStatusTopic, payload); err != nil {
		h.logger.Error("cannot publish table status event", "error", err, "table_id", table.ID.String())
	}
}

// Group Handlers

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateGroup")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	req, ok := h.decodeGroupCreatePayload(w, r, log)
	if !ok {
		return
	}

	group := NewGroup()
	group.TableID = req.TableID
	group.Name = req.Name
	group.BeforeCreate()

	if err := h.groupRepo.Create(ctx, group); err != nil {
		log.Error("cannot create group", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not create group")
		return
	}

	links := aqm.RESTfulLinksFor(group)
	w.WriteHeader(http.StatusCreated)
	aqm.RespondSuccess(w, group, links...)
}

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListGroups")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	tableIDStr := chi.URLParam(r, "tableID")
	tableID, err := uuid.Parse(tableIDStr)
	if err != nil {
		log.Debug("invalid table ID", "tableID", tableIDStr)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid table ID")
		return
	}

	groups, err := h.groupRepo.ListByTable(ctx, tableID)
	if err != nil {
		log.Error("error retrieving groups", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not retrieve groups")
		return
	}

	aqm.RespondCollection(w, groups, "group")
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeleteGroup")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	if err := h.groupRepo.Delete(ctx, id); err != nil {
		log.Error("cannot delete group", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not delete group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Order Handlers

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateOrder")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	tableIDStr := chi.URLParam(r, "tableID")
	tableID, err := uuid.Parse(tableIDStr)
	if err != nil {
		log.Debug("invalid table ID", "tableID", tableIDStr)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid table ID")
		return
	}

	order := NewOrder()
	order.TableID = tableID
	order.BeforeCreate()

	if err := h.orderRepo.Create(ctx, order); err != nil {
		log.Error("cannot create order", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not create order")
		return
	}

	links := aqm.RESTfulLinksFor(order)
	w.WriteHeader(http.StatusCreated)
	aqm.RespondSuccess(w, order, links...)
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
		aqm.RespondError(w, http.StatusNotFound, "Order not found")
		return
	}

	if order == nil {
		aqm.RespondError(w, http.StatusNotFound, "Order not found")
		return
	}

	links := aqm.RESTfulLinksFor(order)
	aqm.RespondSuccess(w, order, links...)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListOrders")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	tableIDStr := chi.URLParam(r, "tableID")
	tableID, err := uuid.Parse(tableIDStr)
	if err != nil {
		log.Debug("invalid table ID", "tableID", tableIDStr)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid table ID")
		return
	}

	orders, err := h.orderRepo.ListByTable(ctx, tableID)
	if err != nil {
		log.Error("error retrieving orders", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not retrieve orders")
		return
	}

	aqm.RespondCollection(w, orders, "order")
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not delete order")
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

	req, ok := h.decodeOrderItemCreatePayload(w, r, log)
	if !ok {
		return
	}

	validationErrors := ValidateOrderItemCreate(ctx, req)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		aqm.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	item := NewOrderItem()
	item.OrderID = req.OrderID
	item.GroupID = req.GroupID
	item.DishName = req.DishName
	item.Category = req.Category
	item.Quantity = req.Quantity
	item.Price = req.Price
	item.Notes = req.Notes
	item.BeforeCreate()

	if err := h.orderItemRepo.Create(ctx, item); err != nil {
		log.Error("cannot create order item", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not create order item")
		return
	}

	links := aqm.RESTfulLinksFor(item)
	w.WriteHeader(http.StatusCreated)
	aqm.RespondSuccess(w, item, links...)
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
		aqm.RespondError(w, http.StatusNotFound, "Order item not found")
		return
	}

	if item == nil {
		aqm.RespondError(w, http.StatusNotFound, "Order item not found")
		return
	}

	links := aqm.RESTfulLinksFor(item)
	aqm.RespondSuccess(w, item, links...)
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
		aqm.RespondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	items, err := h.orderItemRepo.ListByOrder(ctx, orderID)
	if err != nil {
		log.Error("error retrieving order items", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not retrieve order items")
		return
	}

	aqm.RespondCollection(w, items, "order-item")
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

	req, ok := h.decodeOrderItemUpdatePayload(w, r, log)
	if !ok {
		return
	}

	item, err := h.orderItemRepo.Get(ctx, id)
	if err != nil || item == nil {
		log.Error("order item not found", "error", err, "id", id.String())
		aqm.RespondError(w, http.StatusNotFound, "Order item not found")
		return
	}

	if req.Status != "" {
		item.Status = req.Status
	}
	if req.Notes != "" {
		item.Notes = req.Notes
	}

	item.BeforeUpdate()

	if err := h.orderItemRepo.Save(ctx, item); err != nil {
		log.Error("cannot update order item", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update order item")
		return
	}

	links := aqm.RESTfulLinksFor(item)
	aqm.RespondSuccess(w, item, links...)
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
		aqm.RespondError(w, http.StatusInternalServerError, "Could not delete order item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Reservation Handlers

func (h *Handler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.CreateReservation")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	req, ok := h.decodeReservationCreatePayload(w, r, log)
	if !ok {
		return
	}

	validationErrors := ValidateReservationCreate(ctx, req)
	if len(validationErrors) > 0 {
		log.Debug("validation failed", "errors", validationErrors)
		aqm.RespondError(w, http.StatusBadRequest, "Validation failed")
		return
	}

	reservation := NewReservation()
	reservation.TableID = req.TableID
	reservation.GuestCount = req.GuestCount
	reservation.ReservedFor = req.ReservedFor
	reservation.ContactName = req.ContactName
	reservation.ContactInfo = req.ContactInfo
	reservation.Notes = req.Notes
	reservation.BeforeCreate()

	if err := h.reservationRepo.Create(ctx, reservation); err != nil {
		log.Error("cannot create reservation", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not create reservation")
		return
	}

	links := aqm.RESTfulLinksFor(reservation)
	w.WriteHeader(http.StatusCreated)
	aqm.RespondSuccess(w, reservation, links...)
}

func (h *Handler) GetReservation(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetReservation")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	reservation, err := h.reservationRepo.Get(ctx, id)
	if err != nil {
		log.Error("error loading reservation", "error", err, "id", id.String())
		aqm.RespondError(w, http.StatusNotFound, "Reservation not found")
		return
	}

	if reservation == nil {
		aqm.RespondError(w, http.StatusNotFound, "Reservation not found")
		return
	}

	links := aqm.RESTfulLinksFor(reservation)
	aqm.RespondSuccess(w, reservation, links...)
}

func (h *Handler) ListReservations(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListReservations")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	date := r.URL.Query().Get("date")

	var reservations []*Reservation
	var err error

	if date != "" {
		reservations, err = h.reservationRepo.ListByDate(ctx, date)
	} else {
		reservations, err = h.reservationRepo.List(ctx)
	}

	if err != nil {
		log.Error("error retrieving reservations", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not retrieve reservations")
		return
	}

	aqm.RespondCollection(w, reservations, "reservation")
}

func (h *Handler) UpdateReservation(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.UpdateReservation")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	req, ok := h.decodeReservationUpdatePayload(w, r, log)
	if !ok {
		return
	}

	reservation, err := h.reservationRepo.Get(ctx, id)
	if err != nil || reservation == nil {
		log.Error("reservation not found", "error", err, "id", id.String())
		aqm.RespondError(w, http.StatusNotFound, "Reservation not found")
		return
	}

	if req.TableID != nil {
		reservation.TableID = req.TableID
	}
	if req.GuestCount > 0 {
		reservation.GuestCount = req.GuestCount
	}
	if req.ReservedFor != nil {
		reservation.ReservedFor = *req.ReservedFor
	}
	if req.ContactName != "" {
		reservation.ContactName = req.ContactName
	}
	if req.ContactInfo != "" {
		reservation.ContactInfo = req.ContactInfo
	}
	if req.Status != "" {
		reservation.Status = req.Status
	}
	if req.Notes != "" {
		reservation.Notes = req.Notes
	}

	reservation.BeforeUpdate()

	if err := h.reservationRepo.Save(ctx, reservation); err != nil {
		log.Error("cannot update reservation", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update reservation")
		return
	}

	links := aqm.RESTfulLinksFor(reservation)
	aqm.RespondSuccess(w, reservation, links...)
}

func (h *Handler) DeleteReservation(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeleteReservation")
	defer finish()

	log := h.log(r)
	ctx := r.Context()

	id, ok := h.parseIDParam(w, r, log)
	if !ok {
		return
	}

	if err := h.reservationRepo.Delete(ctx, id); err != nil {
		log.Error("cannot delete reservation", "error", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not delete reservation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *Handler) log(r *http.Request) aqm.Logger {
	return h.logger.With("request_id", r.Context().Value("request_id"))
}

func (h *Handler) parseIDParam(w http.ResponseWriter, r *http.Request, log aqm.Logger) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		log.Debug("missing id parameter")
		aqm.RespondError(w, http.StatusBadRequest, "Missing id parameter")
		return uuid.Nil, false
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Debug("invalid id parameter", "id", idStr, "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid id parameter")
		return uuid.Nil, false
	}

	return id, true
}

func (h *Handler) decodeTableCreatePayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (TableCreateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return TableCreateRequest{}, false
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		aqm.RespondError(w, http.StatusBadRequest, "Request body is empty")
		return TableCreateRequest{}, false
	}

	var req TableCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return TableCreateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeTableUpdatePayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (TableUpdateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return TableUpdateRequest{}, false
	}

	var req TableUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return TableUpdateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeTableOpenPayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (TableOpenRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return TableOpenRequest{}, false
	}

	var req TableOpenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return TableOpenRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeGroupCreatePayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (GroupCreateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return GroupCreateRequest{}, false
	}

	var req GroupCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return GroupCreateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeOrderItemCreatePayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (OrderItemCreateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return OrderItemCreateRequest{}, false
	}

	var req OrderItemCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return OrderItemCreateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeOrderItemUpdatePayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (OrderItemUpdateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return OrderItemUpdateRequest{}, false
	}

	var req OrderItemUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return OrderItemUpdateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeReservationCreatePayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (ReservationCreateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return ReservationCreateRequest{}, false
	}

	var req ReservationCreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return ReservationCreateRequest{}, false
	}

	return req, true
}

func (h *Handler) decodeReservationUpdatePayload(w http.ResponseWriter, r *http.Request, log aqm.Logger) (ReservationUpdateRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Debug("error reading request body", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return ReservationUpdateRequest{}, false
	}

	var req ReservationUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Debug("error decoding JSON", "error", err)
		aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return ReservationUpdateRequest{}, false
	}

	return req, true
}
