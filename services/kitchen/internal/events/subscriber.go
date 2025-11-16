package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/appetiteclub/appetite/services/kitchen/internal/kitchen"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/google/uuid"
)

type OrderItemEvent struct {
	EventType         string    `json:"event_type"`
	OccurredAt        time.Time `json:"occurred_at"`
	OrderID           string    `json:"order_id"`
	OrderItemID       string    `json:"order_item_id"`
	MenuItemID        string    `json:"menu_item_id"`
	Quantity          int       `json:"quantity"`
	Notes             string    `json:"notes,omitempty"`
	RequiresProduction bool      `json:"requires_production"`
	ProductionStation string    `json:"production_station,omitempty"`
}

type OrderItemSubscriber struct {
	subscriber events.Subscriber
	repo       kitchen.TicketRepository
	publisher  events.Publisher
	logger     aqm.Logger
}

func NewOrderItemSubscriber(
	subscriber events.Subscriber,
	repo kitchen.TicketRepository,
	publisher events.Publisher,
	logger aqm.Logger,
) *OrderItemSubscriber {
	return &OrderItemSubscriber{
		subscriber: subscriber,
		repo:       repo,
		publisher:  publisher,
		logger:     logger,
	}
}

func (s *OrderItemSubscriber) Start(ctx context.Context) error {
	s.logger.Info("Starting OrderItemSubscriber for topic: orders.items")

	if err := s.subscriber.Subscribe(ctx, "orders.items", s.handleEvent); err != nil {
		return fmt.Errorf("failed to subscribe to orders.items: %w", err)
	}

	s.logger.Info("OrderItemSubscriber started successfully")
	return nil
}

func (s *OrderItemSubscriber) handleEvent(ctx context.Context, msg []byte) error {
	var event OrderItemEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		s.logger.Errorf("Failed to unmarshal event: %v", err)
		return nil
	}

	if !event.RequiresProduction {
		return nil
	}

	switch event.EventType {
	case "order.item.created":
		return s.handleCreated(ctx, &event)
	case "order.item.updated":
		return s.handleUpdated(ctx, &event)
	case "order.item.cancelled":
		return s.handleCancelled(ctx, &event)
	default:
		s.logger.Infof("Unknown event type: %s", event.EventType)
	}

	return nil
}

func (s *OrderItemSubscriber) handleCreated(ctx context.Context, event *OrderItemEvent) error {
	orderItemID, err := uuid.Parse(event.OrderItemID)
	if err != nil {
		s.logger.Errorf("Invalid order_item_id: %v", err)
		return nil
	}

	existing, err := s.repo.FindByOrderItemID(ctx, orderItemID)
	if err != nil {
		s.logger.Errorf("Error checking existing ticket: %v", err)
		return err
	}

	if existing != nil {
		return nil
	}

	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		s.logger.Errorf("Invalid order_id: %v", err)
		return nil
	}

	menuItemID, err := uuid.Parse(event.MenuItemID)
	if err != nil {
		s.logger.Errorf("Invalid menu_item_id: %v", err)
		return nil
	}

	stationID, err := uuid.Parse(event.ProductionStation)
	if err != nil {
		s.logger.Errorf("Invalid production_station: %v", err)
		return nil
	}

	statusID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	ticket := &kitchen.Ticket{
		ID:          uuid.New(),
		OrderID:     orderID,
		OrderItemID: orderItemID,
		MenuItemID:  menuItemID,
		StationID:   stationID,
		Quantity:    event.Quantity,
		StatusID:    statusID,
		Notes:       event.Notes,
	}

	if err := s.repo.Create(ctx, ticket); err != nil {
		s.logger.Errorf("Failed to create ticket: %v", err)
		return err
	}

	s.logger.Infof("Created ticket %s for order item %s", ticket.ID, event.OrderItemID)

	ticketEvent := map[string]interface{}{
		"event_type":  "kitchen.ticket.created",
		"occurred_at": time.Now(),
		"ticket_id":   ticket.ID.String(),
		"order_id":    ticket.OrderID.String(),
		"station_id":  ticket.StationID.String(),
		"status_id":   ticket.StatusID.String(),
	}

	eventBytes, _ := json.Marshal(ticketEvent)
	if err := s.publisher.Publish(ctx, "kitchen.tickets", eventBytes); err != nil {
		s.logger.Errorf("Failed to publish ticket.created event: %v", err)
	}

	return nil
}

func (s *OrderItemSubscriber) handleUpdated(ctx context.Context, event *OrderItemEvent) error {
	orderItemID, err := uuid.Parse(event.OrderItemID)
	if err != nil {
		return nil
	}

	ticket, err := s.repo.FindByOrderItemID(ctx, orderItemID)
	if err != nil || ticket == nil {
		return err
	}

	ticket.Quantity = event.Quantity
	ticket.Notes = event.Notes

	if err := s.repo.Update(ctx, ticket); err != nil {
		s.logger.Errorf("Failed to update ticket: %v", err)
		return err
	}

	s.logger.Infof("Updated ticket %s for order item %s", ticket.ID, event.OrderItemID)
	return nil
}

func (s *OrderItemSubscriber) handleCancelled(ctx context.Context, event *OrderItemEvent) error {
	orderItemID, err := uuid.Parse(event.OrderItemID)
	if err != nil {
		return nil
	}

	ticket, err := s.repo.FindByOrderItemID(ctx, orderItemID)
	if err != nil || ticket == nil {
		return err
	}

	cancelledStatusID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	ticket.StatusID = cancelledStatusID

	if err := s.repo.Update(ctx, ticket); err != nil {
		s.logger.Errorf("Failed to cancel ticket: %v", err)
		return err
	}

	s.logger.Infof("Cancelled ticket %s for order item %s", ticket.ID, event.OrderItemID)

	ticketEvent := map[string]interface{}{
		"event_type":       "kitchen.ticket.status_changed",
		"occurred_at":      time.Now(),
		"ticket_id":        ticket.ID.String(),
		"order_id":         ticket.OrderID.String(),
		"new_status_id":    ticket.StatusID.String(),
		"previous_status_id": "00000000-0000-0000-0000-000000000001",
	}

	eventBytes, _ := json.Marshal(ticketEvent)
	if err := s.publisher.Publish(ctx, "kitchen.tickets", eventBytes); err != nil {
		s.logger.Errorf("Failed to publish ticket.status_changed event: %v", err)
	}

	return nil
}
