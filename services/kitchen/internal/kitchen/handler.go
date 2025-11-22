package kitchen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/appetiteclub/appetite/pkg/enums/kitchenstatus"
	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/aquamarinepk/aqm/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const MaxBodyBytes = 1 << 20

type Handler struct {
	config    *aqm.Config
	logger    aqm.Logger
	tlm       *telemetry.HTTP
	repo      TicketRepository
	cache     *TicketStateCache
	publisher events.Publisher
}

type HandlerDeps struct {
	Repo      TicketRepository
	Cache     *TicketStateCache
	Publisher events.Publisher
}

func NewHandler(hd HandlerDeps, config *aqm.Config, logger aqm.Logger) *Handler {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &Handler{
		config:    config,
		logger:    logger,
		tlm:       telemetry.NewHTTP(),
		repo:      hd.Repo,
		cache:     hd.Cache,
		publisher: hd.Publisher,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/tickets", func(r chi.Router) {
		r.Get("/", h.ListTickets)
		r.Get("/{id}", h.GetTicket)
		r.Patch("/{id}/status", h.UpdateTicketStatus)
		r.Patch("/{id}/accept", h.AcceptTicket)
		r.Patch("/{id}/start", h.StartTicket)
		r.Patch("/{id}/ready", h.ReadyTicket)
		r.Patch("/{id}/deliver", h.DeliverTicket)
		r.Patch("/{id}/standby", h.StandbyTicket)
		r.Patch("/{id}/block", h.BlockTicket)
		r.Patch("/{id}/reject", h.RejectTicket)
		r.Patch("/{id}/cancel", h.CancelTicket)
	})

	// Internal endpoints for operations/debugging
	r.Route("/internal", func(r chi.Router) {
		r.Post("/reload-cache", h.ReloadCache)
	})
}

func (h *Handler) log(r *http.Request) aqm.Logger {
	return h.logger.With("request_id", aqm.RequestIDFrom(r.Context()))
}

func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ListTickets")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	filter := TicketFilter{}

	station := r.URL.Query().Get("station")
	if station != "" {
		filter.Station = &station
	}

	status := r.URL.Query().Get("status")
	if status != "" {
		filter.Status = &status
	}

	orderIDStr := r.URL.Query().Get("order_id")
	if orderIDStr != "" {
		orderID, err := uuid.Parse(orderIDStr)
		if err != nil {
			aqm.RespondError(w, http.StatusBadRequest, "Invalid order ID")
			return
		}

		filter.OrderID = &orderID
	}

	if orderItemIDStr := r.URL.Query().Get("order_item_id"); orderItemIDStr != "" {
		orderItemID, err := uuid.Parse(orderItemIDStr)
		if err != nil {
			aqm.RespondError(w, http.StatusBadRequest, "Invalid order item ID")
			return
		}

		filter.OrderItemID = &orderItemID
	}

	var tickets []*Ticket

	// Use cache for simple queries (station, status, or all)
	// Fall back to repo for complex filters (order_id, order_item_id)
	if h.cache != nil && filter.OrderID == nil && filter.OrderItemID == nil {
		// Fast path: read from cache
		if filter.Station != nil && filter.Status != nil {
			tickets = h.cache.GetByStationAndStatusCode(*filter.Station, *filter.Status)
		} else if filter.Station != nil {
			tickets = h.cache.GetByStationCode(*filter.Station)
		} else if filter.Status != nil {
			tickets = h.cache.GetByStatusCode(*filter.Status)
		} else {
			tickets = h.cache.GetAll()
		}
	} else {
		// Slow path: query MongoDB for complex filters
		repoTickets, err := h.repo.List(ctx, filter)
		if err != nil {
			log.Errorf("cannot list tickets: %v", err)
			aqm.RespondError(w, http.StatusInternalServerError, "Could not list tickets")
			return
		}
		// Convert []Ticket to []*Ticket
		tickets = make([]*Ticket, len(repoTickets))
		for i := range repoTickets {
			tickets[i] = &repoTickets[i]
		}
	}

	aqm.Respond(w, http.StatusOK, map[string]interface{}{
		"tickets": tickets,
	}, nil)
}

func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.GetTicket")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid ticket ID")
		return
	}

	ticket, err := h.repo.FindByID(ctx, id)
	if err != nil {
		log.Errorf("cannot find ticket: %v", err)
		aqm.RespondError(w, http.StatusNotFound, "Ticket not found")
		return
	}

	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) AcceptTicket(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "accept", kitchenstatus.Statuses.Accepted.Code())
}

