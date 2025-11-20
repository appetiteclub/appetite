package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/appetiteclub/appetite/services/kitchen/internal/events"
	"github.com/appetiteclub/appetite/services/kitchen/internal/kitchen"
	"github.com/appetiteclub/appetite/services/kitchen/internal/mongo"
	"github.com/aquamarinepk/aqm"
	aqmevents "github.com/aquamarinepk/aqm/events"
	"github.com/aquamarinepk/aqm/middleware"
)

const (
	appNamespace = "KITCHEN"
	appName      = "kitchen"
	appVersion   = "0.1.0"
)

func main() {
	config, err := aqm.LoadConfig(appNamespace, os.Args[1:])
	if err != nil {
		log.Fatalf("Cannot setup %s(%s): %v", appName, appVersion, err)
	}

	logLevel, _ := config.GetString("log.level")
	logger := aqm.NewLogger(logLevel)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	ticketRepo := mongo.NewTicketRepo(config, logger)

	natsURL, _ := config.GetString("nats.url")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Initialize NATS Stream (JetStream) for persistent event publishing
	var kitchenStream *pkg.NATSStream
	var orderSubscriber *pkg.NATSSubscriber
	streamEnabled, _ := config.GetString("nats.stream.enabled")

	if streamEnabled == "true" && natsURL != "" {
		// Create Stream for kitchen.tickets (persistent)
		streamCfg := pkg.NATSStreamConfig{
			URL:          natsURL,
			StreamName:   "KITCHEN_EVENTS",
			Topic:        "kitchen.tickets",
			ConsumerName: "kitchen-publisher", // Not used for publishing, but required
			MaxAge:       24 * time.Hour,
			MaxMsgs:      0,
		}
		kitchenStream, err = pkg.NewNATSStream(streamCfg)
		if err != nil {
			log.Fatalf("Cannot initialize NATS stream: %v", err)
		}
		logger.Info("NATS stream initialized for persistent events")

		// Use separate subscriber for orders.items (no persistence needed)
		orderSubscriber, err = pkg.NewNATSSubscriber(natsURL)
		if err != nil {
			log.Fatalf("Cannot connect to NATS subscriber: %v", err)
		}
	} else {
		// Fallback to legacy NATS Core (non-persistent)
		publisher, err := pkg.NewNATSPublisher(natsURL)
		if err != nil {
			log.Fatalf("Cannot connect to NATS publisher: %v", err)
		}
		kitchenStream = nil // Will use publisher as Publisher interface

		orderSubscriber, err = pkg.NewNATSSubscriber(natsURL)
		if err != nil {
			log.Fatalf("Cannot connect to NATS subscriber: %v", err)
		}

		// Wrap publisher to use as Stream (for compatibility)
		// This allows handler to always use Stream interface
		_ = publisher // Keep for now
	}

	// Event subscriber consumes orders.items and publishes to kitchen.tickets
	var eventPublisher aqmevents.Publisher
	if kitchenStream != nil {
		eventPublisher = kitchenStream
	} else {
		// Fallback: create basic publisher
		pub, err := pkg.NewNATSPublisher(natsURL)
		if err != nil {
			log.Fatalf("Cannot create fallback publisher: %v", err)
		}
		eventPublisher = pub
	}

	// Initialize ticket cache with Stream (preferred) or MongoDB fallback
	// Convert typed nil pointer to actual nil interface to avoid panic
	var streamForCache aqmevents.StreamConsumer
	if kitchenStream != nil {
		streamForCache = kitchenStream
	}
	// If kitchenStream was nil, streamForCache stays nil (proper nil interface)
	ticketCache := kitchen.NewTicketStateCache(streamForCache, ticketRepo, logger)

	eventSubscriber := events.NewOrderItemSubscriber(orderSubscriber, ticketRepo, ticketCache, eventPublisher, logger)

	// Handler uses Stream for publishing ticket events and cache for reads
	handler := kitchen.NewHandler(ticketRepo, ticketCache, eventPublisher, config, logger)

	// Initialize gRPC streaming server for real-time events
	grpcStreamServer := kitchen.NewEventStreamServer(ticketCache, logger)
	ticketCache.SetStreamServer(grpcStreamServer)

	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      logger,
		DisableCORS: true,
	})
	stack = append(stack, middleware.InternalOnly())

	// Setup lifecycle hooks
	lifecycles := []interface{}{ticketRepo, eventSubscriber}

	// Warm cache after repo is started
	cacheLifecycle := aqm.LifecycleHooks{
		OnStart: func(ctx context.Context) error {
			if err := ticketCache.Warm(ctx); err != nil {
				logger.Info("failed to warm ticket cache", "error", err)
			}
			return nil
		},
	}
	lifecycles = append(lifecycles, cacheLifecycle)

	if kitchenStream != nil {
		streamLifecycle := aqm.LifecycleHooks{
			OnStop: func(context.Context) error { return kitchenStream.Close() },
		}
		lifecycles = append(lifecycles, streamLifecycle)
	}
	if orderSubscriber != nil {
		subscriberLifecycle := aqm.LifecycleHooks{
			OnStop: func(context.Context) error { return orderSubscriber.Close() },
		}
		lifecycles = append(lifecycles, subscriberLifecycle)
	}

	options := []aqm.Option{
		aqm.WithConfig(config),
		aqm.WithLogger(logger),
		aqm.WithHTTPMiddleware(stack...),
		aqm.WithHTTPServerModules("web.port", handler),
		aqm.WithGRPCServerModules("grpc.port", grpcStreamServer),
		aqm.WithLifecycle(lifecycles...),
		aqm.WithHealthChecks(appName),
	}

	ms := aqm.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
