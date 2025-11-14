package authn

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aquamarinepk/aqm"
	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/google/uuid"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrUsernameExists     = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveAccount    = errors.New("account is not active")
)

func SignUpUser(ctx context.Context, repo UserRepo, config *aqm.Config, email, password, username, name string) (*User, error) {
	if repo == nil {
		return nil, errors.New("user repository is required")
	}
	if config == nil {
		return nil, errors.New("configuration is required")
	}

	normalizedEmail := authpkg.NormalizeEmail(email)
	encryptionKeyStr, _ := config.GetString("auth.encryption.key")
	signingKeyStr, _ := config.GetString("auth.signing.key")
	encryptionKey := []byte(encryptionKeyStr)
	signingKey := []byte(signingKeyStr)

	encryptedEmail, err := authpkg.EncryptEmail(normalizedEmail, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt email: %w", err)
	}

	emailLookup := authpkg.ComputeLookupHash(normalizedEmail, signingKey)

	existing, err := repo.GetByEmailLookup(ctx, emailLookup)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	normalizedUsername, err := normalizeUsername(username)
	if err != nil {
		return nil, err
	}

	if existingUsername, err := repo.GetByUsername(ctx, normalizedUsername); err != nil {
		return nil, fmt.Errorf("lookup username: %w", err)
	} else if existingUsername != nil {
		return nil, ErrUsernameExists
	}

	displayName, err := normalizeDisplayName(name)
	if err != nil {
		return nil, err
	}

	salt := authpkg.GeneratePasswordSalt()
	passwordHash := authpkg.HashPassword([]byte(password), salt)

	user := NewUser()
	user.Username = normalizedUsername
	user.Name = displayName
	user.EmailCT = encryptedEmail.Ciphertext
	user.EmailIV = encryptedEmail.IV
	user.EmailTag = encryptedEmail.Tag
	user.EmailLookup = emailLookup
	user.PasswordHash = passwordHash
	user.PasswordSalt = salt
	user.BeforeCreate()

	if err := repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func SignInUser(ctx context.Context, repo UserRepo, config *aqm.Config, email, password string) (*User, string, error) {
	if repo == nil {
		return nil, "", errors.New("user repository is required")
	}
	if config == nil {
		return nil, "", errors.New("configuration is required")
	}

	normalizedEmail := authpkg.NormalizeEmail(email)
	signingKeyStr, _ := config.GetString("auth.signing.key")
	signingKey := []byte(signingKeyStr)
	emailLookup := authpkg.ComputeLookupHash(normalizedEmail, signingKey)

	user, err := repo.GetByEmailLookup(ctx, emailLookup)
	if err != nil {
		return nil, "", fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	if !authpkg.VerifyPasswordHash([]byte(password), user.PasswordHash, user.PasswordSalt) {
		return nil, "", ErrInvalidCredentials
	}

	if user.Status != authpkg.UserStatusActive {
		return nil, "", ErrInactiveAccount
	}

	token, err := generateSessionToken(config, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("generate session token: %w", err)
	}

	return user, token, nil
}

// SignInByPIN authenticates a user using their PIN and returns the user.
// This is designed for lightweight authentication in the conversational interface.
func SignInByPIN(ctx context.Context, repo UserRepo, config *aqm.Config, pin string) (*User, error) {
	if repo == nil {
		return nil, errors.New("user repository is required")
	}
	if config == nil {
		return nil, errors.New("configuration is required")
	}

	normalizedPIN := authpkg.NormalizePIN(pin)
	if normalizedPIN == "" {
		return nil, ErrInvalidCredentials
	}

	signingKeyStr, _ := config.GetString("auth.signing.key")
	signingKey := []byte(signingKeyStr)
	pinLookup := authpkg.ComputePINLookupHash(normalizedPIN, signingKey)

	user, err := repo.GetByPINLookup(ctx, pinLookup)
	if err != nil {
		return nil, fmt.Errorf("lookup user by PIN: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if user.Status != authpkg.UserStatusActive {
		return nil, ErrInactiveAccount
	}

	return user, nil
}

func GenerateBootstrapStatus(ctx context.Context, repo UserRepo, config *aqm.Config) (*User, error) {
	if repo == nil {
		return nil, errors.New("user repository is required")
	}
	if config == nil {
		return nil, errors.New("configuration is required")
	}

	signingKeyStr, _ := config.GetString("auth.signing.key")
	signingKey := []byte(signingKeyStr)
	normalizedEmail := authpkg.NormalizeEmail(SuperadminEmail)
	lookupHash := authpkg.ComputeLookupHash(normalizedEmail, signingKey)

	return repo.GetByEmailLookup(ctx, lookupHash)
}

func BootstrapSuperadmin(ctx context.Context, repo UserRepo, config *aqm.Config) (*User, string, error) {
	if repo == nil {
		return nil, "", errors.New("user repository is required")
	}
	if config == nil {
		return nil, "", errors.New("configuration is required")
	}

	signingKeyStr, _ := config.GetString("auth.signing.key")
	encryptionKeyStr, _ := config.GetString("auth.encryption.key")
	signingKey := []byte(signingKeyStr)
	encryptionKey := []byte(encryptionKeyStr)

	normalizedEmail := authpkg.NormalizeEmail(SuperadminEmail)
	lookupHash := authpkg.ComputeLookupHash(normalizedEmail, signingKey)

	existing, err := repo.GetByEmailLookup(ctx, lookupHash)
	if err != nil {
		return nil, "", fmt.Errorf("lookup superadmin: %w", err)
	}
	if existing != nil {
		return existing, "", nil
	}

	encryptedEmail, err := authpkg.EncryptEmail(normalizedEmail, encryptionKey)
	if err != nil {
		return nil, "", fmt.Errorf("encrypt email: %w", err)
	}

	password := authpkg.GenerateSecurePassword(32)
	passwordSalt := authpkg.GeneratePasswordSalt()
	passwordHash := authpkg.HashPassword([]byte(password), passwordSalt)

	user := &User{
		ID:           uuid.New(),
		Username:     "superadmin",
		Name:         "Super Administrator",
		EmailCT:      encryptedEmail.Ciphertext,
		EmailIV:      encryptedEmail.IV,
		EmailTag:     encryptedEmail.Tag,
		EmailLookup:  lookupHash,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		Status:       authpkg.UserStatusActive,
		CreatedAt:    time.Now(),
		CreatedBy:    "system",
		UpdatedAt:    time.Now(),
		UpdatedBy:    "system",
	}

	if err := repo.Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("create superadmin: %w", err)
	}

	return user, password, nil
}

// GeneratePINForUser generates a unique PIN for a user and stores it encrypted.
// Returns the plain PIN (which should be shown only once) and updates the user.
func GeneratePINForUser(ctx context.Context, repo UserRepo, config *aqm.Config, user *User) (string, error) {
	if repo == nil {
		return "", errors.New("user repository is required")
	}
	if config == nil {
		return "", errors.New("configuration is required")
	}
	if user == nil {
		return "", errors.New("user is required")
	}

	encryptionKeyStr, _ := config.GetString("auth.encryption.key")
	signingKeyStr, _ := config.GetString("auth.signing.key")
	encryptionKey := []byte(encryptionKeyStr)
	signingKey := []byte(signingKeyStr)

	// Try up to 10 times to generate a unique PIN
	const maxAttempts = 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		pin := authpkg.GeneratePIN()
		pinLookup := authpkg.ComputePINLookupHash(pin, signingKey)

		// Check if PIN already exists
		existing, err := repo.GetByPINLookup(ctx, pinLookup)
		if err != nil {
			return "", fmt.Errorf("check PIN collision: %w", err)
		}
		if existing != nil {
			continue
		}

		// Encrypt the PIN
		encryptedPIN, err := authpkg.EncryptEmail(pin, encryptionKey) // Reusing email encryption
		if err != nil {
			return "", fmt.Errorf("encrypt PIN: %w", err)
		}

		// Store encrypted PIN in user
		user.PINCT = encryptedPIN.Ciphertext
		user.PINIV = encryptedPIN.IV
		user.PINTag = encryptedPIN.Tag
		user.PINLookup = pinLookup

		return pin, nil
	}

	return "", errors.New("failed to generate unique PIN after multiple attempts")
}

func generateSessionToken(config *aqm.Config, userID uuid.UUID) (string, error) {
	sessionTTLStr, _ := config.GetString("auth.session.ttl")
	ttl, err := time.ParseDuration(sessionTTLStr)
	if err != nil {
		return "", fmt.Errorf("invalid session TTL: %w", err)
	}

	tokenKeyStr, _ := config.GetString("auth.token.key.private")
	privateKey, err := tokenPrivateKey(tokenKeyStr)
	if err != nil {
		return "", fmt.Errorf("get private key: %w", err)
	}

	sessionID := uuid.New().String()

	return authpkg.GenerateSessionToken(userID.String(), sessionID, privateKey, ttl)
}

func tokenPrivateKey(encoded string) (ed25519.PrivateKey, error) {
	if encoded != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode private key: %w", err)
		}
		return ed25519.PrivateKey(keyBytes), nil
	}

	_, privateKey, err := authpkg.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}
	return privateKey, nil
}

