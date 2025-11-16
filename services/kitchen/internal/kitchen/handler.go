package kitchen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/aquamarinepk/aqm/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const MaxBodyBytes = 1 << 20

type Handler struct {
	repo      TicketRepository
	publisher events.Publisher
	logger    aqm.Logger
	config    *aqm.Config
	tlm       *telemetry.HTTP
}

func NewHandler(repo TicketRepository, publisher events.Publisher, config *aqm.Config, logger aqm.Logger) *Handler {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &Handler{
		repo:      repo,
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

	tickets, err := h.repo.List(ctx, filter)
	if err != nil {
		log.Errorf("cannot list tickets: %v", err)
		aqm.RespondError(w, http.StatusInternalServerError, "Could not list tickets")
		return
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

	h.publishStatusChange(ctx, ticket, previousStatus)
	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) RejectTicket(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "reject", uuid.MustParse("00000000-0000-0000-0000-000000000006"))
}

func (h *Handler) CancelTicket(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "cancel", uuid.MustParse("00000000-0000-0000-0000-000000000010"))
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

	h.publishStatusChange(ctx, ticket, previousStatus)
	aqm.Respond(w, http.StatusOK, ticket, nil)
}

func (h *Handler) publishStatusChange(ctx context.Context, ticket *Ticket, previousStatus StatusID) {
	event := map[string]interface{}{
		"event_type":         "kitchen.ticket.status_changed",
		"occurred_at":        time.Now(),
		"ticket_id":          ticket.ID.String(),
		"order_id":           ticket.OrderID.String(),
		"new_status_id":      ticket.StatusID.String(),
		"previous_status_id": previousStatus.String(),
	}

	if ticket.ReasonCodeID != nil {
		event["reason_code_id"] = ticket.ReasonCodeID.String()
	}

	if ticket.Notes != "" {
		event["notes"] = ticket.Notes
	}

	eventBytes, _ := json.Marshal(event)
	if err := h.publisher.Publish(ctx, "kitchen.tickets", eventBytes); err != nil {
		h.logger.Errorf("Failed to publish status_changed event: %v", err)
	}
}
