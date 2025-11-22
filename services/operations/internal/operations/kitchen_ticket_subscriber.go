package operations

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
)

// KitchenTicketSubscriber listens to kitchen.tickets events and updates the cache.
type KitchenTicketSubscriber struct {
	subscriber events.Subscriber
	cache      *TicketStateCache
	logger     aqm.Logger
}

// NewKitchenTicketSubscriber creates a new subscriber for kitchen ticket events.
func NewKitchenTicketSubscriber(subscriber events.Subscriber, cache *TicketStateCache, logger aqm.Logger) *KitchenTicketSubscriber {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &KitchenTicketSubscriber{
		subscriber: subscriber,
		cache:      cache,
		logger:     logger,
	}
}

// Start begins listening to kitchen ticket events.
func (s *KitchenTicketSubscriber) Start(ctx context.Context) error {
	if s.subscriber == nil {
		s.logger.Info("NATS subscriber not configured, skipping kitchen ticket subscription")
		return nil
	}

	s.logger.Info("subscribing to kitchen.tickets topic")
	if err := s.subscriber.Subscribe(ctx, event.KitchenTicketsTopic, s.handleEvent); err != nil {
		return fmt.Errorf("failed to subscribe to kitchen.tickets: %w", err)
	}

	s.logger.Info("kitchen ticket subscriber started")
	return nil
}

// Stop is a no-op for lifecycle compatibility.
func (s *KitchenTicketSubscriber) Stop(ctx context.Context) error {
	return nil
}

func (s *KitchenTicketSubscriber) handleEvent(ctx context.Context, msg []byte) error {
	var baseEvent struct {
		EventType string `json:"event_type"`
	}

	if err := json.Unmarshal(msg, &baseEvent); err != nil {
		s.logger.Error("failed to unmarshal event type", "error", err)
		return nil
	}

	switch baseEvent.EventType {
	case event.EventKitchenTicketCreated:
		return s.handleTicketCreated(ctx, msg)
	case event.EventKitchenTicketStatusChange:
		return s.handleTicketStatusChanged(ctx, msg)
	default:
		s.logger.Debug("ignoring unknown event type", "event_type", baseEvent.EventType)
		return nil
	}
}

func (s *KitchenTicketSubscriber) handleTicketCreated(ctx context.Context, msg []byte) error {
	var evt event.KitchenTicketCreatedEvent
	if err := json.Unmarshal(msg, &evt); err != nil {
		s.logger.Error("failed to unmarshal ticket.created event", "error", err)
		return nil
	}

	// Convert event to cache resource
	ticket := &kitchenTicketResource{
		ID:           evt.TicketID,
		OrderID:      evt.OrderID,
		OrderItemID:  evt.OrderItemID,
		MenuItemID:   evt.MenuItemID,
		Station:      evt.Station,
		Status:       evt.Status,
		Quantity:     evt.Quantity,
		Notes:        evt.Notes,
		MenuItemName: evt.MenuItemName,
		StationName:  evt.StationName,
		TableNumber:  evt.TableNumber,
		CreatedAt:    evt.OccurredAt,
		UpdatedAt:    evt.OccurredAt,
	}

	s.cache.Set(ticket)
	s.logger.Debug("ticket created", "ticket_id", evt.TicketID, "station", evt.StationName)
	return nil
}

func (s *KitchenTicketSubscriber) handleTicketStatusChanged(ctx context.Context, msg []byte) error {
	var evt event.KitchenTicketStatusChangedEvent
	if err := json.Unmarshal(msg, &evt); err != nil {
		s.logger.Error("failed to unmarshal ticket.status_changed event", "error", err)
		return nil
	}

	// Get existing ticket from cache or create minimal one
	ticket := s.cache.Get(evt.TicketID)
	if ticket == nil {
		// Ticket not in cache, create minimal entry
		ticket = &kitchenTicketResource{
			ID:           evt.TicketID,
			OrderID:      evt.OrderID,
			OrderItemID:  evt.OrderItemID,
			MenuItemID:   evt.MenuItemID,
			Station:      evt.Station,
			MenuItemName: evt.MenuItemName,
			StationName:  evt.StationName,
			TableNumber:  evt.TableNumber,
		}
	}

	// Update status and timestamps
	ticket.Status = evt.NewStatus
	ticket.Notes = evt.Notes
	ticket.UpdatedAt = evt.OccurredAt
	ticket.StartedAt = evt.StartedAt
	ticket.FinishedAt = evt.FinishedAt
	ticket.DeliveredAt = evt.DeliveredAt

	if evt.ReasonCodeID != "" {
		ticket.ReasonCodeID = &evt.ReasonCodeID
	}

	s.cache.Set(ticket)
	s.logger.Debug("ticket status changed", "ticket_id", evt.TicketID, "new_status", evt.NewStatus)
	return nil
}
