package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/middleware"

	"github.com/appetiteclub/appetite/services/authz/internal/authz"
	"github.com/appetiteclub/appetite/services/authz/internal/mongo"
)

//go:embed seed.json
var seedFS embed.FS

const (
	appName    = "authz"
	appVersion = "0.1.0"
)

func main() {
	config, err := apt.LoadConfig("AUTHZ", os.Args[1:])
	if err != nil {
		log.Fatalf("Cannot setup %s(%s): %v", appName, appVersion, err)
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

	roleRepo := mongo.NewRoleMongoRepo(config, logger)
	grantRepo := mongo.NewGrantMongoRepo(logger, config)

	policyEngine := authz.NewPolicyEngine(roleRepo, grantRepo)
	roleHandler := authz.NewRoleHandler(roleRepo, config, logger)
	grantHandler := authz.NewGrantHandler(grantRepo, roleRepo, config, logger)
	policyHandler := authz.NewPolicyHandler(policyEngine, config, logger)

	bootstrapService := authz.NewBootstrapService(roleRepo, grantRepo, seedFS, config, logger)
	bootstrapHooks := apt.LifecycleHooks{
		OnStart: authz.BootstrapFunc(bootstrapService, logger),
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
		apt.WithHTTPServerModules("web.port", roleHandler, grantHandler, policyHandler),
		apt.WithLifecycle(roleRepo, grantRepo, bootstrapHooks),
		apt.WithHealthChecks(appName),
	}

	ms := apt.NewMicro(options...)
	logger.Infof("Starting %s(%s)", appName, appVersion)

	if err := ms.Run(ctx); err != nil {
		log.Fatalf("%s(%s) stopped: %v", appName, appVersion, err)
	}

	logger.Infof("%s(%s) stopped", appName, appVersion)
}
