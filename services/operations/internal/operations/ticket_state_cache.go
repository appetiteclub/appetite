package operations

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/appetiteclub/appetite/pkg/event"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/events"
)

// TicketStateCache maintains an in-memory cache of kitchen tickets,
// indexed by station and status for efficient Kanban board queries.
type TicketStateCache struct {
	mu     sync.RWMutex
	// tickets indexed by ticket_id
	tickets map[string]*kitchenTicketResource
	// index by station_id -> ticket_id
	byStation map[string][]string
	// index by status_id -> ticket_id
	byStatus map[string][]string

	stream    events.StreamConsumer // For event replay on startup
	kitchenDA *KitchenDataAccess    // Fallback for HTTP-based warming
	logger    aqm.Logger
}

// NewTicketStateCache creates a new ticket cache.
func NewTicketStateCache(stream events.StreamConsumer, kitchenDA *KitchenDataAccess, logger aqm.Logger) *TicketStateCache {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &TicketStateCache{
		tickets:   make(map[string]*kitchenTicketResource),
		byStation: make(map[string][]string),
		byStatus:  make(map[string][]string),
		stream:    stream,
		kitchenDA: kitchenDA,
		logger:    logger,
	}
}

// Warm loads tickets into the cache using event replay from Stream.
// Falls back to HTTP GET from Kitchen service if Stream is unavailable.
func (c *TicketStateCache) Warm(ctx context.Context) error {
	// Try event replay first (preferred method)
	if c.stream != nil {
		if err := c.warmFromStream(ctx); err != nil {
			c.logger.Info("stream replay failed, falling back to HTTP", "error", err)
		} else {
			c.removeCompletedTickets()
			return nil
		}
	}

	// Fallback to HTTP GET from Kitchen service
	if c.kitchenDA == nil {
		c.logger.Info("neither stream nor kitchen DA configured, cache remains empty")
		return nil
	}

	return c.warmFromHTTP(ctx)
}

