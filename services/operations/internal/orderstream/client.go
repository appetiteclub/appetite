package orderstream

import (
	"context"
	"io"
	"sync"
	"time"

	proto "github.com/appetiteclub/appetite/services/operations/internal/orderstream/proto"
	"github.com/aquamarinepk/aqm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client manages connection to Order gRPC stream and broadcasts to SSE subscribers
type Client struct {
	addr   string
	logger aqm.Logger

	mu          sync.RWMutex
	subscribers map[string]chan *proto.OrderItemEvent
	conn        *grpc.ClientConn
	stream      proto.OrderEventStream_StreamOrderItemEventsClient
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewClient creates a new Order stream client
func NewClient(addr string, logger aqm.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		addr:        addr,
		logger:      logger,
		subscribers: make(map[string]chan *proto.OrderItemEvent),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start connects to Order gRPC stream and starts broadcasting
func (c *Client) Start(ctx context.Context) error {
	c.logger.Info("starting Order stream client", "addr", c.addr)

	// Start connection attempt in background - don't block startup
	go c.connectWithRetry()

	return nil
}

// connectWithRetry attempts to connect with exponential backoff
func (c *Client) connectWithRetry() {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Order stream client shutdown, stopping connection attempts")
			return
		default:
		}

		c.logger.Info("attempting to connect to Order gRPC stream", "addr", c.addr)

		conn, err := grpc.NewClient(c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			c.logger.Error("failed to create gRPC client", "error", err, "retry_in", backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		c.conn = conn
		client := proto.NewOrderEventStreamClient(conn)

		// Subscribe to all events (no filters)
		req := &proto.SubscribeOrderItemEventsRequest{
			TableId: "",
			OrderId: "",
		}

		stream, err := client.StreamOrderItemEvents(c.ctx, req)
		if err != nil {
			c.logger.Error("failed to subscribe to Order events", "error", err, "retry_in", backoff)
			conn.Close()
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		c.stream = stream
		c.logger.Info("connected to Order gRPC stream successfully")

		// Reset backoff on successful connection
		backoff = 1 * time.Second

		// Start receiving events - blocks until disconnect
		c.receiveEvents()
	}
}

// receiveEvents receives events from gRPC stream and broadcasts to SSE subscribers
func (c *Client) receiveEvents() {
	for {
		evt, err := c.stream.Recv()
		if err == io.EOF {
			c.logger.Info("Order gRPC stream closed (EOF)")
			return
		}
		if err != nil {
			c.logger.Error("error receiving from Order stream", "error", err)
			return
		}

		// Broadcast to all SSE subscribers
		c.broadcastToSubscribers(evt)
	}
}

// broadcastToSubscribers sends event to all SSE subscribers
func (c *Client) broadcastToSubscribers(evt *proto.OrderItemEvent) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for subscriberID, ch := range c.subscribers {
		select {
		case ch <- evt:
			// Event sent successfully
		default:
			// Channel full, subscriber too slow - skip this event
			c.logger.Info("subscriber channel full, dropping event", "subscriber_id", subscriberID)
		}
	}
}

// Subscribe adds a new SSE subscriber and returns event channel
func (c *Client) Subscribe(subscriberID string) <-chan *proto.OrderItemEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan *proto.OrderItemEvent, 100)
	c.subscribers[subscriberID] = ch

	c.logger.Info("new SSE subscriber for Order events", "subscriber_id", subscriberID, "total_subscribers", len(c.subscribers))

	return ch
}

// Unsubscribe removes an SSE subscriber
func (c *Client) Unsubscribe(subscriberID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ch, ok := c.subscribers[subscriberID]; ok {
		close(ch)
		delete(c.subscribers, subscriberID)
		c.logger.Info("SSE subscriber for Order events disconnected", "subscriber_id", subscriberID, "total_subscribers", len(c.subscribers))
	}
}

// Stop closes connection to Order gRPC stream
func (c *Client) Stop(ctx context.Context) error {
	c.logger.Info("stopping Order stream client")

	c.cancel()

	// Close all subscriber channels
	c.mu.Lock()
	for id, ch := range c.subscribers {
		close(ch)
		delete(c.subscribers, id)
	}
	c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}
