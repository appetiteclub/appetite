package tables

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/seed"
)

// ApplyDemoSeeds ensures demo tables exist with realistic states for demonstration.
// It applies standard seeding first, then modifies some tables to "open" status
// so they can have orders assigned to them.
func ApplyDemoSeeds(ctx context.Context, repo TableRepo, seedFS embed.FS, logger apt.Logger) error {
	if repo == nil {
		return errors.New("table repository is required")
	}

	// First, apply standard seeds to ensure base tables exist
	if err := ApplyTableSeeds(ctx, repo, seedFS, logger); err != nil {
		return fmt.Errorf("apply standard table seeds: %w", err)
	}

	// Build demo-specific seeds to modify tables for demo scenario
	demoSeeds := buildDemoTableSeeds(repo, logger)
	if len(demoSeeds) == 0 {
		logger.Info("No demo table modifications to apply")
		return nil
	}

	tracker, err := trackerFromRepo(repo)
	if err != nil {
		return err
	}

	logger.Info("Applying demo table modifications")
	if err := seed.Apply(ctx, tracker, demoSeeds, tableSeedApplication); err != nil {
		return err
	}
	logger.Info("Demo table modifications applied successfully")
	return nil
}

func buildDemoTableSeeds(repo TableRepo, logger apt.Logger) []seed.Seed {
	// Tables to modify for demo: set to "open" status with guest counts
	// This allows order seeding to assign orders to these tables
	demoModifications := []struct {
		number     string
		guestCount int
	}{
		{"Window-1", 2},   // Couple having drinks and desserts
		{"Center-2", 4},   // Group of 4 having dinner
		{"Patio-3", 1},    // Solo diner
		{"Booth-7", 3},    // Small group with cocktails
		{"Terrace-8", 6},  // Large group
	}

	var defs []seed.Seed
	for _, mod := range demoModifications {
		tableNumber := mod.number
		guestCount := mod.guestCount

		logger.Info("Including demo table modification", "number", tableNumber, "status", "open", "guest_count", guestCount)

		seedID := fmt.Sprintf("2024-11-23_demo_table_open_%s", seedIdentifier(tableNumber))
		description := fmt.Sprintf("Set table %s to open status for demo", tableNumber)

		defs = append(defs, seed.Seed{
			ID:          seedID,
			Description: description,
			Run: func(ctx context.Context) error {
				return setTableToOpen(ctx, repo, tableNumber, guestCount, logger)
			},
		})
	}

	return defs
}

func setTableToOpen(ctx context.Context, repo TableRepo, number string, guestCount int, logger apt.Logger) error {
	// Find the table by number
	tables, err := repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	var targetTable *Table
	for _, t := range tables {
		if t.Number == number {
			targetTable = t
			break
		}
	}

	if targetTable == nil {
		logger.Info("Demo table not found, skipping", "number", number)
		return nil
	}

	// Update to "open" status with guest count
	targetTable.Status = "open"
	targetTable.GuestCount = guestCount
	targetTable.UpdatedBy = "seed:demo"
	targetTable.BeforeUpdate()

	if err := repo.Save(ctx, targetTable); err != nil {
		return fmt.Errorf("save table %s to open: %w", number, err)
	}

	logger.Info("Demo table set to open", "number", number, "guest_count", guestCount)
	return nil
}

// DemoSeedingFunc returns an aqm lifecycle OnStart-compatible function which
// starts applying demo table seeds in the background.
func DemoSeedingFunc(seedCtx context.Context, repo TableRepo, seedFS embed.FS, logger apt.Logger) func(ctx context.Context) error {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}

	return func(ctx context.Context) error {
		logger.Info("Starting demo table seeding in background")
		go func() {
			if err := ApplyDemoSeeds(seedCtx, repo, seedFS, logger); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("❌ Demo table seeds failed: %v", err)
			} else if err == nil {
				logger.Info("✓ Demo table seeding completed successfully")
			}
		}()
		return nil
	}
}
