package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/middleware"

	"github.com/appetiteclub/appetite/services/order/internal/mongo"
	"github.com/appetiteclub/appetite/services/order/internal/order"
)

const (
	appNamespace = "ORDER"
	appName      = "order"
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

	baseRepo := mongo.NewBaseRepo(config, logger)
	if err := baseRepo.Start(ctx); err != nil {
		log.Fatalf("Cannot start base repository: %v", err)
	}

	db := baseRepo.GetDatabase()
	if db == nil {
		log.Fatalf("cannot initialize repository database")
	}

	orderRepo := mongo.NewOrderRepo(db)
	orderItemRepo := mongo.NewOrderItemRepo(db)

	natsURL, _ := config.GetString("nats.url")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	publisher, err := pkg.NewNATSPublisher(natsURL)
	if err != nil {
		log.Fatalf("Cannot connect to NATS publisher: %v", err)
	}

	subscriber, err := pkg.NewNATSSubscriber(natsURL)
	if err != nil {
		log.Fatalf("Cannot connect to NATS subscriber: %v", err)
	}

	tableURL, _ := config.GetString("services.table.url")
	tableClient := aqm.NewServiceClient(tableURL)
	tableStateCache := order.NewTableStateCache(tableClient, logger)
	tableStatusSubscriber := order.NewTableStatusSubscriber(subscriber, tableStateCache, logger)

	publisherLifecycle := aqm.LifecycleHooks{OnStop: func(context.Context) error { return publisher.Close() }}
	subscriberLifecycle := aqm.LifecycleHooks{OnStop: func(context.Context) error { return subscriber.Close() }}

	handler := order.NewHandler(
		orderRepo,
		orderItemRepo,
		logger,
		config,
		tableStateCache,
		publisher,
	)

	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      logger,
		DisableCORS: true, // Internal API service
	})
	// Defense-in-depth: restrict to internal networks only.
	// This complements (does not replace) network policies at the infrastructure level.
	stack = append(stack, middleware.InternalOnly())

	options := []aqm.Option{
		aqm.WithConfig(config),
		aqm.WithLogger(logger),
		aqm.WithHTTPMiddleware(stack...),
		aqm.WithHTTPServerModules("web.port", handler),
		aqm.WithLifecycle(aqm.LifecycleHooks{OnStop: baseRepo.Stop}, tableStatusSubscriber, publisherLifecycle, subscriberLifecycle),
		aqm.WithHealthChecks(appName),
	}

	ms := aqm.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		baseRepo.Stop(context.Background())
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
