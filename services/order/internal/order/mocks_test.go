package order

import (
	"context"
	"fmt"
	"sync"

	"github.com/appetiteclub/apt/events"
	"github.com/google/uuid"
)

// MockPublisher is a mock implementation of events.Publisher for testing
type MockPublisher struct {
	PublishFunc func(ctx context.Context, topic string, msg []byte) error
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{}
}

func (m *MockPublisher) Publish(ctx context.Context, topic string, msg []byte) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, topic, msg)
	}
	return nil
}

// MockSubscriber is a mock implementation of events.Subscriber for testing
type MockSubscriber struct {
	SubscribeFunc func(ctx context.Context, topic string, handler events.HandlerFunc) error
}

func NewMockSubscriber() *MockSubscriber {
	return &MockSubscriber{}
}

func (m *MockSubscriber) Subscribe(ctx context.Context, topic string, handler events.HandlerFunc) error {
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(ctx, topic, handler)
	}
	return nil
}

// MockOrderRepo is a mock implementation of OrderRepo for testing
type MockOrderRepo struct {
	mu      sync.RWMutex
	orders  map[uuid.UUID]*Order
	CreateFunc func(ctx context.Context, order *Order) error
	GetFunc    func(ctx context.Context, id uuid.UUID) (*Order, error)
	SaveFunc   func(ctx context.Context, order *Order) error
	DeleteFunc func(ctx context.Context, id uuid.UUID) error
}

func NewMockOrderRepo() *MockOrderRepo {
	return &MockOrderRepo{
		orders: make(map[uuid.UUID]*Order),
	}
}

func (m *MockOrderRepo) Create(ctx context.Context, order *Order) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, order)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[order.ID] = order
	return nil
}

func (m *MockOrderRepo) Get(ctx context.Context, id uuid.UUID) (*Order, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	order, ok := m.orders[id]
	if !ok {
		return nil, fmt.Errorf("order not found")
	}
	return order, nil
}

func (m *MockOrderRepo) List(ctx context.Context) ([]*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Order
	for _, o := range m.orders {
		result = append(result, o)
	}
	return result, nil
}

func (m *MockOrderRepo) ListByTable(ctx context.Context, tableID uuid.UUID) ([]*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Order
	for _, o := range m.orders {
		if o.TableID == tableID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (m *MockOrderRepo) ListByStatus(ctx context.Context, status string) ([]*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Order
	for _, o := range m.orders {
		if o.Status == status {
			result = append(result, o)
		}
	}
	return result, nil
}

func (m *MockOrderRepo) Save(ctx context.Context, order *Order) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, order)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[order.ID] = order
	return nil
}

func (m *MockOrderRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.orders, id)
	return nil
}

// MockOrderItemRepo is a mock implementation of OrderItemRepo for testing
type MockOrderItemRepo struct {
	mu    sync.RWMutex
	items map[uuid.UUID]*OrderItem
	CreateFunc func(ctx context.Context, item *OrderItem) error
	GetFunc    func(ctx context.Context, id uuid.UUID) (*OrderItem, error)
	SaveFunc   func(ctx context.Context, item *OrderItem) error
	DeleteFunc func(ctx context.Context, id uuid.UUID) error
}

func NewMockOrderItemRepo() *MockOrderItemRepo {
	return &MockOrderItemRepo{
		items: make(map[uuid.UUID]*OrderItem),
	}
}

func (m *MockOrderItemRepo) Create(ctx context.Context, item *OrderItem) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, item)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = item
	return nil
}

func (m *MockOrderItemRepo) Get(ctx context.Context, id uuid.UUID) (*OrderItem, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("order item not found")
	}
	return item, nil
}

func (m *MockOrderItemRepo) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]*OrderItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*OrderItem
	for _, item := range m.items {
		if item.OrderID == orderID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (m *MockOrderItemRepo) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]*OrderItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*OrderItem
	for _, item := range m.items {
		if item.GroupID != nil && *item.GroupID == groupID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (m *MockOrderItemRepo) Save(ctx context.Context, item *OrderItem) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, item)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = item
	return nil
}

func (m *MockOrderItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}

// MockOrderGroupRepo is a mock implementation of OrderGroupRepo for testing
type MockOrderGroupRepo struct {
	mu     sync.RWMutex
	groups map[uuid.UUID]*OrderGroup
	CreateFunc func(ctx context.Context, group *OrderGroup) error
	DeleteFunc func(ctx context.Context, id uuid.UUID) error
}

func NewMockOrderGroupRepo() *MockOrderGroupRepo {
	return &MockOrderGroupRepo{
		groups: make(map[uuid.UUID]*OrderGroup),
	}
}

func (m *MockOrderGroupRepo) Create(ctx context.Context, group *OrderGroup) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, group)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups[group.ID] = group
	return nil
}

func (m *MockOrderGroupRepo) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]*OrderGroup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*OrderGroup
	for _, g := range m.groups {
		if g.OrderID == orderID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *MockOrderGroupRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.groups, id)
	return nil
}
