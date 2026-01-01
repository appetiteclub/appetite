package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/appetiteclub/appetite/cmd/utils/internal/commands"
	"github.com/appetiteclub/apt"
)

const (
	appName    = "appetite-utils"
	appVersion = "0.1.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load config from UTILS namespace (or use default mongo connection)
	config, err := apt.LoadConfig("UTILS", os.Args[2:])
	if err != nil {
		log.Fatalf("Cannot load config: %v", err)
	}

	logLevel, _ := config.GetString("log.level")
	if logLevel == "" {
		logLevel = "info"
	}
	logger := apt.NewLogger(logLevel)

	ctx := context.Background()
	command := os.Args[1]

	switch command {
	case "seed-demo":
		if err := commands.SeedDemo(ctx, config, logger); err != nil {
			log.Fatalf("❌ Demo seeding failed: %v", err)
		}
		logger.Info("✅ Demo seeding completed successfully")

	case "clear-demo":
		if err := commands.ClearDemo(ctx, config, logger); err != nil {
			log.Fatalf("❌ Clear demo data failed: %v", err)
		}
		logger.Info("✅ Demo data cleared successfully")

	case "reset-db":
		if err := commands.ResetDB(ctx, config, logger); err != nil {
			log.Fatalf("❌ Database reset failed: %v", err)
		}
		logger.Info("✅ Database reset completed successfully")

	case "version":
		fmt.Printf("%s version %s\n", appName, appVersion)

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`%s - Appetite utility commands

Usage:
  %s <command> [options]

Commands:
  seed-demo    Apply demo seeding (creates sample orders and kitchen tickets)
  clear-demo   Clear demo data (removes demo orders and kitchen tickets)
  reset-db     Full database reset (drops all databases - USE WITH CAUTION)
  version      Print version information
  help         Show this help message

Environment Variables:
  UTILS_MONGO_URL   MongoDB connection URL (default: mongodb://admin:password@localhost:27017/admin?authSource=admin)
  UTILS_LOG_LEVEL   Log level: debug, info, warn, error (default: info)

Examples:
  %s seed-demo
  %s clear-demo
  UTILS_MONGO_URL=mongodb://localhost:27017 %s reset-db

`, appName, appName, appName, appName, appName)
}
