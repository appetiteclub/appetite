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

	seedCtx, cancelSeeds := context.WithCancel(ctx)
	defer cancelSeeds()

	tableRepo := mongo.NewTableRepo(config, logger)
	if err := tableRepo.Start(ctx); err != nil {
		log.Fatalf("Cannot start table repository: %v", err)
	}

	db := tableRepo.GetDatabase()
	if db == nil {
		log.Fatalf("cannot initialize table repo database")
	}

	groupRepo := mongo.NewGroupRepo(db)
	orderRepo := mongo.NewOrderRepo(db)
	orderItemRepo := mongo.NewOrderItemRepo(db)
	reservationRepo := mongo.NewReservationRepo(db)

	handler := tables.NewHandler(
		tableRepo,
		groupRepo,
		orderRepo,
		orderItemRepo,
		reservationRepo,
		logger,
		config,
	)

	seedHooks := aqm.LifecycleHooks{
		OnStart: tables.SeedingFunc(seedCtx, tableRepo, seedFS, logger),
		OnStop:  tables.StopFunc(cancelSeeds),
	}

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
		aqm.WithLifecycle(tableRepo, seedHooks),
		aqm.WithHealthChecks(appName),
	}

	ms := aqm.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		tableRepo.Stop(context.Background())
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
