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

	"github.com/appetiteclub/appetite/services/operations/internal/operations"
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

	// Initialize handler
	handler := operations.NewHandler(tmplMgr, roleRepo, grantRepo, config, logger)

	// Configure middleware stack (no InternalOnly - this is public-facing)
	stack := middleware.DefaultStack(middleware.StackOptions{
		Logger:      logger,
		DisableCORS: false, // Enable CORS for operations service
	})

	options := []aqm.Option{
		aqm.WithConfig(config),
		aqm.WithLogger(logger),
		aqm.WithHTTPMiddleware(stack...),
		aqm.WithHTTPServerModules("web.port", handler),
		aqm.WithLifecycle(tmplMgr),
		aqm.WithHealthChecks(appName),
	}

	ms := aqm.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		log.Fatalf("%s(%s) stopped with error: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
