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
	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/middleware"

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
	config, err := apt.LoadConfig(appNamespace, os.Args[1:])
	if err != nil {
		log.Fatalf("%s(%s) cannot setup with error: %v", appName, appVersion, err)
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

	lifecycle := []interface{}{}

	tableRepo := mongo.NewTableRepo(config, logger)
	err = tableRepo.Start(ctx)
	if err != nil {
		log.Fatalf("%s(%s) cannot start table repositoryr: %v", appName, appVersion, err)
	}

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

	publisherLifecycle := apt.LifecycleHooks{
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

	// Choose seeding strategy based on config
	demoEnabled, _ := config.GetString("seeding.demo")
	var seedingFunc func(ctx context.Context) error
	if demoEnabled == "true" {
		logger.Info("Demo seeding enabled for table service")
		seedingFunc = tables.DemoSeedingFunc(seedCtx, tableRepo, seedFS, logger)
	} else {
		seedingFunc = tables.SeedingFunc(seedCtx, tableRepo, seedFS, logger)
	}

	seedHooks := apt.LifecycleHooks{
		OnStart: seedingFunc,
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

	options := []apt.Option{
		apt.WithConfig(config),
		apt.WithLogger(logger),
		apt.WithHTTPMiddleware(stack...),
		apt.WithHTTPServerModules("web.port", handler),
		apt.WithLifecycle(lifecycle...),
		apt.WithHealthChecks(appName),
	}

	ms := apt.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		_ = tableRepo.Stop(context.Background())
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