// warmFromStream replays events from the persistent stream to rebuild cache state.
func (c *TicketStateCache) warmFromStream(ctx context.Context) error {
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

// warmFromHTTP loads tickets via HTTP GET from Kitchen service (fallback).
func (c *TicketStateCache) warmFromHTTP(ctx context.Context) error {
	c.logger.Info("warming cache from Kitchen HTTP API")

	tickets, err := c.kitchenDA.ListTickets(ctx)
	if err != nil {
		c.logger.Error("failed to warm ticket cache from HTTP", "error", err)
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range tickets {
		ticket := &tickets[i]
		c.setLocked(ticket)
	}

	c.logger.Info("cache warmed from HTTP", "count", len(tickets))
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

	ticket := &kitchenTicketResource{
		ID:           evt.TicketID,
		OrderID:      evt.OrderID,
		OrderItemID:  evt.OrderItemID,
		MenuItemID:   evt.MenuItemID,
		StationID:    evt.StationID,
		StatusID:     evt.StatusID,
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

	ticket := c.tickets[evt.TicketID]
	if ticket == nil {
		// Create minimal entry if ticket doesn't exist
		ticket = &kitchenTicketResource{
			ID:           evt.TicketID,
			OrderID:      evt.OrderID,
			OrderItemID:  evt.OrderItemID,
			MenuItemID:   evt.MenuItemID,
			StationID:    evt.StationID,
			MenuItemName: evt.MenuItemName,
			StationName:  evt.StationName,
			TableNumber:  evt.TableNumber,
		}
	}

	// Update status and timestamps
	ticket.StatusID = evt.NewStatusID
	ticket.Notes = evt.Notes
	ticket.UpdatedAt = evt.OccurredAt
	ticket.StartedAt = evt.StartedAt
	ticket.FinishedAt = evt.FinishedAt
	ticket.DeliveredAt = evt.DeliveredAt

	if evt.ReasonCodeID != "" {
		ticket.ReasonCodeID = &evt.ReasonCodeID
	}

	c.setLocked(ticket)
}

// removeCompletedTickets filters out delivered and cancelled tickets from the cache.
// This should be called after warming from stream to show only active tickets.
func (c *TicketStateCache) removeCompletedTickets() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var removed int
	for id, ticket := range c.tickets {
		if ticket.StatusID == StatusDelivered || ticket.StatusID == StatusCancelled {
			c.removeFromIndex(c.byStation, ticket.StationID, id)
			c.removeFromIndex(c.byStatus, ticket.StatusID, id)
			delete(c.tickets, id)
			removed++
		}
	}

	c.logger.Info("removed completed tickets from cache", "count", removed)
}

// Set updates or adds a ticket to the cache.
func (c *TicketStateCache) Set(ticket *kitchenTicketResource) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLocked(ticket)
}

func (c *TicketStateCache) setLocked(ticket *kitchenTicketResource) {
	if ticket == nil {
		return
	}

	ticketID := ticket.ID

	// Remove from old indexes if ticket already exists
	if old, exists := c.tickets[ticketID]; exists {
		c.removeFromIndex(c.byStation, old.StationID, ticketID)
		c.removeFromIndex(c.byStatus, old.StatusID, ticketID)
	}

	// Update ticket
	c.tickets[ticketID] = ticket

	// Update indexes
	c.addToIndex(c.byStation, ticket.StationID, ticketID)
	c.addToIndex(c.byStatus, ticket.StatusID, ticketID)
}

// Get retrieves a ticket by ID.
func (c *TicketStateCache) Get(ticketID string) *kitchenTicketResource {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tickets[ticketID]
}

// GetByStation returns all tickets for a given station.
func (c *TicketStateCache) GetByStation(stationID string) []*kitchenTicketResource {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ticketIDs := c.byStation[stationID]
	result := make([]*kitchenTicketResource, 0, len(ticketIDs))
	for _, id := range ticketIDs {
		if ticket := c.tickets[id]; ticket != nil {
			result = append(result, ticket)
		}
	}
	return result
}

// GetByStatus returns all tickets for a given status.
func (c *TicketStateCache) GetByStatus(statusID string) []*kitchenTicketResource {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ticketIDs := c.byStatus[statusID]
	result := make([]*kitchenTicketResource, 0, len(ticketIDs))
	for _, id := range ticketIDs {
		if ticket := c.tickets[id]; ticket != nil {
			result = append(result, ticket)
		}
	}
	return result
}

// GetByStationAndStatus returns tickets filtered by both station and status.
func (c *TicketStateCache) GetByStationAndStatus(stationID, statusID string) []*kitchenTicketResource {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ticketIDs := c.byStation[stationID]
	result := make([]*kitchenTicketResource, 0)
	for _, id := range ticketIDs {
		if ticket := c.tickets[id]; ticket != nil && ticket.StatusID == statusID {
			result = append(result, ticket)
		}
	}
	return result
}

// GetAll returns all cached tickets.
func (c *TicketStateCache) GetAll() []*kitchenTicketResource {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*kitchenTicketResource, 0, len(c.tickets))
	for _, ticket := range c.tickets {
		result = append(result, ticket)
	}
	return result
}

// Remove deletes a ticket from the cache.
func (c *TicketStateCache) Remove(ticketID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ticket := c.tickets[ticketID]
	if ticket == nil {
		return
	}

	c.removeFromIndex(c.byStation, ticket.StationID, ticketID)
	c.removeFromIndex(c.byStatus, ticket.StatusID, ticketID)
	delete(c.tickets, ticketID)
}

// Helper functions for index management

func (c *TicketStateCache) addToIndex(index map[string][]string, key, ticketID string) {
	index[key] = append(index[key], ticketID)
}

func (c *TicketStateCache) removeFromIndex(index map[string][]string, key, ticketID string) {
	ids := index[key]
	for i, id := range ids {
		if id == ticketID {
			index[key] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
}
