package kitchen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/aquamarinepk/aqm/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const MaxBodyBytes = 1 << 20

// Status UUIDs for ticket states
var (
	StatusCreated   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	StatusAccepted  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	StatusStarted   = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	StatusReady     = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	StatusDelivered = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	StatusCancelled = uuid.MustParse("00000000-0000-0000-0000-000000000010")
)

type Handler struct {
	repo      TicketRepository
	cache     *TicketStateCache
	publisher events.Publisher
	logger    aqm.Logger
	config    *aqm.Config
	tlm       *telemetry.HTTP
}

func NewHandler(repo TicketRepository, cache *TicketStateCache, publisher events.Publisher, config *aqm.Config, logger aqm.Logger) *Handler {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &Handler{
		repo:      repo,
		cache:     cache,
		publisher: publisher,
		logger:    logger,
		config:    config,
		tlm:       telemetry.NewHTTP(),
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

	if stationIDStr := r.URL.Query().Get("station"); stationIDStr != "" {
		stationID, err := uuid.Parse(stationIDStr)
		if err != nil {
			aqm.RespondError(w, http.StatusBadRequest, "Invalid station ID")
			return
		}
		filter.StationID = &stationID
	}

	if statusIDStr := r.URL.Query().Get("status"); statusIDStr != "" {
		statusID, err := uuid.Parse(statusIDStr)
		if err != nil {
			aqm.RespondError(w, http.StatusBadRequest, "Invalid status ID")
			return
		}
		filter.StatusID = &statusID
	}

	if orderIDStr := r.URL.Query().Get("order_id"); orderIDStr != "" {
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
		if filter.StationID != nil && filter.StatusID != nil {
			tickets = h.cache.GetByStationAndStatus(*filter.StationID, *filter.StatusID)
		} else if filter.StationID != nil {
			tickets = h.cache.GetByStation(*filter.StationID)
		} else if filter.StatusID != nil {
			tickets = h.cache.GetByStatus(*filter.StatusID)
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
	h.updateStatus(w, r, "accept", uuid.MustParse("00000000-0000-0000-0000-000000000002"))
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

	previousStatus := ticket.StatusID
	ticket.StatusID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
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

	previousStatus := ticket.StatusID
	ticket.StatusID = uuid.MustParse("00000000-0000-0000-0000-000000000004")
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

	previousStatus := ticket.StatusID
	ticket.StatusID = uuid.MustParse("00000000-0000-0000-0000-000000000005")
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
	h.updateStatus(w, r, "standby", uuid.MustParse("00000000-0000-0000-0000-000000000007"))
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

	previousStatus := ticket.StatusID
	ticket.StatusID = uuid.MustParse("00000000-0000-0000-0000-000000000008")

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
	h.updateStatus(w, r, "reject", uuid.MustParse("00000000-0000-0000-0000-000000000006"))
}

func (h *Handler) CancelTicket(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "cancel", uuid.MustParse("00000000-0000-0000-0000-000000000010"))
}

// UpdateTicketStatus handles generic status updates via PATCH /tickets/:id/status
// Accepts {"status_id": "uuid"} in request body
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
		StatusID string `json:"status_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	newStatusID, err := uuid.Parse(req.StatusID)
	if err != nil {
		aqm.RespondError(w, http.StatusBadRequest, "Invalid status_id")
		return
	}

	ticket, err := h.repo.FindByID(ctx, id)
	if err != nil {
		log.Errorf("cannot find ticket: %v", err)
		aqm.RespondError(w, http.StatusNotFound, "Ticket not found")
		return
	}

	// Prevent moving tickets that are already delivered or cancelled
	terminalStatuses := []uuid.UUID{StatusDelivered, StatusCancelled}
	if aqm.IsInList(ticket.StatusID, terminalStatuses) {
		aqm.RespondError(w, http.StatusBadRequest, "Cannot modify delivered or cancelled tickets")
		return
	}

	// Prevent chef from marking tickets as delivered (only waiters can do this from orders)
	forbiddenFromReady := []uuid.UUID{StatusDelivered}
	if ticket.StatusID == StatusReady && aqm.IsInList(newStatusID, forbiddenFromReady) {
		aqm.RespondError(w, http.StatusBadRequest, "Cannot transition from ready to delivered. This must be done from the order by waitstaff.")
		return
	}

	previousStatus := ticket.StatusID
	ticket.StatusID = newStatusID

	// Update timestamps based on status
	now := time.Now().UTC()
	switch newStatusID.String() {
	case StatusStarted.String():
		if ticket.StartedAt == nil {
			ticket.StartedAt = &now
		}
	case StatusReady.String():
		if ticket.FinishedAt == nil {
			ticket.FinishedAt = &now
		}
	case StatusDelivered.String():
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

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request, action string, newStatusID StatusID) {
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

	previousStatus := ticket.StatusID
	ticket.StatusID = newStatusID

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

func (h *Handler) publishStatusChange(ctx context.Context, ticket *Ticket, previousStatus StatusID) {
	eventPayload := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    event.EventKitchenTicketStatusChange,
			OccurredAt:   time.Now().UTC(),
			TicketID:     ticket.ID.String(),
			OrderID:      ticket.OrderID.String(),
			OrderItemID:  ticket.OrderItemID.String(),
			MenuItemID:   ticket.MenuItemID.String(),
			StationID:    ticket.StationID.String(),
			MenuItemName: ticket.MenuItemName,
			StationName:  ticket.StationName,
			TableNumber:  ticket.TableNumber,
		},
		NewStatusID:      ticket.StatusID.String(),
		PreviousStatusID: previousStatus.String(),
		Notes:            ticket.Notes,
		StartedAt:        ticket.StartedAt,
		FinishedAt:       ticket.FinishedAt,
		DeliveredAt:      ticket.DeliveredAt,
	}
	if ticket.ReasonCodeID != nil {
		eventPayload.ReasonCodeID = ticket.ReasonCodeID.String()
	}

	eventBytes, _ := json.Marshal(eventPayload)
	if err := h.publisher.Publish(ctx, event.KitchenTicketsTopic, eventBytes); err != nil {
		h.logger.Errorf("Failed to publish status_changed event: %v", err)
	}
}
