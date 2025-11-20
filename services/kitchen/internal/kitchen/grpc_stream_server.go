package kitchen

import (
	"sync"
	"time"

	"github.com/appetiteclub/appetite/pkg/event"
	proto "github.com/appetiteclub/appetite/services/kitchen/internal/kitchen/proto"
	"github.com/aquamarinepk/aqm"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventStreamServer implements the gRPC EventStream service
type EventStreamServer struct {
	proto.UnimplementedEventStreamServer
	cache  *TicketStateCache
	logger aqm.Logger

	// Manage active stream subscribers
	mu          sync.RWMutex
	subscribers map[string]chan *proto.KitchenTicketEvent
}

// RegisterGRPCService registers this service with the gRPC server (aqm.GRPCServiceRegistrar interface)
func (s *EventStreamServer) RegisterGRPCService(server *grpc.Server) {
	proto.RegisterEventStreamServer(server, s)
}

// NewEventStreamServer creates a new gRPC streaming server
func NewEventStreamServer(cache *TicketStateCache, logger aqm.Logger) *EventStreamServer {
	return &EventStreamServer{
		cache:       cache,
		logger:      logger,
		subscribers: make(map[string]chan *proto.KitchenTicketEvent),
	}
}

// StreamKitchenEvents implements the gRPC streaming endpoint
func (s *EventStreamServer) StreamKitchenEvents(req *proto.SubscribeKitchenEventsRequest, stream proto.EventStream_StreamKitchenEventsServer) error {
	ctx := stream.Context()
	subscriberID := generateSubscriberID()

	s.logger.Info("new kitchen events subscriber", "subscriber_id", subscriberID, "station_filter", req.StationId)

	// Create channel for this subscriber
	eventChan := make(chan *proto.KitchenTicketEvent, 100)

	s.mu.Lock()
	s.subscribers[subscriberID] = eventChan
	s.mu.Unlock()

	// Cleanup on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.subscribers, subscriberID)
		s.mu.Unlock()
		close(eventChan)
		s.logger.Info("kitchen events subscriber disconnected", "subscriber_id", subscriberID)
	}()

	// Send initial state (all current tickets)
	initialTickets := s.cache.GetAll()
	for _, ticket := range initialTickets {
		// Apply station filter if provided
		if req.StationId != "" && ticket.StationID.String() != req.StationId {
			continue
		}

		evt := &proto.KitchenTicketEvent{
			EventType:      "kitchen.ticket.created",
			OccurredAt:     timestamppb.New(ticket.CreatedAt),
			TicketId:       ticket.ID.String(),
			OrderId:        ticket.OrderID.String(),
			OrderItemId:    ticket.OrderItemID.String(),
			MenuItemId:     ticket.MenuItemID.String(),
			StationId:      ticket.StationID.String(),
			MenuItemName:   ticket.MenuItemName,
			StationName:    ticket.StationName,
			TableNumber:    ticket.TableNumber,
			NewStatusId:    ticket.StatusID.String(),
			Quantity:       int32(ticket.Quantity),
			Notes:          ticket.Notes,
		}

		if ticket.StartedAt != nil {
			evt.StartedAt = timestamppb.New(*ticket.StartedAt)
		}
		if ticket.FinishedAt != nil {
			evt.FinishedAt = timestamppb.New(*ticket.FinishedAt)
		}
		if ticket.DeliveredAt != nil {
			evt.DeliveredAt = timestamppb.New(*ticket.DeliveredAt)
		}

		if err := stream.Send(evt); err != nil {
			s.logger.Errorf("failed to send initial ticket: %v", err)
			return err
		}
	}

	// Stream real-time updates
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt := <-eventChan:
			// Apply station filter
			if req.StationId != "" && evt.StationId != req.StationId {
				continue
			}

			if err := stream.Send(evt); err != nil {
				s.logger.Errorf("failed to send event: %v", err)
				return err
			}
		}
	}
}

// BroadcastTicketEvent sends an event to all connected subscribers
// This should be called by the TicketStateCache when it receives NATS events
func (s *EventStreamServer) BroadcastTicketEvent(evt *event.KitchenTicketStatusChangedEvent) {
	protoEvt := &proto.KitchenTicketEvent{
		EventType:        evt.EventType,
		OccurredAt:       timestamppb.New(evt.OccurredAt),
		TicketId:         evt.TicketID,
		OrderId:          evt.OrderID,
		OrderItemId:      evt.OrderItemID,
		MenuItemId:       evt.MenuItemID,
		StationId:        evt.StationID,
		MenuItemName:     evt.MenuItemName,
		StationName:      evt.StationName,
		TableNumber:      evt.TableNumber,
		NewStatusId:      evt.NewStatusID,
		PreviousStatusId: evt.PreviousStatusID,
		Notes:            evt.Notes,
	}

	if evt.StartedAt != nil {
		protoEvt.StartedAt = timestamppb.New(*evt.StartedAt)
	}
	if evt.FinishedAt != nil {
		protoEvt.FinishedAt = timestamppb.New(*evt.FinishedAt)
	}
	if evt.DeliveredAt != nil {
		protoEvt.DeliveredAt = timestamppb.New(*evt.DeliveredAt)
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
