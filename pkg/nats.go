package pkg

import (
	"context"
	"fmt"

	"github.com/appetiteclub/apt/events"
	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn *nats.Conn
}

func NewNATSPublisher(url string) (*NATSPublisher, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	return &NATSPublisher{conn: conn}, nil
}

func (p *NATSPublisher) Publish(ctx context.Context, topic string, msg []byte) error {
	return p.conn.Publish(topic, msg)
}

func (p *NATSPublisher) Close() error {
	p.conn.Close()
	return nil
}

type NATSSubscriber struct {
	conn *nats.Conn
}

func NewNATSSubscriber(url string) (*NATSSubscriber, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	return &NATSSubscriber{conn: conn}, nil
}

func (s *NATSSubscriber) Subscribe(ctx context.Context, topic string, handler events.HandlerFunc) error {
	_, err := s.conn.Subscribe(topic, func(msg *nats.Msg) {
		if err := handler(ctx, msg.Data); err != nil {
		}
	})
	return err
}

func (s *NATSSubscriber) Close() error {
	s.conn.Close()
	return nil
}
