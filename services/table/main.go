package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/appetiteclub/appetite/pkg"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/middleware"

	"github.com/appetiteclub/appetite/services/table/internal/mongo"
	"github.com/appetiteclub/appetite/services/table/internal/tables"
)

//go:embed seed.json
var seedFS embed.FS

const (
	appNamespace = "TABLE"
	appName      = "table"
	appVersion   = "0.1.0"
)

func main() {
	config, err := aqm.LoadConfig(appNamespace, os.Args[1:])
	if err != nil {
		log.Fatalf("%s(%s) cannot setup with error: %v", appName, appVersion, err)
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

	seedCtx, cancelSeeds := context.WithCancel(ctx)
	defer cancelSeeds()

	var lifecycle []aqm.LifecycleHooks

	tableRepo := mongo.NewTableRepo(config, logger)
	err = tableRepo.Start(ctx)
	if err != nil {
		log.Fatalf("%s(%s) cannot start table repositoryr: %v", appName, appVersion, err)
	}

	lifecycle = append(lifecycle, aqm.LifecycleHooks{})

	db := tableRepo.GetDatabase()
	if db == nil {
		err := errors.New("cannot get table repo database")
		log.Fatalf("%s(%s) cannot initialize database: %v", appName, appVersion, err)
	}

	groupRepo := mongo.NewGroupRepo(db)
	orderRepo := mongo.NewOrderRepo(db)
	orderItemRepo := mongo.NewOrderItemRepo(db)
	reservationRepo := mongo.NewReservationRepo(db)

	natsURL := config.GetStringOrDef("nats.url", "nats://localhost:4222")

	publisher, err := pkg.NewNATSPublisher(natsURL)
	if err != nil {
		log.Fatalf("%s(%s) cannot connect to NATS publisher: %v", appName, appVersion, err)
	}

	publisherLifecycle := aqm.LifecycleHooks{
		OnStop: func(context.Context) error {
			return publisher.Close()
		},
	}
	lifecycle = append(lifecycle, publisherLifecycle)

	repos := tables.Repos{
		TableRepo:       tableRepo,
		GroupRepo:       groupRepo,
		OrderRepo:       orderRepo,
		OrderItemRepo:   orderItemRepo,
		ReservationRepo: reservationRepo,
	}

	hd := tables.HandlerDeps{
		Repos:     repos,
		Publisher: publisher,
	}

	handler := tables.NewHandler(
		hd,
		config,
		logger,
	)

	seedHooks := aqm.LifecycleHooks{
		OnStart: tables.SeedingFunc(seedCtx, tableRepo, seedFS, logger),
		OnStop:  tables.StopFunc(cancelSeeds),
	}
	lifecycle = append(lifecycle, seedHooks)

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
		aqm.WithLifecycle(lifecycle),
		aqm.WithHealthChecks(appName),
	}

	ms := aqm.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		_ = tableRepo.Stop(context.Background())
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
