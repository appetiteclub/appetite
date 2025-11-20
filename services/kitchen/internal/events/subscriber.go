package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/appetiteclub/appetite/services/kitchen/internal/kitchen"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/google/uuid"
)


type OrderItemSubscriber struct {
	subscriber events.Subscriber
	repo       kitchen.TicketRepository
	cache      *kitchen.TicketStateCache
	publisher  events.Publisher
	logger     aqm.Logger
}

func NewOrderItemSubscriber(
	subscriber events.Subscriber,
	repo kitchen.TicketRepository,
	cache *kitchen.TicketStateCache,
	publisher events.Publisher,
	logger aqm.Logger,
) *OrderItemSubscriber {
	return &OrderItemSubscriber{
		subscriber: subscriber,
		repo:       repo,
		cache:      cache,
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
	var evt event.OrderItemEvent
	if err := json.Unmarshal(msg, &evt); err != nil {
		s.logger.Errorf("Failed to unmarshal event: %v", err)
		return nil
	}

	if !evt.RequiresProduction {
		return nil
	}

	switch evt.EventType {
	case event.EventOrderItemCreated:
		return s.handleCreated(ctx, &evt)
	case event.EventOrderItemUpdated:
		return s.handleUpdated(ctx, &evt)
	case event.EventOrderItemCancelled:
		return s.handleCancelled(ctx, &evt)
	default:
		s.logger.Infof("Unknown event type: %s", evt.EventType)
	}

	return nil
}

func (s *OrderItemSubscriber) handleCreated(ctx context.Context, evt *event.OrderItemEvent) error {
	orderItemID, err := uuid.Parse(evt.OrderItemID)
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

	orderID, err := uuid.Parse(evt.OrderID)
	if err != nil {
		s.logger.Errorf("Invalid order_id: %v", err)
		return nil
	}

	menuItemID, err := uuid.Parse(evt.MenuItemID)
	if err != nil {
		s.logger.Errorf("Invalid menu_item_id: %v", err)
		return nil
	}

	stationID, err := uuid.Parse(evt.ProductionStation)
	if err != nil {
		s.logger.Errorf("Invalid production_station: %v", err)
		return nil
	}

	statusID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	ticket := &kitchen.Ticket{
		ID:           uuid.New(),
		OrderID:      orderID,
		OrderItemID:  orderItemID,
		MenuItemID:   menuItemID,
		StationID:    stationID,
		Quantity:     evt.Quantity,
		StatusID:     statusID,
		Notes:        evt.Notes,
		MenuItemName: evt.MenuItemName,
		StationName:  evt.StationName,
		TableNumber:  evt.TableNumber,
	}

	if err := s.repo.Create(ctx, ticket); err != nil {
		s.logger.Errorf("Failed to create ticket: %v", err)
		return err
	}

	// Update cache with new ticket
	if s.cache != nil {
		s.cache.Set(ticket)
	}

	s.logger.Infof("Created ticket %s for order item %s", ticket.ID, evt.OrderItemID)

	eventPayload := event.KitchenTicketCreatedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:    event.EventKitchenTicketCreated,
			OccurredAt:   time.Now().UTC(),
			TicketID:     ticket.ID.String(),
			OrderID:      ticket.OrderID.String(),
			OrderItemID:  ticket.OrderItemID.String(),
			MenuItemID:   ticket.MenuItemID.String(),
			StationID:    ticket.StationID.String(),
			MenuItemName: evt.MenuItemName,
			StationName:  evt.StationName,
			TableNumber:  evt.TableNumber,
		},
		StatusID: ticket.StatusID.String(),
		Quantity: ticket.Quantity,
		Notes:    ticket.Notes,
	}

	eventBytes, _ := json.Marshal(eventPayload)
	if err := s.publisher.Publish(ctx, event.KitchenTicketsTopic, eventBytes); err != nil {
		s.logger.Errorf("Failed to publish ticket.created event: %v", err)
	}

	return nil
}

func (s *OrderItemSubscriber) handleUpdated(ctx context.Context, evt *event.OrderItemEvent) error{
	orderItemID, err := uuid.Parse(evt.OrderItemID)
	if err != nil {
		return nil
	}

	ticket, err := s.repo.FindByOrderItemID(ctx, orderItemID)
	if err != nil || ticket == nil {
		return err
	}

	ticket.Quantity = evt.Quantity
	ticket.Notes = evt.Notes

	if err := s.repo.Update(ctx, ticket); err != nil {
		s.logger.Errorf("Failed to update ticket: %v", err)
		return err
	}

	// Update cache
	if s.cache != nil {
		s.cache.Set(ticket)
	}

	s.logger.Infof("Updated ticket %s for order item %s", ticket.ID, evt.OrderItemID)
	return nil
}

func (s *OrderItemSubscriber) handleCancelled(ctx context.Context, evt *event.OrderItemEvent) error {
	orderItemID, err := uuid.Parse(evt.OrderItemID)
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

	// Update cache (or remove if filtering out cancelled)
	if s.cache != nil {
		s.cache.Set(ticket)
	}

	s.logger.Infof("Cancelled ticket %s for order item %s", ticket.ID, evt.OrderItemID)

	eventPayload := event.KitchenTicketStatusChangedEvent{
		KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
			EventType:   event.EventKitchenTicketStatusChange,
			OccurredAt:  time.Now().UTC(),
			TicketID:    ticket.ID.String(),
			OrderID:     ticket.OrderID.String(),
			OrderItemID: ticket.OrderItemID.String(),
			MenuItemID:  ticket.MenuItemID.String(),
			StationID:   ticket.StationID.String(),
		},
		NewStatusID:      ticket.StatusID.String(),
		PreviousStatusID: "00000000-0000-0000-0000-000000000001",
		Notes:            ticket.Notes,
	}

	eventBytes, _ := json.Marshal(eventPayload)
	if err := s.publisher.Publish(ctx, event.KitchenTicketsTopic, eventBytes); err != nil {
		s.logger.Errorf("Failed to publish ticket.status_changed event: %v", err)
	}

	return nil
}
