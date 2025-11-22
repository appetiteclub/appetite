package app

import (
	"context"
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
	AppName    = "kitchen"
	AppVersion = "0.1.0"
)

// App encapsulates the kitchen service application
type App struct {
	config     *aqm.Config
	logger     aqm.Logger
	micro      *aqm.Micro
	ticketRepo *mongo.TicketRepo
}

// New creates a new kitchen service application
func New(config *aqm.Config, logger aqm.Logger) (*App, error) {
	return &App{
		config: config,
		logger: logger,
	}, nil
}

// Initialize sets up all dependencies and components
func (a *App) Initialize(ctx context.Context) error {
	// Initialize ticket repository
	a.ticketRepo = mongo.NewTicketRepo(a.config, a.logger)

	// Apply demo seeds if enabled
	if err := kitchen.ApplyDemoSeeds(ctx, a.config, a.ticketRepo.GetDatabase, a.logger); err != nil {
		a.logger.Errorf("Demo seeding failed (non-fatal): %v", err)
	}

	// Initialize NATS
	natsURL, _ := a.config.GetString("nats.url")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Initialize NATS Stream or Publisher
	var kitchenStream *pkg.NATSStream
	var orderSubscriber *pkg.NATSSubscriber
	var eventPublisher aqmevents.Publisher

	streamEnabled, _ := a.config.GetString("nats.stream.enabled")
	if streamEnabled == "true" && natsURL != "" {
		// Create Stream for kitchen.tickets (persistent)
		streamCfg := pkg.NATSStreamConfig{
			URL:          natsURL,
			StreamName:   "KITCHEN_EVENTS",
			Topic:        "kitchen.tickets",
			ConsumerName: "kitchen-publisher",
			MaxAge:       24 * time.Hour,
			MaxMsgs:      0,
		}
		var err error
		kitchenStream, err = pkg.NewNATSStream(streamCfg)
		if err != nil {
			return err
		}
		a.logger.Info("NATS stream initialized for persistent events")
		eventPublisher = kitchenStream

		// Use separate subscriber for orders.items
		orderSubscriber, err = pkg.NewNATSSubscriber(natsURL)
		if err != nil {
			return err
		}
	} else {
		// Fallback to legacy NATS Core
		publisher, err := pkg.NewNATSPublisher(natsURL)
		if err != nil {
			return err
		}
		eventPublisher = publisher

		orderSubscriber, err = pkg.NewNATSSubscriber(natsURL)
		if err != nil {
			return err
		}
	}

	// Initialize ticket cache
	var streamForCache aqmevents.StreamConsumer
	if kitchenStream != nil {
		streamForCache = kitchenStream
	}
	ticketCache := kitchen.NewTicketStateCache(streamForCache, a.ticketRepo, a.logger)

	// Initialize event subscriber
	eventSubscriber := events.NewOrderItemSubscriber(orderSubscriber, a.ticketRepo, ticketCache, eventPublisher, a.logger)

	// Initialize HTTP handler
	handler := kitchen.NewHandler(a.ticketRepo, ticketCache, eventPublisher, a.config, a.logger)

	// Initialize gRPC streaming server
	grpcStreamServer := kitchen.NewEventStreamServer(ticketCache, a.logger)
	ticketCache.SetStreamServer(grpcStreamServer)

	// Setup middleware
	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      a.logger,
		DisableCORS: true,
	})
	stack = append(stack, middleware.InternalOnly())

	// Setup lifecycle hooks
	lifecycles := []interface{}{a.ticketRepo, eventSubscriber}

	// Warm cache after repo is started
	cacheLifecycle := aqm.LifecycleHooks{
		OnStart: func(ctx context.Context) error {
			if err := ticketCache.Warm(ctx); err != nil {
				a.logger.Info("failed to warm ticket cache", "error", err)
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

	// Build micro service
	options := []aqm.Option{
		aqm.WithConfig(a.config),
		aqm.WithLogger(a.logger),
		aqm.WithHTTPMiddleware(stack...),
		aqm.WithHTTPServerModules("web.port", handler),
		aqm.WithGRPCServerModules("grpc.port", grpcStreamServer),
		aqm.WithLifecycle(lifecycles...),
		aqm.WithHealthChecks(AppName),
	}

	a.micro = aqm.NewMicro(options...)
	return nil
}

// Run starts the application
func (a *App) Run(ctx context.Context) error {
	a.logger.Infof("Starting %s(%s)", AppName, AppVersion)
	if err := a.micro.Run(ctx); err != nil {
		return err
	}
	a.logger.Infof("%s(%s) stopped", AppName, AppVersion)
	return nil
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown(ctx context.Context) error {
	// Lifecycle cleanup is handled by aqm.Micro
	return nil
}
