package kitchen

import (
	"context"
	"errors"

	"github.com/appetiteclub/apt/events"
	"github.com/google/uuid"
)

// MockTicketRepository is a test mock for TicketRepository
type MockTicketRepository struct {
	tickets        map[uuid.UUID]*Ticket
	byOrderItemID  map[uuid.UUID]*Ticket
	CreateFunc     func(ctx context.Context, t *Ticket) error
	UpdateFunc     func(ctx context.Context, t *Ticket) error
	FindByIDFunc   func(ctx context.Context, id TicketID) (*Ticket, error)
	FindByOrderItemIDFunc func(ctx context.Context, id OrderItemID) (*Ticket, error)
	ListFunc       func(ctx context.Context, filter TicketFilter) ([]Ticket, error)
}

func NewMockTicketRepository() *MockTicketRepository {
	return &MockTicketRepository{
		tickets:       make(map[uuid.UUID]*Ticket),
		byOrderItemID: make(map[uuid.UUID]*Ticket),
	}
}

func (m *MockTicketRepository) Create(ctx context.Context, t *Ticket) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, t)
	}
	m.tickets[t.ID] = t
	m.byOrderItemID[t.OrderItemID] = t
	return nil
}

func (m *MockTicketRepository) Update(ctx context.Context, t *Ticket) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, t)
	}
	if _, exists := m.tickets[t.ID]; !exists {
		return errors.New("ticket not found")
	}
	m.tickets[t.ID] = t
	return nil
}

func (m *MockTicketRepository) FindByID(ctx context.Context, id TicketID) (*Ticket, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	t, exists := m.tickets[id]
	if !exists {
		return nil, errors.New("ticket not found")
	}
	return t, nil
}

func (m *MockTicketRepository) FindByOrderItemID(ctx context.Context, id OrderItemID) (*Ticket, error) {
	if m.FindByOrderItemIDFunc != nil {
		return m.FindByOrderItemIDFunc(ctx, id)
	}
	return m.byOrderItemID[id], nil
}

func (m *MockTicketRepository) List(ctx context.Context, filter TicketFilter) ([]Ticket, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filter)
	}
	result := make([]Ticket, 0, len(m.tickets))
	for _, t := range m.tickets {
		if filter.Station != nil && t.Station != *filter.Station {
			continue
		}
		if filter.Status != nil && t.Status != *filter.Status {
			continue
		}
		if filter.OrderID != nil && t.OrderID != *filter.OrderID {
			continue
		}
		if filter.OrderItemID != nil && t.OrderItemID != *filter.OrderItemID {
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}

// AddTicket is a helper to seed the mock repository
func (m *MockTicketRepository) AddTicket(t *Ticket) {
	m.tickets[t.ID] = t
	m.byOrderItemID[t.OrderItemID] = t
}

// MockPublisher is a test mock for events.Publisher
type MockPublisher struct {
	PublishedEvents []PublishedEvent
	PublishFunc     func(ctx context.Context, topic string, data []byte) error
}

type PublishedEvent struct {
	Topic string
	Data  []byte
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		PublishedEvents: make([]PublishedEvent, 0),
	}
}

func (m *MockPublisher) Publish(ctx context.Context, topic string, data []byte) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, topic, data)
	}
	m.PublishedEvents = append(m.PublishedEvents, PublishedEvent{Topic: topic, Data: data})
	return nil
}

// MockStreamConsumer is a test mock for events.StreamConsumer
type MockStreamConsumer struct {
	messages           []events.StreamMessage
	FetchFunc          func(ctx context.Context, maxMessages int) ([]events.StreamMessage, error)
	SubscribeStreamFunc func(ctx context.Context, handler events.HandlerFunc) error
}

func NewMockStreamConsumer() *MockStreamConsumer {
	return &MockStreamConsumer{
		messages: make([]events.StreamMessage, 0),
	}
}

func (m *MockStreamConsumer) Fetch(ctx context.Context, maxMessages int) ([]events.StreamMessage, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(ctx, maxMessages)
	}
	return m.messages, nil
}

func (m *MockStreamConsumer) SubscribeStream(ctx context.Context, handler events.HandlerFunc) error {
	if m.SubscribeStreamFunc != nil {
		return m.SubscribeStreamFunc(ctx, handler)
	}
	return nil
}

func (m *MockStreamConsumer) AddMessage(data []byte) {
	m.messages = append(m.messages, events.StreamMessage{Data: data})
}
