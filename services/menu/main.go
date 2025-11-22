package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/appetiteclub/appetite/services/menu/internal/dictionary"
	"github.com/appetiteclub/appetite/services/menu/internal/menu"
	"github.com/appetiteclub/appetite/services/menu/internal/mongo"
	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/middleware"
)

const (
	appNamespace = "MENU"
	appName      = "menu"
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

	// Initialize repositories
	itemRepo := mongo.NewMenuItemRepo(config, logger)
	menuRepo := mongo.NewMenuRepo(itemRepo, logger)

	// Initialize dictionary client
	dictURL := config.GetStringOrDef("services.dictionary.url", "http://localhost:8084")
	dictClient := dictionary.NewHTTPClient(dictURL)

	hd := menu.HandlerDeps{
		ItemRepo:   itemRepo,
		MenuRepo:   menuRepo,
		DictClient: dictClient,
	}

	// Initialize handler
	handler, err := menu.NewHandler(hd, config, logger)
	if err != nil {
		log.Fatalf("Cannot create handler %s(%s): %v", appName, appVersion, err)
	}

	// Setup seeding hooks
	seedHooks := aqm.LifecycleHooks{
		OnStart: menu.SeedingFunc(appName, itemRepo.GetDatabase, logger),
	}

	// Setup middleware
	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      logger,
		DisableCORS: true, // Internal API service
	})
	// Defense-in-depth: restrict to internal networks only.
	// This complements (does not replace) network policies at the infrastructure level.
	stack = append(stack, middleware.InternalOnly())

	// Register with Micro framework
	options := []aqm.Option{
		aqm.WithConfig(config),
		aqm.WithLogger(logger),
		aqm.WithHTTPMiddleware(stack...),
		aqm.WithHTTPServerModules("web.port", handler),
		aqm.WithLifecycle(itemRepo, menuRepo, seedHooks),
		aqm.WithHealthChecks(appName),
	}

	ms := aqm.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