func (h *Handler) StartTicket(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.StartTicket")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid ticket ID")
		return
	}

	ticket, err := h.repo.FindByID(ctx, id)
	if err != nil {
		log.Errorf("cannot find ticket: %v", err)
		aqm.RespondError(w, http.StatusNotFound, "Ticket not found")
		return
	}

	previousStatus := ticket.Status
	ticket.Status = kitchenstatus.Statuses.Started.Code()
	now := time.Now()
	ticket.StartedAt = &now

	if err := h.repo.Update(ctx, ticket); err != nil {
		log.Errorf("cannot update ticket: %v", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update ticket")
		return
	}

	// Update cache after successful DB write
	if h.cache != nil {
		h.cache.Set(ticket)
	}

	h.publishStatusChange(ctx, ticket, previousStatus)
	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) ReadyTicket(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ReadyTicket")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid ticket ID")
		return
	}

	ticket, err := h.repo.FindByID(ctx, id)
	if err != nil {
		log.Errorf("cannot find ticket: %v", err)
		aqm.RespondError(w, http.StatusNotFound, "Ticket not found")
		return
	}

	previousStatus := ticket.Status
	ticket.Status = kitchenstatus.Statuses.Ready.Code()
	now := time.Now()
	ticket.FinishedAt = &now

	if err := h.repo.Update(ctx, ticket); err != nil {
		log.Errorf("cannot update ticket: %v", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update ticket")
		return
	}

	// Update cache after successful DB write
	if h.cache != nil {
		h.cache.Set(ticket)
	}

	h.publishStatusChange(ctx, ticket, previousStatus)
	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) DeliverTicket(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.DeliverTicket")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid ticket ID")
		return
	}

	ticket, err := h.repo.FindByID(ctx, id)
	if err != nil {
		log.Errorf("cannot find ticket: %v", err)
		aqm.RespondError(w, http.StatusNotFound, "Ticket not found")
		return
	}

	previousStatus := ticket.Status
	ticket.Status = kitchenstatus.Statuses.Delivered.Code()
	now := time.Now()
	ticket.DeliveredAt = &now

	if err := h.repo.Update(ctx, ticket); err != nil {
		log.Errorf("cannot update ticket: %v", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update ticket")
		return
	}

	// Update cache after successful DB write
	if h.cache != nil {
		h.cache.Set(ticket)
	}

	h.publishStatusChange(ctx, ticket, previousStatus)
	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) StandbyTicket(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "standby", kitchenstatus.Statuses.Standby.Code())
}

func (h *Handler) BlockTicket(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.BlockTicket")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid ticket ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Could not read request body")
		return
	}

	var payload struct {
		ReasonCodeID string `json:"reason_code_id"`
		Notes        string `json:"notes"`
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			aqm.RespondError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
	}

	ticket, err := h.repo.FindByID(ctx, id)
	if err != nil {
		log.Errorf("cannot find ticket: %v", err)
		aqm.RespondError(w, http.StatusNotFound, "Ticket not found")
		return
	}

	previousStatus := ticket.Status
	ticket.Status = kitchenstatus.Statuses.Block.Code()

	if payload.ReasonCodeID != "" {
		reasonID, err := uuid.Parse(payload.ReasonCodeID)
		if err == nil {
			ticket.ReasonCodeID = &reasonID
		}
	}

	if payload.Notes != "" {
		ticket.Notes = payload.Notes
	}

	if err := h.repo.Update(ctx, ticket); err != nil {
		log.Errorf("cannot update ticket: %v", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update ticket")
		return
	}

	// Update cache after successful DB write
	if h.cache != nil {
		h.cache.Set(ticket)
	}

	h.publishStatusChange(ctx, ticket, previousStatus)
	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) RejectTicket(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "reject", kitchenstatus.Statuses.Reject.Code())
}

func (h *Handler) CancelTicket(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "cancel", kitchenstatus.Statuses.Cancelled.Code())
}

