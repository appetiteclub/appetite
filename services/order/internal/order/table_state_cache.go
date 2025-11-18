package order

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aquamarinepk/aqm"
	"github.com/google/uuid"
)

type TableStateCache struct {
	mu     sync.RWMutex
	state  map[uuid.UUID]string
	client *aqm.ServiceClient
	logger aqm.Logger
}

func NewTableStateCache(client *aqm.ServiceClient, logger aqm.Logger) *TableStateCache {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &TableStateCache{
		state:  make(map[uuid.UUID]string),
		client: client,
		logger: logger,
	}
}

func (c *TableStateCache) Warm(ctx context.Context) error {
	if c.client == nil {
		return nil
	}
	resp, err := c.client.List(ctx, "tables")
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	return c.ingestCollection(resp.Data)
}

func (c *TableStateCache) Ensure(ctx context.Context, id uuid.UUID) (string, error) {
	if id == uuid.Nil {
		return "", fmt.Errorf("invalid table id")
	}
	if status, ok := c.Get(id); ok {
		return status, nil
	}
	return c.Refresh(ctx, id)
}

func (c *TableStateCache) Refresh(ctx context.Context, id uuid.UUID) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("table cache uninitialized")
	}
	resp, err := c.client.Get(ctx, "tables", id.String())
	if err != nil {
		return "", fmt.Errorf("failed to fetch table %s: %w", id, err)
	}
	var dto tableStateDTO
	if err := rehydrate(resp.Data, &dto); err != nil {
		return "", fmt.Errorf("failed to decode table %s: %w", id, err)
	}
	idValue, parseErr := uuid.Parse(dto.ID)
	if parseErr != nil {
		return "", fmt.Errorf("invalid table id %s", dto.ID)
	}
	c.Set(idValue, dto.Status)
	return dto.Status, nil
}

func (c *TableStateCache) Get(id uuid.UUID) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status, ok := c.state[id]
	return status, ok
}

func (c *TableStateCache) Set(id uuid.UUID, status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state[id] = status
}

func (c *TableStateCache) ingestCollection(data interface{}) error {
	var records []tableStateDTO
	if err := rehydrate(data, &records); err != nil {
		return err
	}
	for _, record := range records {
		id, err := uuid.Parse(record.ID)
		if err != nil {
			c.logger.Debug("skipping invalid table id", "table_id", record.ID)
			continue
		}
		c.Set(id, record.Status)
	}
	return nil
}

type tableStateDTO struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func rehydrate(data interface{}, out interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, out)
}
