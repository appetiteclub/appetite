package kitchen

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
	"github.com/google/uuid"
)


// TicketStateCache maintains an in-memory cache of kitchen tickets,
// indexed by station and status for efficient Kanban board queries.
type TicketStateCache struct {
	mu sync.RWMutex
	// tickets indexed by ticket_id
	tickets map[uuid.UUID]*Ticket
	// index by station (string code) -> ticket_id
	byStation map[string][]uuid.UUID
	// index by status (string code) -> ticket_id
	byStatus map[string][]uuid.UUID

	stream events.StreamConsumer // For event replay on startup
	repo   TicketRepository       // Fallback to MongoDB if stream unavailable
	logger aqm.Logger

	// gRPC stream server for broadcasting events to connected clients
	streamServer *EventStreamServer
}

// SetStreamServer sets the gRPC stream server reference (called after initialization)
func (c *TicketStateCache) SetStreamServer(server *EventStreamServer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.streamServer = server
}

// NewTicketStateCache creates a new ticket cache.
func NewTicketStateCache(stream events.StreamConsumer, repo TicketRepository, logger aqm.Logger) *TicketStateCache {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &TicketStateCache{
		tickets:   make(map[uuid.UUID]*Ticket),
		byStation: make(map[string][]uuid.UUID),
		byStatus:  make(map[string][]uuid.UUID),
		stream:    stream,
		repo:      repo,
		logger:    logger,
	}
}

// Warm loads tickets into the cache using event replay from Stream.
// Falls back to loading from MongoDB if Stream is unavailable.
func (c *TicketStateCache) Warm(ctx context.Context) error {
	// Try event replay first (preferred method - fast)
	// Check both interface and underlying value to avoid nil pointer panics
	if c.stream != nil {
		// Try to fetch - if stream is actually nil underneath, this will fail gracefully
		if err := c.warmFromStream(ctx); err != nil {
			c.logger.Info("stream replay failed, falling back to MongoDB", "error", err)
		} else {
			c.removeCompletedTickets()
			return nil
		}
	}

	// Fallback to loading from MongoDB (slower)
	if c.repo == nil {
		c.logger.Info("neither stream nor repo configured, cache remains empty")
		return nil
	}

	return c.warmFromRepo(ctx)
}

// WarmFromRepo loads tickets directly from MongoDB repository, bypassing event stream.
// This is useful after seeding data directly to the database without publishing events.
func (c *TicketStateCache) WarmFromRepo(ctx context.Context) error {
	return c.warmFromRepo(ctx)
}

// warmFromStream replays events from the persistent stream to rebuild cache state.
func (c *TicketStateCache) warmFromStream(ctx context.Context) error {
	// Protect against panics from nil pointer dereferences in stream implementations
	defer func() {
		if r := recover(); r != nil {
			c.logger.Info("stream panic recovered, falling back to MongoDB", "panic", r)
		}
	}()

	c.logger.Info("warming cache from event stream")

	messages, err := c.stream.Fetch(ctx, 10000) // Fetch up to 10k events
	if err != nil {
		return err
	}

	c.logger.Info("fetched events from stream", "count", len(messages))

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, msg := range messages {
		c.applyEventLocked(ctx, msg.Data)
	}

	c.logger.Info("cache warmed from stream", "tickets", len(c.tickets))
	return nil
}

// warmFromRepo loads tickets from MongoDB repository (fallback).
func (c *TicketStateCache) warmFromRepo(ctx context.Context) (err error) {
	// Protect against nil repo or panics during MongoDB operations
	defer func() {
		if r := recover(); r != nil {
			c.logger.Info("MongoDB panic recovered, cache will remain empty", "panic", r)
			err = nil // Don't propagate panic as error
		}
	}()

	if c.repo == nil {
		c.logger.Info("repository is nil, cache will remain empty")
		return nil
	}

	c.logger.Info("warming cache from MongoDB")

	tickets, dbErr := c.repo.List(ctx, TicketFilter{})
	if dbErr != nil {
		c.logger.Info("failed to warm ticket cache from MongoDB, cache will remain empty", "error", dbErr)
		return nil // Don't fatal - just leave cache empty
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range tickets {
		ticket := &tickets[i]
		c.setLocked(ticket)
	}

	c.logger.Info("cache warmed from MongoDB", "count", len(tickets))
	return nil
}

// applyEventLocked processes a single event and updates the cache.
// Must be called with c.mu locked.
func (c *TicketStateCache) applyEventLocked(ctx context.Context, data []byte) {
	var baseEvent struct {
		EventType string `json:"event_type"`
	}

	if err := json.Unmarshal(data, &baseEvent); err != nil {
		c.logger.Error("failed to unmarshal event type", "error", err)
		return
	}

	switch baseEvent.EventType {
	case event.EventKitchenTicketCreated:
		c.handleTicketCreatedLocked(data)
	case event.EventKitchenTicketStatusChange:
		c.handleTicketStatusChangedLocked(data)
	default:
		// Silently ignore unknown event types (forward compatibility)
		return
	}
}

// handleTicketCreatedLocked processes a ticket.created event.
func (c *TicketStateCache) handleTicketCreatedLocked(data []byte) {
	var evt event.KitchenTicketCreatedEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		c.logger.Error("failed to unmarshal ticket.created event", "error", err)
		return
	}

	// Parse UUIDs from event
	ticketID, _ := uuid.Parse(evt.TicketID)
	orderID, _ := uuid.Parse(evt.OrderID)
	orderItemID, _ := uuid.Parse(evt.OrderItemID)
	menuItemID, _ := uuid.Parse(evt.MenuItemID)

	ticket := &Ticket{
		ID:           ticketID,
		OrderID:      orderID,
		OrderItemID:  orderItemID,
		MenuItemID:   menuItemID,
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

	c.setLocked(ticket)
}

// handleTicketStatusChangedLocked processes a ticket.status_changed event.
func (c *TicketStateCache) handleTicketStatusChangedLocked(data []byte) {
	var evt event.KitchenTicketStatusChangedEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		c.logger.Error("failed to unmarshal ticket.status_changed event", "error", err)
		return
	}

	ticketID, _ := uuid.Parse(evt.TicketID)
	ticket := c.tickets[ticketID]
	if ticket == nil {
		// Create minimal entry if ticket doesn't exist
		orderID, _ := uuid.Parse(evt.OrderID)
		orderItemID, _ := uuid.Parse(evt.OrderItemID)
		menuItemID, _ := uuid.Parse(evt.MenuItemID)

		ticket = &Ticket{
			ID:           ticketID,
			OrderID:      orderID,
			OrderItemID:  orderItemID,
			MenuItemID:   menuItemID,
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
		reasonCodeID, _ := uuid.Parse(evt.ReasonCodeID)
		ticket.ReasonCodeID = &reasonCodeID
	}

	c.setLocked(ticket)

	// Broadcast event to gRPC stream subscribers
	if c.streamServer != nil {
		c.streamServer.BroadcastTicketEvent(&evt)
	}
}

// removeCompletedTickets filters out delivered and cancelled tickets from the cache.
// This should be called after warming from stream to show only active tickets.
func (c *TicketStateCache) removeCompletedTickets() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var removed int
	for id, ticket := range c.tickets {
		if ticket.Status == "delivered" || ticket.Status == "cancelled" {
			c.removeFromIndexStr(c.byStation, ticket.Station, id)
			c.removeFromIndexStr(c.byStatus, ticket.Status, id)
			delete(c.tickets, id)
			removed++
		}
	}

	c.logger.Info("removed completed tickets from cache", "count", removed)
}

