package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/appetiteclub/apt/events"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATSStream implements events.Stream using NATS JetStream for persistent event streaming.
type NATSStream struct {
	conn     *nats.Conn
	js       jetstream.JetStream
	stream   jetstream.Stream
	consumer jetstream.Consumer
	topic    string
}

// NATSStreamConfig configures a NATSStream instance.
type NATSStreamConfig struct {
	URL          string        // NATS server URL
	StreamName   string        // JetStream stream name (e.g., "KITCHEN_EVENTS")
	Topic        string        // Subject/topic pattern (e.g., "kitchen.tickets")
	ConsumerName string        // Durable consumer name for this service
	MaxAge       time.Duration // How long to retain events (e.g., 24 hours)
	MaxMsgs      int64         // Maximum number of messages to retain (0 = unlimited)
}

// NewNATSStream creates a new NATSStream and ensures the stream and consumer exist.
func NewNATSStream(cfg NATSStreamConfig) (*NATSStream, error) {
	// Connect to NATS
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create or update stream
	streamConfig := jetstream.StreamConfig{
		Name:     cfg.StreamName,
		Subjects: []string{cfg.Topic},
		MaxAge:   cfg.MaxAge,
	}
	if cfg.MaxMsgs > 0 {
		streamConfig.MaxMsgs = cfg.MaxMsgs
	}

	stream, err := js.CreateOrUpdateStream(context.Background(), streamConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create/update stream %s: %w", cfg.StreamName, err)
	}

	// Create or update durable consumer for replay
	consumerConfig := jetstream.ConsumerConfig{
		Name:          cfg.ConsumerName,
		Durable:       cfg.ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy, // Replay from beginning
		FilterSubject: cfg.Topic,
	}

	consumer, err := stream.CreateOrUpdateConsumer(context.Background(), consumerConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create/update consumer %s: %w", cfg.ConsumerName, err)
	}

	return &NATSStream{
		conn:     conn,
		js:       js,
		stream:   stream,
		consumer: consumer,
		topic:    cfg.Topic,
	}, nil
}

// Publish publishes a message to the stream.
func (s *NATSStream) Publish(ctx context.Context, topic string, msg []byte) error {
	_, err := s.js.Publish(ctx, topic, msg)
	if err != nil {
		return fmt.Errorf("failed to publish to stream: %w", err)
	}
	return nil
}

// Fetch retrieves up to limit messages from the stream (for replay on startup).
func (s *NATSStream) Fetch(ctx context.Context, limit int) ([]events.StreamMessage, error) {
	if limit <= 0 {
		limit = 1000 // Default batch size
	}

	// Fetch messages from consumer
	msgBatch, err := s.consumer.Fetch(limit, jetstream.FetchMaxWait(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	var messages []events.StreamMessage
	for msg := range msgBatch.Messages() {
		metadata, err := msg.Metadata()
		if err != nil {
			// Skip messages with invalid metadata
			msg.Ack()
			continue
		}

		messages = append(messages, events.StreamMessage{
			Data:      msg.Data(),
			Sequence:  metadata.Sequence.Stream,
			Timestamp: metadata.Timestamp.UnixNano(),
		})

		// Acknowledge the message
		msg.Ack()
	}

	return messages, nil
}

// SubscribeStream subscribes to new messages arriving on the stream (real-time).
func (s *NATSStream) SubscribeStream(ctx context.Context, handler events.HandlerFunc) error {
	_, err := s.consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(ctx, msg.Data()); err != nil {
			// TODO: Add logging for handler errors
			msg.Nak() // Negative acknowledge - redelivery
		} else {
			msg.Ack()
		}
	})
	return err
}

// Subscribe implements events.Subscriber interface.
// For streams, topic is ignored (already configured in consumer).
func (s *NATSStream) Subscribe(ctx context.Context, topic string, handler events.HandlerFunc) error {
	// Topic is ignored for streams - consumer is already bound to specific subject
	return s.SubscribeStream(ctx, handler)
}

// Close closes the NATS connection.
func (s *NATSStream) Close() error {
	s.conn.Close()
	return nil
}
