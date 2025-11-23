package main

import (
	"context"
	"errors"
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
		log.Fatalf("%s(%s) cannot setup: %v", appName, appVersion, err)
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

	natsURL := config.GetStringOrDef("nats.url", "nats://localhost:4222")

	// Initialize NATS Stream (JetStream) for persistent event publishing
	var kitchenStream *pkg.NATSStream
	var orderSubscriber *pkg.NATSSubscriber
	streamEnabled := config.GetBoolOrFalse("nats.stream.enabled")
	if !streamEnabled {
		log.Fatalf("%s(%s) NATS stream should be enabled: %v", appName, appVersion, errors.New("nats stream disabled"))
	}

	streamCfg := pkg.NATSStreamConfig{
		URL:          natsURL,
		StreamName:   "KITCHEN_EVENTS",
		Topic:        "kitchen.tickets",
		ConsumerName: "kitchen-publisher",
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

	// Initialize ticket cache with Stream (required)
	ticketCache := kitchen.NewTicketStateCache(kitchenStream, ticketRepo, logger)

	eventSubscriber := events.NewOrderItemSubscriber(orderSubscriber, ticketRepo, ticketCache, eventPublisher, logger)

	hd := kitchen.HandlerDeps{
		Repo:      ticketRepo,
		Cache:     ticketCache,
		Publisher: eventPublisher,
	}

	// Handler uses Stream for publishing ticket events and cache for reads
	handler := kitchen.NewHandler(hd, config, logger)

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
			err := ticketCache.Warm(ctx)
			if err != nil {
				logger.Info("failed to warm ticket cache", "error", err)
			}
			return nil
		},
	}
	lifecycles = append(lifecycles, cacheLifecycle)

	// Setup demo seeding if enabled
	demoEnabled, _ := config.GetString("seeding.demo")
	if demoEnabled == "true" {
		logger.Info("Demo seeding enabled for kitchen service")
		seedCtx, cancelSeeds := context.WithCancel(ctx)
		defer cancelSeeds()

		// Note: db will be available when OnStart runs (after ticketRepo lifecycle starts)
		seedHooks := aqm.LifecycleHooks{
			OnStart: func(startCtx context.Context) error {
				db := ticketRepo.GetDatabase()
				if db == nil {
					logger.Info("Cannot run demo seeding: database not available")
					return nil
				}
				return kitchen.DemoSeedingFunc(seedCtx, ticketRepo, ticketCache, db, logger)(startCtx)
			},
			OnStop: func(context.Context) error {
				cancelSeeds()
				return nil
			},
		}
		lifecycles = append(lifecycles, seedHooks)
	}

	if kitchenStream != nil {
		streamLifecycle := aqm.LifecycleHooks{
			OnStop: func(context.Context) error {
				return kitchenStream.Close()
			},
		}
		lifecycles = append(lifecycles, streamLifecycle)
	}

	if orderSubscriber != nil {
		subscriberLifecycle := aqm.LifecycleHooks{
			OnStop: func(context.Context) error {
				return orderSubscriber.Close()
			},
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
		log.Fatalf("%s(%s) stopped: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
