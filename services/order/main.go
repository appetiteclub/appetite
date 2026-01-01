package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/middleware"

	"github.com/appetiteclub/appetite/services/order/internal/mongo"
	"github.com/appetiteclub/appetite/services/order/internal/order"
)

const (
	appNamespace = "ORDER"
	appName      = "order"
	appVersion   = "0.1.0"
)

func main() {
	config, err := apt.LoadConfig(appNamespace, os.Args[1:])
	if err != nil {
		log.Fatalf("%s(%s) cannot setup: %v", appName, appVersion, err)
	}

	logLevel, _ := config.GetString("log.level")
	logger := apt.NewLogger(logLevel)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	seedCtx, cancelSeeds := context.WithCancel(ctx)
	defer cancelSeeds()

	baseRepo := mongo.NewBaseRepo(config, logger)
	err = baseRepo.Start(ctx)
	if err != nil {
		log.Fatalf("%s(%s) cannot start base repository: %v", appName, appVersion, err)
	}

	db := baseRepo.GetDatabase()
	if db == nil {
		log.Fatalf("%s(%s) cannot initialize repository database: %v", appName, appVersion, errors.New("repository database is nil"))
	}

	orderRepo := mongo.NewOrderRepo(db)
	orderItemRepo := mongo.NewOrderItemRepo(db)
	orderGroupRepo := mongo.NewOrderGroupRepo(db)

	repos := order.Repos{
		OrderRepo:      orderRepo,
		OrderItemRepo:  orderItemRepo,
		OrderGroupRepo: orderGroupRepo,
	}

	natsURL := config.GetStringOrDef("nats.url", "nats://localhost:4222")

	pub, err := pkg.NewNATSPublisher(natsURL)
	if err != nil {
		log.Fatalf("%s(%s) cannot connect to NATS publisher: %v", appName, appVersion, err)
	}

	sub, err := pkg.NewNATSSubscriber(natsURL)
	if err != nil {
		log.Fatalf("%s(%s) cannot connect to NATS subscriber: %v", appName, appVersion, err)
	}

	tableURL, _ := config.GetString("services.table.url")
	tableClient := apt.NewServiceClient(tableURL)
	tableStateCache := order.NewTableStateCache(tableClient, logger)
	tableStatusSub := order.NewTableStatusSubscriber(sub, tableStateCache, logger)

	// Kitchen service client for updating tickets when order items change
	kitchenURL := config.GetStringOrDef("services.kitchen.url", "")
	if kitchenURL == "" {
		log.Fatalf("Cannot create kitchen service client: %v", err)
	}
	kitchenClient := apt.NewServiceClient(kitchenURL)

	// Initialize gRPC streaming server for real-time order item events
	orderEvents := order.NewOrderEventStreamServer(orderItemRepo, logger)

	// Subscribe to kitchen ticket events to sync OrderItem status
	kitchenSub := order.NewKitchenTicketSubscriber(sub, orderItemRepo, logger)
	kitchenSub.SetStreamServer(orderEvents)

	publisherLifecycle := apt.LifecycleHooks{
		OnStop: func(context.Context) error {
			return pub.Close()
		},
	}

	subLifecycle := apt.LifecycleHooks{
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	}

	hd := order.HandlerDeps{
		Repos:             repos,
		TableStatesCache:  tableStateCache,
		KitchenClient:     kitchenClient,
		Publisher:         pub,
		OrderStreamServer: orderEvents,
	}

	handler := order.NewHandler(hd, config, logger)

	// Setup demo seeding if enabled
	demoEnabled, _ := config.GetString("seeding.demo")
	var seedHooks apt.LifecycleHooks
	if demoEnabled == "true" {
		logger.Info("Demo seeding enabled for order service")
		seedHooks = apt.LifecycleHooks{
			OnStart: order.DemoSeedingFunc(seedCtx, repos, db, logger),
			OnStop: func(context.Context) error {
				cancelSeeds()
				return nil
			},
		}
	}

	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      logger,
		DisableCORS: true, // Internal API service
	})

	// Defense-in-depth: restrict to internal networks only.
	// This complements (does not replace) network policies at the infrastructure level.
	stack = append(stack, middleware.InternalOnly())

	// Build lifecycle hooks
	lifecycles := []interface{}{
		apt.LifecycleHooks{OnStop: baseRepo.Stop},
		tableStatusSub,
		kitchenSub,
		publisherLifecycle,
		subLifecycle,
	}
	if demoEnabled == "true" {
		lifecycles = append(lifecycles, seedHooks)
	}

	options := []apt.Option{
		apt.WithConfig(config),
		apt.WithLogger(logger),
		apt.WithHTTPMiddleware(stack...),
		apt.WithHTTPServerModules("web.port", handler),
		apt.WithGRPCServerModules("grpc.port", orderEvents),
		apt.WithLifecycle(lifecycles...),
		apt.WithHealthChecks(appName),
	}

	ms := apt.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	err = ms.Run(ctx)
	if err != nil {
		_ = baseRepo.Stop(context.Background())
		log.Fatalf("%s(%s) stopped: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