func normalizeUsername(candidate string) (string, error) {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	if candidate == "" {
		return "", errors.New("username is required")
	}
	if len([]rune(candidate)) < 3 {
		return "", errors.New("username must be at least 3 characters")
	}
	if len([]rune(candidate)) > 32 {
		return "", errors.New("username must be 32 characters or less")
	}
	for _, r := range candidate {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
		case r == '-', r == '_', r == '.':
		default:
			return "", errors.New("username can only contain letters, numbers, '.', '-' or '_' characters")
		}
	}
	candidate = strings.Trim(candidate, "._-")
	if candidate == "" {
		return "", errors.New("username cannot consist solely of '.', '-' or '_' characters")
	}
	return candidate, nil
}

func normalizeDisplayName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New("name is required")
	}
	if len([]rune(trimmed)) > 120 {
		return "", errors.New("name must be 120 characters or less")
	}
	return trimmed, nil
}

func deriveUsernameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	base := strings.TrimSpace(parts[0])
	if slug := slugifyUsernameFromName(base); slug != "" {
		return slug
	}
	return fmt.Sprintf("user-%s", uuid.New().String()[:6])
}

func fallbackNameFromEmail(email string) string {
	local := strings.Split(email, "@")
	segment := local[0]
	if segment == "" {
		return "User"
	}
	parts := strings.FieldsFunc(segment, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	for i, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return "User"
	}
	return name
}
