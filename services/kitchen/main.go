package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/appetiteclub/appetite/services/kitchen/internal/events"
	"github.com/appetiteclub/appetite/services/kitchen/internal/kitchen"
	"github.com/appetiteclub/appetite/services/kitchen/internal/mongo"
	"github.com/aquamarinepk/aqm"
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

	publisher, err := pkg.NewNATSPublisher(natsURL)
	if err != nil {
		log.Fatalf("Cannot connect to NATS publisher: %v", err)
	}

	subscriber, err := pkg.NewNATSSubscriber(natsURL)
	if err != nil {
		log.Fatalf("Cannot connect to NATS subscriber: %v", err)
	}

	eventSubscriber := events.NewOrderItemSubscriber(subscriber, ticketRepo, publisher, logger)

	handler := kitchen.NewHandler(ticketRepo, publisher, config, logger)

	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      logger,
		DisableCORS: true,
	})
	stack = append(stack, middleware.InternalOnly())

	options := []aqm.Option{
		aqm.WithConfig(config),
		aqm.WithLogger(logger),
		aqm.WithHTTPMiddleware(stack...),
		aqm.WithHTTPServerModules("web.port", handler),
		aqm.WithLifecycle(ticketRepo, eventSubscriber),
		aqm.WithHealthChecks(appName),
	}

	ms := aqm.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
