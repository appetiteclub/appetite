package order

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/events"
	"github.com/google/uuid"
)

type TableStatusSubscriber struct {
	subscriber events.Subscriber
	cache      *TableStateCache
	logger     apt.Logger
}

func NewTableStatusSubscriber(sub events.Subscriber, cache *TableStateCache, logger apt.Logger) *TableStatusSubscriber {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}
	return &TableStatusSubscriber{
		subscriber: sub,
		cache:      cache,
		logger:     logger,
	}
}

func (s *TableStatusSubscriber) Start(ctx context.Context) error {
	s.logger.Info("starting table status subscriber", "topic", pkg.TableStatusTopic)
	if s.cache != nil {
		if err := s.cache.Warm(ctx); err != nil {
			s.logger.Info("table cache warmup failed", "error", err)
		}
	}
	if s.subscriber == nil {
		return fmt.Errorf("table status subscriber not configured")
	}
	return s.subscriber.Subscribe(ctx, pkg.TableStatusTopic, s.handleEvent)
}

func (s *TableStatusSubscriber) handleEvent(ctx context.Context, msg []byte) error {
	var event pkg.TableStatusEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		s.logger.Info("invalid table status event", "error", err)
		return nil
	}

	id, err := uuid.Parse(event.TableID)
	if err != nil {
		s.logger.Info("invalid table id in event", "table_id", event.TableID)
		return nil
	}

	s.cache.Set(id, event.Status)
	s.logger.Debug("table status updated", "table_id", id.String(), "status", event.Status)
	return nil
}
