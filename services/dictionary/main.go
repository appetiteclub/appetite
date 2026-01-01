package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/appetiteclub/appetite/services/dictionary/internal/dictionary"
	"github.com/appetiteclub/appetite/services/dictionary/internal/mongo"
	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/middleware"
)

const (
	appNamespace = "DICTIONARY"
	appName      = "dictionary"
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

	setRepo := mongo.NewSetRepo(config, logger)
	optionRepo := mongo.NewOptionRepo(setRepo, config, logger)
	handler := dictionary.NewHandler(setRepo, optionRepo, config, logger)

	seedHooks := apt.LifecycleHooks{
		OnStart: dictionary.SeedingFunc(appName, setRepo.GetDatabase, logger),
	}

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
		apt.WithLifecycle(setRepo, optionRepo, seedHooks),
		apt.WithHealthChecks(appName),
	}

	ms := apt.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		log.Fatalf("%s(%s) stopped: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
