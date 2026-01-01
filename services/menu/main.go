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
	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/middleware"
)

const (
	appNamespace = "MENU"
	appName      = "menu"
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
	seedHooks := apt.LifecycleHooks{
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
	options := []apt.Option{
		apt.WithConfig(config),
		apt.WithLogger(logger),
		apt.WithHTTPMiddleware(stack...),
		apt.WithHTTPServerModules("web.port", handler),
		apt.WithLifecycle(itemRepo, menuRepo, seedHooks),
		apt.WithHealthChecks(appName),
	}

	ms := apt.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
