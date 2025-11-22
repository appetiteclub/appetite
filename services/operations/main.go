package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/middleware"
	aqmtemplate "github.com/aquamarinepk/aqm/template"

	"github.com/appetiteclub/appetite/services/operations/internal/kitchenstream"
	"github.com/appetiteclub/appetite/services/operations/internal/operations"
	"github.com/appetiteclub/appetite/services/operations/internal/orderstream"
)

//go:embed assets
var assetsFS embed.FS

const (
	appNamespace = "OPERATIONS"
	appName      = "operations"
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

	// Initialize template manager
	tmplMgr := aqmtemplate.NewManager(assetsFS, aqmtemplate.WithLogger(logger))

	// Initialize AuthZ repos
	roleRepo, err := operations.NewAPIRoleRepo(config, logger)
	if err != nil {
		log.Fatalf("cannot initialize role repo: %v", err)
	}

	grantRepo, err := operations.NewAPIGrantRepo(config, logger)
	if err != nil {
		log.Fatalf("cannot initialize grant repo: %v", err)
	}

	// Initialize Kitchen service HTTP client
	kitchenURL, _ := config.GetString("services.kitchen.url")
	var kitchenDA *operations.KitchenDataAccess
	if kitchenURL != "" {
		kitchenClient := aqm.NewServiceClient(kitchenURL)
		kitchenDA = operations.NewKitchenDataAccess(kitchenClient)
	}

	// Initialize handler (HTTP-only - no NATS, no cache, no events)
	handler := operations.NewHandler(tmplMgr, roleRepo, grantRepo, kitchenDA, config, logger)

	// Initialize Kitchen gRPC stream client
	kitchenGRPCAddr, _ := config.GetString("services.kitchen.grpc_addr")
	kitchenStreamClient := kitchenstream.NewClient(kitchenGRPCAddr, logger)

	// Initialize Order gRPC stream client
	orderGRPCAddr, _ := config.GetString("services.order.grpc_addr")
	orderStreamClient := orderstream.NewClient(orderGRPCAddr, logger)

	// Create adapter for order item data access
	orderItemAdapter := operations.NewOrderItemAdapter(handler.GetOrderDataAccess())

	// Initialize SSE handler for Kitchen and Order events
	sseHandler := kitchenstream.NewSSEHandler(kitchenStreamClient, logger, tmplMgr, orderItemAdapter)
	sseHandler.SetOrderClient(orderStreamClient)

	// Add SSE endpoint to handler
	handler.SetSSEHandler(sseHandler)

	// Configure middleware stack (no InternalOnly - this is public-facing)
	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      logger,
		DisableCORS: false, // Enable CORS for operations service
	})

	lifecycles := []interface{}{tmplMgr, kitchenStreamClient, orderStreamClient}

	options := []aqm.Option{
		aqm.WithConfig(config),
		aqm.WithLogger(logger),
		aqm.WithHTTPMiddleware(stack...),
		aqm.WithHTTPServerModules("web.port", handler),
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
