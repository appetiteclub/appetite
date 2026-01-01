package tables

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/appetiteclub/apt"
	"github.com/appetiteclub/apt/seed"
	"go.mongodb.org/mongo-driver/mongo"
)

const tableSeedApplication = "table"

type bootstrapSeedDocument struct {
	Tables []tableSeed `json:"tables"`
}

type tableSeed struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	GuestCount int    `json:"guest_count"`
}

func loadTableSeeds(seedFS embed.FS) ([]tableSeed, error) {
	seedBytes, err := seedFS.ReadFile("seed.json")
	if err != nil {
		return nil, fmt.Errorf("read seed.json: %w", err)
	}

	if len(seedBytes) == 0 {
		return nil, errors.New("table seed file is empty")
	}

	var doc bootstrapSeedDocument
	if err := json.Unmarshal(seedBytes, &doc); err != nil {
		return nil, fmt.Errorf("decode table seed file: %w", err)
	}

	if len(doc.Tables) == 0 {
		return nil, errors.New("table seed file does not contain tables")
	}

	return doc.Tables, nil
}

// ApplyTableSeeds ensures all predefined tables exist.
func ApplyTableSeeds(ctx context.Context, repo TableRepo, seedFS embed.FS, logger apt.Logger) error {
	if repo == nil {
		return errors.New("table repository is required")
	}

	seedDocs, err := loadTableSeeds(seedFS)
	if err != nil {
		return err
	}

	seedDefs, err := buildTableSeedDefinitions(seedDocs, repo, logger)
	if err != nil {
		return err
	}
	if len(seedDefs) == 0 {
		logger.Info("No table seeds to apply")
		return nil
	}

	tracker, err := trackerFromRepo(repo)
	if err != nil {
		return err
	}

	logger.Info("Applying table seeds")
	if err := seed.Apply(ctx, tracker, seedDefs, tableSeedApplication); err != nil {
		return err
	}
	logger.Info("Table seeds applied successfully")
	return nil
}

func trackerFromRepo(repo TableRepo) (seed.Tracker, error) {
	provider, ok := repo.(mongoDatabaseProvider)
	if !ok {
		return nil, errors.New("table repository does not expose MongoDB access for seeding")
	}
	db := provider.GetDatabase()
	if db == nil {
		return nil, errors.New("table repository database is not initialized")
	}
	return seed.NewMongoTracker(db), nil
}

type mongoDatabaseProvider interface {
	GetDatabase() *mongo.Database
}

func buildTableSeedDefinitions(raw []tableSeed, repo TableRepo, logger apt.Logger) ([]seed.Seed, error) {
	var defs []seed.Seed

	for _, s := range raw {
		seedData := s
		if strings.TrimSpace(seedData.Number) == "" {
			logger.Info("Skipping seed table with empty number")
			continue
		}

		logger.Info("Including seed table", "number", seedData.Number, "status", seedData.Status, "guest_count", seedData.GuestCount)

		seedID := fmt.Sprintf("2024-11-17_table_%s", seedIdentifier(seedData.Number))
		description := fmt.Sprintf("Ensure table %s exists", seedData.Number)

		defs = append(defs, seed.Seed{
			ID:          seedID,
			Description: description,
			Run: func(ctx context.Context) error {
				return seedData.ensureTable(ctx, repo, logger)
			},
		})
	}

	return defs, nil
}

func seedIdentifier(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}

	replacer := strings.NewReplacer("-", "_", " ", "_", "/", "_", "\\", "_")
	value = replacer.Replace(value)

	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		}
	}

	result := builder.String()
	if result == "" {
		return "seed"
	}
	return result
}

func (s tableSeed) ensureTable(ctx context.Context, repo TableRepo, logger apt.Logger) error {
	number := strings.TrimSpace(s.Number)
	if number == "" {
		return errors.New("table number is required")
	}

	status := s.Status
	if status == "" {
		status = "available"
	}

	// Check if table with this number already exists
	tables, err := repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list existing tables: %w", err)
	}

	for _, existing := range tables {
		if existing.Number == number {
			logger.Info("Seed table already exists", "number", number)
			return nil
		}
	}

	// Create new table
	table := NewTable()
	table.Number = number
	table.Status = status
	table.GuestCount = s.GuestCount
	table.CreatedBy = "seed:bootstrap"
	table.UpdatedBy = "seed:bootstrap"
	table.BeforeCreate()

	if err := repo.Create(ctx, table); err != nil {
		return fmt.Errorf("create seed table %s: %w", number, err)
	}

	logger.Info("Seed table created", "number", number, "id", table.ID.String())
	return nil
}

// SeedingFunc returns an aqm lifecycle OnStart-compatible function which
// starts applying table seeds in the background.
func SeedingFunc(seedCtx context.Context, repo TableRepo, seedFS embed.FS, logger apt.Logger) func(ctx context.Context) error {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}

	return func(ctx context.Context) error {
		logger.Info("Starting table seeding in background")
		go func() {
			if err := ApplyTableSeeds(seedCtx, repo, seedFS, logger); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("❌ Table seeds failed: %v", err)
			} else if err == nil {
				logger.Info("✓ Table seeding completed successfully")
			}
		}()
		return nil
	}
}

// StopFunc returns an aqm lifecycle OnStop-compatible function which calls
// the provided cancel function to stop any background seeding goroutine.
func StopFunc(cancelFunc context.CancelFunc) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if cancelFunc != nil {
			cancelFunc()
		}
		return nil
	}
}
