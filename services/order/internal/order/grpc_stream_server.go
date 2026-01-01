package order

import (
	"sync"
	"time"

	proto "github.com/appetiteclub/appetite/services/order/internal/order/proto"
	"github.com/appetiteclub/apt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OrderEventStreamServer implements the gRPC OrderEventStream service
type OrderEventStreamServer struct {
	proto.UnimplementedOrderEventStreamServer
	orderItemRepo OrderItemRepo
	logger        apt.Logger

	// Manage active stream subscribers
	mu          sync.RWMutex
	subscribers map[string]chan *proto.OrderItemEvent
}

// RegisterGRPCService registers this service with the gRPC server (apt.GRPCServiceRegistrar interface)
func (s *OrderEventStreamServer) RegisterGRPCService(server *grpc.Server) {
	proto.RegisterOrderEventStreamServer(server, s)
}

// NewOrderEventStreamServer creates a new gRPC streaming server for order items
func NewOrderEventStreamServer(orderItemRepo OrderItemRepo, logger apt.Logger) *OrderEventStreamServer {
	return &OrderEventStreamServer{
		orderItemRepo: orderItemRepo,
		logger:        logger,
		subscribers:   make(map[string]chan *proto.OrderItemEvent),
	}
}

// StreamOrderItemEvents implements the gRPC streaming endpoint
func (s *OrderEventStreamServer) StreamOrderItemEvents(req *proto.SubscribeOrderItemEventsRequest, stream proto.OrderEventStream_StreamOrderItemEventsServer) error {
	ctx := stream.Context()
	subscriberID := generateSubscriberID()

	s.logger.Info("new order item events subscriber", "subscriber_id", subscriberID, "table_filter", req.TableId, "order_filter", req.OrderId)

	// Create channel for this subscriber
	eventChan := make(chan *proto.OrderItemEvent, 100)

	s.mu.Lock()
	s.subscribers[subscriberID] = eventChan
	s.mu.Unlock()

	// Cleanup on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.subscribers, subscriberID)
		s.mu.Unlock()
		close(eventChan)
		s.logger.Info("order item events subscriber disconnected", "subscriber_id", subscriberID)
	}()

	// Send initial state - get all order items based on filters
	// For now, we'll skip initial state to simplify
	// In production, you might want to load recent order items from the DB
	// based on req.TableId or req.OrderId filters

	// Stream real-time updates
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt := <-eventChan:
			// TODO: Apply filters if needed (table_id, order_id)
			if err := stream.Send(evt); err != nil {
				s.logger.Errorf("failed to send event: %v", err)
				return err
			}
		}
	}
}

// BroadcastOrderItemEvent sends an event to all connected subscribers
// This should be called when an OrderItem status changes
func (s *OrderEventStreamServer) BroadcastOrderItemEvent(item *OrderItem, eventType string, previousStatus string) {
	s.logger.Info("broadcasting order item event",
		"order_item_id", item.ID.String(),
		"event_type", eventType,
		"old_status", previousStatus,
		"new_status", item.Status,
		"total_subscribers", len(s.subscribers),
	)

	protoEvt := &proto.OrderItemEvent{
		EventType:          eventType,
		OccurredAt:         timestamppb.New(time.Now()),
		OrderItemId:        item.ID.String(),
		OrderId:            item.OrderID.String(),
		DishName:           item.DishName,
		Category:           item.Category,
		NewStatus:          item.Status,
		PreviousStatus:     previousStatus,
		Quantity:           int32(item.Quantity),
		Price:              item.Price,
		RequiresProduction: item.RequiresProduction,
		Notes:              item.Notes,
	}

	if item.MenuItemID != nil {
		protoEvt.MenuItemId = item.MenuItemID.String()
	}

	if item.DeliveredAt != nil {
		protoEvt.DeliveredAt = timestamppb.New(*item.DeliveredAt)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for subscriberID, ch := range s.subscribers {
		select {
		case ch <- protoEvt:
			// Event sent successfully
		default:
			// Channel full, subscriber too slow - skip this event
			s.logger.Info("subscriber channel full, dropping event", "subscriber_id", subscriberID)
		}
	}
}

// Helper to generate unique subscriber IDs
func generateSubscriberID() string {
	return time.Now().Format("20060102150405.000000")
}