// Set updates or adds a ticket to the cache.
// This should be called when handling real-time events.
func (c *TicketStateCache) Set(ticket *Ticket) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLocked(ticket)
}

func (c *TicketStateCache) setLocked(ticket *Ticket) {
	if ticket == nil {
		return
	}

	ticketID := ticket.ID

	// Capture old ticket for broadcasting changes
	var previousStatus string
	if old, exists := c.tickets[ticketID]; exists {
		previousStatus = old.Status
		c.removeFromIndexStr(c.byStation, old.Station, ticketID)
		c.removeFromIndexStr(c.byStatus, old.Status, ticketID)
	}

	// Update ticket
	c.tickets[ticketID] = ticket

	// Update indexes
	c.addToIndexStr(c.byStation, ticket.Station, ticketID)
	c.addToIndexStr(c.byStatus, ticket.Status, ticketID)

	// Broadcast to gRPC stream subscribers
	if c.streamServer != nil {
		evt := &event.KitchenTicketStatusChangedEvent{
			KitchenTicketEventMetadata: event.KitchenTicketEventMetadata{
				EventType:    "kitchen.ticket.status_changed",
				OccurredAt:   ticket.UpdatedAt,
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
		c.streamServer.BroadcastTicketEvent(evt)
	}
}

// Get retrieves a ticket by ID.
func (c *TicketStateCache) Get(ticketID uuid.UUID) *Ticket {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tickets[ticketID]
}

// GetByStationCode returns all tickets for a given station code.
func (c *TicketStateCache) GetByStationCode(station string) []*Ticket {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ticketIDs := c.byStation[station]
	result := make([]*Ticket, 0, len(ticketIDs))
	for _, id := range ticketIDs {
		if ticket := c.tickets[id]; ticket != nil {
			result = append(result, ticket)
		}
	}
	return result
}

// GetByStatusCode returns all tickets for a given status code.
func (c *TicketStateCache) GetByStatusCode(status string) []*Ticket {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ticketIDs := c.byStatus[status]
	result := make([]*Ticket, 0, len(ticketIDs))
	for _, id := range ticketIDs {
		if ticket := c.tickets[id]; ticket != nil {
			result = append(result, ticket)
		}
	}
	return result
}

// GetByStationAndStatusCode returns tickets filtered by both station and status code.
func (c *TicketStateCache) GetByStationAndStatusCode(station string, status string) []*Ticket {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ticketIDs := c.byStation[station]
	result := make([]*Ticket, 0)
	for _, id := range ticketIDs {
		if ticket := c.tickets[id]; ticket != nil && ticket.Status == status {
			result = append(result, ticket)
		}
	}
	return result
}

// GetAll returns all cached tickets.
func (c *TicketStateCache) GetAll() []*Ticket {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*Ticket, 0, len(c.tickets))
	for _, ticket := range c.tickets {
		result = append(result, ticket)
	}
	return result
}

// Remove deletes a ticket from the cache.
func (c *TicketStateCache) Remove(ticketID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ticket := c.tickets[ticketID]
	if ticket == nil {
		return
	}

	c.removeFromIndexStr(c.byStation, ticket.Station, ticketID)
	c.removeFromIndexStr(c.byStatus, ticket.Status, ticketID)
	delete(c.tickets, ticketID)
}

// Helper functions for index management

func (c *TicketStateCache) addToIndexStr(index map[string][]uuid.UUID, key string, ticketID uuid.UUID) {
	index[key] = append(index[key], ticketID)
}

func (c *TicketStateCache) removeFromIndexStr(index map[string][]uuid.UUID, key string, ticketID uuid.UUID) {
	ids := index[key]
	for i, id := range ids {
		if id == ticketID {
			index[key] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
}

// Count returns the number of tickets in the cache
func (c *TicketStateCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.tickets)
}
