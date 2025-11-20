package order

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/google/uuid"
)

type KitchenTicketSubscriber struct {
	subscriber    events.Subscriber
	orderItemRepo OrderItemRepo
	streamServer  *OrderEventStreamServer
	logger        aqm.Logger
}

func NewKitchenTicketSubscriber(sub events.Subscriber, orderItemRepo OrderItemRepo, logger aqm.Logger) *KitchenTicketSubscriber {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &KitchenTicketSubscriber{
		subscriber:    sub,
		orderItemRepo: orderItemRepo,
		logger:        logger,
	}
}

// SetStreamServer sets the gRPC stream server for broadcasting events
func (s *KitchenTicketSubscriber) SetStreamServer(streamServer *OrderEventStreamServer) {
	s.streamServer = streamServer
}

func (s *KitchenTicketSubscriber) Start(ctx context.Context) error {
	s.logger.Info("starting kitchen ticket subscriber", "topic", event.KitchenTicketsTopic)
	if s.subscriber == nil {
		return fmt.Errorf("kitchen ticket subscriber not configured")
	}
	return s.subscriber.Subscribe(ctx, event.KitchenTicketsTopic, s.handleEvent)
}

func (s *KitchenTicketSubscriber) handleEvent(ctx context.Context, msg []byte) error {
	// Parse base metadata to determine event type
	var metadata event.KitchenTicketEventMetadata
	if err := json.Unmarshal(msg, &metadata); err != nil {
		s.logger.Info("invalid kitchen ticket event", "error", err)
		return nil
	}

	switch metadata.EventType {
	case event.EventKitchenTicketStatusChange:
		return s.handleStatusChange(ctx, msg)
	case event.EventKitchenTicketCreated:
		// We don't need to handle ticket creation - it was triggered by OrderItem creation
		return nil
	default:
		s.logger.Debug("unknown kitchen ticket event type", "event_type", metadata.EventType)
		return nil
	}
}

func (s *KitchenTicketSubscriber) handleStatusChange(ctx context.Context, msg []byte) error {
	var evt event.KitchenTicketStatusChangedEvent
	if err := json.Unmarshal(msg, &evt); err != nil {
		s.logger.Info("invalid status change event", "error", err)
		return nil
	}

	// Parse OrderItemID from event
	if evt.OrderItemID == "" {
		s.logger.Debug("status change event missing order_item_id", "ticket_id", evt.TicketID)
		return nil
	}

	orderItemID, err := uuid.Parse(evt.OrderItemID)
	if err != nil {
		s.logger.Info("invalid order_item_id in event", "order_item_id", evt.OrderItemID)
		return nil
	}

	// Fetch the OrderItem
	orderItem, err := s.orderItemRepo.Get(ctx, orderItemID)
	if err != nil {
		s.logger.Info("cannot find order item for ticket", "order_item_id", orderItemID, "error", err)
		return nil
	}

	// Map kitchen ticket status to order item status
	newStatus := s.mapKitchenStatusToOrderStatus(evt.NewStatusID)
	if newStatus == "" {
		s.logger.Debug("no status mapping for kitchen status", "status_id", evt.NewStatusID)
		return nil
	}

	// Update OrderItem status
	oldStatus := orderItem.Status
	orderItem.Status = newStatus

	// Update timestamps based on status
	switch newStatus {
	case "preparing":
		orderItem.MarkAsPreparing()
	case "ready":
		orderItem.MarkAsReady()
	case "delivered":
		orderItem.MarkAsDelivered()
	case "cancelled":
		orderItem.Cancel()
	default:
		orderItem.BeforeUpdate()
	}

	if err := s.orderItemRepo.Save(ctx, orderItem); err != nil {
		s.logger.Info("failed to update order item status", "order_item_id", orderItemID, "error", err)
		return err
	}

	s.logger.Info("order item status updated from kitchen event",
		"order_item_id", orderItemID,
		"old_status", oldStatus,
		"new_status", newStatus,
		"ticket_id", evt.TicketID,
	)

	// Broadcast the status change to gRPC stream subscribers
	if s.streamServer != nil {
		s.streamServer.BroadcastOrderItemEvent(orderItem, "order.item.status_changed", oldStatus)
	} else {
		s.logger.Info("streamServer is nil, cannot broadcast event", "order_item_id", orderItemID)
	}

	return nil
}

// mapKitchenStatusToOrderStatus maps kitchen ticket status UUIDs to order item status strings
func (s *KitchenTicketSubscriber) mapKitchenStatusToOrderStatus(kitchenStatusID string) string {
	// Kitchen status IDs from kitchen service:
	// 00000000-0000-0000-0000-000000000001 = Created (Received)
	// 00000000-0000-0000-0000-000000000003 = Started (In Preparation)
	// 00000000-0000-0000-0000-000000000004 = Ready (Ready for Delivery)
	// 00000000-0000-0000-0000-000000000005 = Delivered
	// 00000000-0000-0000-0000-000000000010 = Cancelled (Rejected)

	switch kitchenStatusID {
	case "00000000-0000-0000-0000-000000000001":
		return "pending" // Kitchen received the order
	case "00000000-0000-0000-0000-000000000003":
		return "preparing" // Kitchen started preparation
	case "00000000-0000-0000-0000-000000000004":
		return "ready" // Ready for delivery
	case "00000000-0000-0000-0000-000000000005":
		return "delivered" // Delivered to customer
	case "00000000-0000-0000-0000-000000000010":
		return "cancelled" // Kitchen rejected/cancelled
	default:
		return "" // Unknown status, no mapping
	}
}