// UpdateTicketStatus handles generic status updates via PATCH /tickets/:id/status
// Accepts {"status": "status-code"} in request body
func (h *Handler) UpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.UpdateTicketStatus")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid ticket ID")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Status == "" {
		aqm.RespondError(w, http.StatusBadRequest, "Status is required")
		return
	}

	ticket, err := h.repo.FindByID(ctx, id)
	if err != nil {
		log.Errorf("cannot find ticket: %v", err)
		aqm.RespondError(w, http.StatusNotFound, "Ticket not found")
		return
	}

	// Prevent moving tickets that are already delivered or cancelled
	terminalStatuses := []string{kitchenstatus.Statuses.Delivered.Code(), kitchenstatus.Statuses.Cancelled.Code()}
	if aqm.IsInList(ticket.Status, terminalStatuses) {
		aqm.RespondError(w, http.StatusBadRequest, "Cannot modify delivered or cancelled tickets")
		return
	}

	// Prevent chef from marking tickets as delivered (only waiters can do this from orders)
	forbiddenFromReady := []string{kitchenstatus.Statuses.Delivered.Code()}
	if ticket.Status == kitchenstatus.Statuses.Ready.Code() && aqm.IsInList(req.Status, forbiddenFromReady) {
		aqm.RespondError(w, http.StatusBadRequest, "Cannot transition from ready to delivered. This must be done from the order by waitstaff.")
		return
	}

	previousStatus := ticket.Status
	ticket.Status = req.Status

	// Update timestamps based on status
	now := time.Now().UTC()
	switch req.Status {
	case kitchenstatus.Statuses.Started.Code():
		if ticket.StartedAt == nil {
			ticket.StartedAt = &now
		}
	case kitchenstatus.Statuses.Ready.Code():
		if ticket.FinishedAt == nil {
			ticket.FinishedAt = &now
		}
	case kitchenstatus.Statuses.Delivered.Code():
		if ticket.DeliveredAt == nil {
			ticket.DeliveredAt = &now
		}
	}

	if err := h.repo.Update(ctx, ticket); err != nil {
		log.Errorf("cannot update ticket: %v", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update ticket")
		return
	}

	// Update cache after successful DB write
	if h.cache != nil {
		h.cache.Set(ticket)
	}

	h.publishStatusChange(ctx, ticket, previousStatus)
	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request, action string, newStatus string) {
	w, r, finish := h.tlm.Start(w, r, fmt.Sprintf("Handler.%sTicket", action))
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid ticket ID")
		return
	}

	ticket, err := h.repo.FindByID(ctx, id)
	if err != nil {
		log.Errorf("cannot find ticket: %v", err)
		aqm.RespondError(w, http.StatusNotFound, "Ticket not found")
		return
	}

	previousStatus := ticket.Status
	ticket.Status = newStatus

	if err := h.repo.Update(ctx, ticket); err != nil {
		log.Errorf("cannot update ticket: %v", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not update ticket")
		return
	}

	// Update cache after successful DB write
	if h.cache != nil {
		h.cache.Set(ticket)
	}

	h.publishStatusChange(ctx, ticket, previousStatus)
	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) publishStatusChange(ctx context.Context, ticket *Ticket, previousStatus string) {
	eventPayload := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    event.EventKitchenTicketStatusChange,
			OccurredAt:   time.Now().UTC(),
			TicketID:     ticket.ID.String(),
			OrderID:      ticket.OrderID.String(),
			OrderItemID:  ticket.OrderItemID.String(),
			MenuItemID:   ticket.MenuItemID.String(),
			Station:      ticket.Station,
			MenuItemName: ticket.MenuItemName,
			StationName:  ticket.StationName,
			TableNumber:  ticket.TableNumber,
		},
		NewStatus:      ticket.Status,
		PreviousStatus: previousStatus,
		Notes:          ticket.Notes,
		StartedAt:      ticket.StartedAt,
		FinishedAt:     ticket.FinishedAt,
		DeliveredAt:    ticket.DeliveredAt,
	}
	if ticket.ReasonCodeID != nil {
		eventPayload.ReasonCodeID = ticket.ReasonCodeID.String()
	}

	eventBytes, _ := json.Marshal(eventPayload)
	if err := h.publisher.Publish(ctx, event.KitchenTicketsTopic, eventBytes); err != nil {
		h.logger.Errorf("Failed to publish status_changed event: %v", err)
	}
}

// ReloadCache reloads the ticket cache from the database
// This is useful after seeding demo data or for cache refresh
func (h *Handler) ReloadCache(w http.ResponseWriter, r *http.Request) {
	w, r, finish := h.tlm.Start(w, r, "Handler.ReloadCache")
	defer finish()
	log := h.log(r)
	ctx := r.Context()

	log.Info("reloading ticket cache")

	if err := h.cache.Warm(ctx); err != nil {
		log.Infof("failed to reload cache: %v", err)
		http.Error(w, fmt.Sprintf("failed to reload cache: %v", err), http.StatusInternalServerError)
		return
	}

	count := h.cache.Count()
	log.Info("cache reloaded successfully", "ticket_count", count)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cache reloaded successfully",
		"count":   count,
	})
}
