package main

import (
	"context"
	"errors"
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
	tableClient := aqm.NewServiceClient(tableURL)
	tableStateCache := order.NewTableStateCache(tableClient, logger)
	tableStatusSub := order.NewTableStatusSubscriber(sub, tableStateCache, logger)

	// Kitchen service client for updating tickets when order items change
	kitchenURL := config.GetStringOrDef("services.kitchen.url", "")
	if kitchenURL == "" {
		log.Fatalf("Cannot create kitchen service client: %v", err)
	}
	kitchenClient := aqm.NewServiceClient(kitchenURL)

	// Initialize gRPC streaming server for real-time order item events
	orderEvents := order.NewOrderEventStreamServer(orderItemRepo, logger)

	// Subscribe to kitchen ticket events to sync OrderItem status
	kitchenSub := order.NewKitchenTicketSubscriber(sub, orderItemRepo, logger)
	kitchenSub.SetStreamServer(orderEvents)

	publisherLifecycle := aqm.LifecycleHooks{
		OnStop: func(context.Context) error {
			return pub.Close()
		},
	}

	subLifecycle := aqm.LifecycleHooks{
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
		aqm.WithGRPCServerModules("grpc.port", orderEvents),
		aqm.WithLifecycle(aqm.LifecycleHooks{OnStop: baseRepo.Stop}, tableStatusSub, kitchenSub, publisherLifecycle, subLifecycle),
		aqm.WithHealthChecks(appName),
	}

	ms := aqm.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	err = ms.Run(ctx)
	if err != nil {
		_ = baseRepo.Stop(context.Background())
		log.Fatalf("%s(%s) stopped: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
