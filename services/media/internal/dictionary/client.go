package dictionary

import (
	"context"

	"github.com/google/uuid"
)

type Client interface {
	EnsureCategory(ctx context.Context, id uuid.UUID) error
	EnsureTags(ctx context.Context, ids []uuid.UUID) error
}

type NoopClient struct{}

func NewNoopClient() *NoopClient {
	return &NoopClient{}
}

func (c *NoopClient) EnsureCategory(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (c *NoopClient) EnsureTags(ctx context.Context, ids []uuid.UUID) error {
	return nil
}
