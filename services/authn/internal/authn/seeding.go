package authn

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aquamarinepk/aqm/seed"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/aquamarinepk/aqm"
	authpkg "github.com/aquamarinepk/aqm/auth"
)

const authnSeedApplication = "authn"

type bootstrapSeedDocument struct {
	Users []userSeed `json:"users"`
}

type userSeed struct {
	Name      string `json:"name"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	PIN       string `json:"pin,omitempty"`
	Status    string `json:"status"`
	Reference bool   `json:"reference"`
}

func loadUserSeeds(seedFS embed.FS) ([]userSeed, error) {
	seedBytes, err := seedFS.ReadFile("seed.json")
	if err != nil {
		return nil, fmt.Errorf("read seed.json: %w", err)
	}

	if len(seedBytes) == 0 {
		return nil, errors.New("bootstrap seed file is empty")
	}

	var doc bootstrapSeedDocument
	if err := json.Unmarshal(seedBytes, &doc); err != nil {
		return nil, fmt.Errorf("decode bootstrap seed file: %w", err)
	}

	if len(doc.Users) == 0 {
		return nil, errors.New("bootstrap seed file does not contain users")
	}

	return doc.Users, nil
}

// ApplyUserSeeds ensures all predefined users exist (except the superadmin).
func ApplyUserSeeds(ctx context.Context, repo UserRepo, seedFS embed.FS, logger aqm.Logger, config *aqm.Config) error {
	if repo == nil {
		return errors.New("user repository is required")
	}

	if config == nil {
		return errors.New("configuration is required")
	}

	if err := waitForSuperadmin(ctx, repo, config, logger); err != nil {
		return err
	}

	seedDocs, err := loadUserSeeds(seedFS)
	if err != nil {
		return err
	}

	seedDefs, err := buildUserSeedDefinitions(seedDocs, repo, config, logger)
	if err != nil {
		return err
	}
	if len(seedDefs) == 0 {
		logger.Info("No AuthN user seeds to apply")
		return nil
	}

	tracker, err := trackerFromRepo(repo)
	if err != nil {
		return err
	}

	logger.Info("Applying AuthN user seeds")
	if err := seed.Apply(ctx, tracker, seedDefs, authnSeedApplication); err != nil {
		return err
	}
	logger.Info("AuthN user seeds applied successfully")
	return nil
}

func trackerFromRepo(repo UserRepo) (seed.Tracker, error) {
	provider, ok := repo.(mongoDatabaseProvider)
	if !ok {
		return nil, errors.New("user repository does not expose MongoDB access for seeding")
	}
	db := provider.Database()
	if db == nil {
		return nil, errors.New("user repository database is not initialized")
	}
	return seed.NewMongoTracker(db), nil
}

func buildUserSeedDefinitions(raw []userSeed, repo UserRepo, config *aqm.Config, logger aqm.Logger) ([]seed.Seed, error) {
	var defs []seed.Seed

	for _, s := range raw {
		seedData := s
		if seedData.shouldSkip() {
			logger.Info("Skipping seed user", "email", seedData.Email, "reference", seedData.Reference, "password", seedData.Password)
			continue
		}

		logger.Info("Including seed user", "email", seedData.Email, "reference", seedData.Reference, "has_pin", seedData.PIN != "")

		seedID := fmt.Sprintf("2024-11-15_authn_user_%s", seedIdentifier(seedData.Email))
		description := fmt.Sprintf("Ensure AuthN bootstrap user %s", seedData.Email)

		defs = append(defs, seed.Seed{
			ID:          seedID,
			Description: description,
			Run: func(ctx context.Context) error {
				return seedData.ensureUser(ctx, repo, config, logger)
			},
		})
	}

	return defs, nil
}

type mongoDatabaseProvider interface {
	Database() *mongo.Database
}

func seedIdentifier(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}

	replacer := strings.NewReplacer("@", "_", ".", "_", "-", "_", "+", "_", " ", "_")
	value = replacer.Replace(value)
	var builder strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '_':
			builder.WriteRune(r)
		}
	}
	result := builder.String()
	if result == "" {
		return "seed"
	}
	return result
}

func waitForSuperadmin(ctx context.Context, repo UserRepo, config *aqm.Config, logger aqm.Logger) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastLog := time.Time{}

	for {
		user, err := GenerateBootstrapStatus(ctx, repo, config)
		if err != nil {
			return fmt.Errorf("check superadmin status: %w", err)
		}

		if user != nil {
			logger.Info("Superadmin detected, continuing with seed users")
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if lastLog.IsZero() || time.Since(lastLog) >= 30*time.Second {
				logger.Info("Waiting for superadmin bootstrap before applying seed users")
				lastLog = time.Now()
			}
		}
	}
}

func (s userSeed) shouldSkip() bool {
	if strings.TrimSpace(s.Email) == "" {
		return true
	}

	// Skip superadmin (handled by bootstrap)
	normalized := authpkg.NormalizeEmail(s.Email)
	if normalized == authpkg.NormalizeEmail(SuperadminEmail) {
		return true
	}

	// Skip only if password is <auto> AND it's NOT a reference user
	// Reference users (like Agent) can have auto-generated passwords
	if strings.Contains(s.Password, "<auto") && !s.Reference {
		return true
	}

	return false
}

func (s userSeed) ensureUser(ctx context.Context, repo UserRepo, config *aqm.Config, logger aqm.Logger) error {
	desiredStatus := s.status()
	username, err := s.username()
	if err != nil {
		return fmt.Errorf("seed user %s username invalid: %w", s.Email, err)
	}

	name := strings.TrimSpace(s.Name)
	if name == "" {
		return fmt.Errorf("seed user %s missing name", s.Email)
	}

	// Generate password if <auto> for reference users
	password := s.Password
	if strings.Contains(password, "<auto>") {
		password = authpkg.GenerateSecurePassword(32)
		// TODO: SECURITY - Remove password logging in production! This is only for development.
		logger.Info("⚠️  DEVELOPMENT ONLY - Generated password for seed user (REMOVE THIS LOG IN PRODUCTION!)", "email", s.Email, "password", password)
	}

	user, err := SignUpUser(ctx, repo, config, s.Email, password, username, name)
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			logger.Info("Seed user already exists", "email", s.Email)

			// If user exists but needs PIN generation, handle it
			if strings.Contains(s.PIN, "<auto>") {
				// Get existing user to check if PIN needs generation
				signingKeyStr, _ := config.GetString("auth.signing.key")
				signingKey := []byte(signingKeyStr)
				normalizedEmail := authpkg.NormalizeEmail(s.Email)
				emailLookup := authpkg.ComputeLookupHash(normalizedEmail, signingKey)

				existingUser, err := repo.GetByEmailLookup(ctx, emailLookup)
				if err != nil {
					return fmt.Errorf("lookup existing user %s: %w", s.Email, err)
				}

				if existingUser != nil && len(existingUser.PINLookup) == 0 {
					// User exists but has no PIN, generate one
					pin, err := GeneratePINForUser(ctx, repo, config, existingUser)
					if err != nil {
						return fmt.Errorf("generate PIN for existing seed user %s: %w", s.Email, err)
					}
					existingUser.UpdatedBy = "seed:bootstrap"
					if err := repo.Save(ctx, existingUser); err != nil {
						return fmt.Errorf("save PIN for existing seed user %s: %w", s.Email, err)
					}
					// TODO: SECURITY - Remove PIN logging in production! This is only for development.
					logger.Info("⚠️  DEVELOPMENT ONLY - Generated PIN for existing seed user (REMOVE THIS LOG IN PRODUCTION!)", "email", s.Email, "pin", pin)
				}
			}

			return nil
		}
		return fmt.Errorf("create seed user %s: %w", s.Email, err)
	}

	needsSave := false

	// Generate PIN if specified
	if strings.Contains(s.PIN, "<auto>") {
		pin, err := GeneratePINForUser(ctx, repo, config, user)
		if err != nil {
			return fmt.Errorf("generate PIN for seed user %s: %w", s.Email, err)
		}
		needsSave = true
		// TODO: SECURITY - Remove PIN logging in production! This is only for development.
		logger.Info("⚠️  DEVELOPMENT ONLY - Generated PIN for seed user (REMOVE THIS LOG IN PRODUCTION!)", "email", s.Email, "pin", pin)
	}

	if desiredStatus != "" && desiredStatus != authpkg.UserStatusActive {
		user.Status = desiredStatus
		needsSave = true
	}

	if needsSave {
		user.UpdatedBy = "seed:bootstrap"
		if err := repo.Save(ctx, user); err != nil {
			return fmt.Errorf("update seed user %s: %w", s.Email, err)
		}
	}

	logger.Info("Seed user created", "email", s.Email)
	return nil
}

func (s userSeed) status() authpkg.UserStatus {
	status := strings.TrimSpace(strings.ToLower(s.Status))
	if status == "" {
		return authpkg.UserStatusActive
	}
	return authpkg.UserStatus(status)
}

func (s userSeed) username() (string, error) {
	if strings.TrimSpace(s.Username) != "" {
		return normalizeUsername(s.Username)
	}
	slug := slugifyUsernameFromName(s.Name)
	if slug == "" {
		return "", errors.New("username is required")
	}
	return normalizeUsername(slug)
}

func slugifyUsernameFromName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		default:
			if !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

// SeedingFunc returns an aqm lifecycle OnStart-compatible function which
// starts applying AuthN user seeds in the background. It accepts the
// seed context (usually created with context.WithCancel), the user repo,
// the embedded seed FS, a logger and config. It mirrors the behaviour of
// the previous inline anonymous function in main.
func SeedingFunc(seedCtx context.Context, repo UserRepo, seedFS embed.FS, config *aqm.Config, logger aqm.Logger) func(ctx context.Context) error {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}

	return func(ctx context.Context) error {
		go func() {
			if err := ApplyUserSeeds(seedCtx, repo, seedFS, logger, config); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("AuthN user seeds failed: %v", err)
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
