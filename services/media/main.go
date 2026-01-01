package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/middleware"

	"github.com/appetiteclub/appetite/services/media/internal/dictionary"
	"github.com/appetiteclub/appetite/services/media/internal/media"
	"github.com/appetiteclub/appetite/services/media/internal/storage"
)

const (
	appNamespace = "MEDIA"
	appName      = "media"
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

	storageBackend, err := storage.FromProperties(config)
	if err != nil {
		log.Fatalf("%s(%s) cannot configure storage backend: %v", appName, appVersion, err)
	}

	repo := media.NewInMemoryRepository()
	dictClient := dictionary.NewNoopClient()

	// Load variant definitions from config
	variants := []media.VariantDefinition{}
	// TODO: Parse variants from properties if configured

	enableCropping, _ := config.GetString("processing.cropping")
	enableCompression, _ := config.GetString("processing.compression")

	service := media.NewService(repo, storageBackend, dictClient, media.ServiceOptions{
		EnableCropping:    enableCropping == "true",
		EnableCompression: enableCompression == "true",
		Variants:          variants,
	})

	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      logger,
		DisableCORS: true, // Internal API service
	})
	// Defense-in-depth: restrict to internal networks only.
	// This complements (does not replace) network policies at the infrastructure level.
	stack = append(stack, middleware.InternalOnly())

	ms := apt.NewMicro(
		apt.WithConfig(config),
		apt.WithLogger(logger),
		apt.WithHTTPMiddleware(stack...),
		apt.WithHealthChecks(appName),
	)

	// Create handler with dependencies from Micro
	handler := media.NewHandler(service, ms.Deps())

	// Register HTTP server with handler
	if err := apt.WithHTTPServerModules("web.port", handler)(ms); err != nil {
		log.Fatalf("%s(%s) cannot register HTTP modules: %v", appName, appVersion, err)
	}

	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		log.Fatalf("%s(%s) stopped: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
