package main

import (
	"context"
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aquamarinepk/aqm"
	"github.com/aquamarinepk/aqm/fileserver"
	aqmmw "github.com/aquamarinepk/aqm/middleware"
	"github.com/aquamarinepk/aqm/template"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/appetiteclub/appetite/services/admin/internal/admin"
)

const (
	appNamespace = "ADMIN"
	appName      = "admin"
	appVersion   = "0.1.0"
)

//go:embed assets
var assetsFS embed.FS

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

	fileServer := fileserver.New(assetsFS, fileserver.WithLogger(logger))
	tmplMgr := template.NewManager(assetsFS, template.WithLogger(logger))

	userRepo, err := admin.NewAPIUserRepo(config, logger)
	if err != nil {
		log.Fatalf("cannot initialize user repo: %v", err)
	}

	propertyRepo, err := admin.NewAPIPropertyRepo(config, logger)
	if err != nil {
		log.Fatalf("cannot initialize property repo: %v", err)
	}

	dictRepo, err := admin.NewAPIDictionaryRepo(config, logger)
	if err != nil {
		log.Fatalf("cannot initialize dictionary repository: %v", err)
	}

	roleRepo := admin.NewFakeRoleRepo()
	grantRepo := admin.NewFakeGrantRepo(userRepo, roleRepo)
	mediaRepo := admin.NewAPIMediaRepo(config, logger)
	locationProvider := admin.NewLocationProvider(config)

	repos := admin.Repos{
		UserRepo:     userRepo,
		RoleRepo:     roleRepo,
		GrantRepo:    grantRepo,
		PropertyRepo: propertyRepo,
		MediaRepo:    mediaRepo,
	}

	adminService := admin.NewDefaultService(repos, locationProvider, config, logger)

	adminHandler, err := admin.NewHandler(tmplMgr, adminService, dictRepo, config, logger)
	if err != nil {
		log.Fatalf("cannot initialize admin handler: %v", err)
	}

	stack := aqmmw.DefaultStack(aqmmw.StackOptions{
		Logger: logger,
	})
	stack = append(stack, chimw.NoCache)

	options := []aqm.Option{
		aqm.WithConfig(config),
		aqm.WithLogger(logger),
		aqm.WithHTTPMiddleware(stack...),
		aqm.WithRouterConfigurator(func(mux *chi.Mux) {
			aqm.RedirectNotFound(mux, "/")
		}),
		aqm.WithHTTPServerModules("web.port", fileServer, adminHandler),
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
