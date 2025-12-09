package operations

import (
	"context"
	"errors"

	"github.com/aquamarinepk/aqm/events"
)

// MockStreamConsumer implements events.StreamConsumer for testing
type MockStreamConsumer struct {
	FetchFunc           func(ctx context.Context, maxMessages int) ([]events.StreamMessage, error)
	SubscribeStreamFunc func(ctx context.Context, handler events.HandlerFunc) error
}

func (m *MockStreamConsumer) Fetch(ctx context.Context, maxMessages int) ([]events.StreamMessage, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(ctx, maxMessages)
	}
	return nil, nil
}

func (m *MockStreamConsumer) SubscribeStream(ctx context.Context, handler events.HandlerFunc) error {
	if m.SubscribeStreamFunc != nil {
		return m.SubscribeStreamFunc(ctx, handler)
	}
	return nil
}

// MockKitchenDataAccess implements a mock for KitchenDataAccess
type MockKitchenDataAccess struct {
	ListTicketsFunc func(ctx context.Context) ([]kitchenTicketResource, error)
}

func (m *MockKitchenDataAccess) ListTickets(ctx context.Context) ([]kitchenTicketResource, error) {
	if m.ListTicketsFunc != nil {
		return m.ListTicketsFunc(ctx)
	}
	return nil, errors.New("not implemented")
}
