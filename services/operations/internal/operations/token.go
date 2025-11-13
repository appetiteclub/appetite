package operations

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
)

// TransientToken represents a temporary session token for chat authentication.
type TransientToken struct {
	Token        string
	UserID       uuid.UUID
	IssuedAt     time.Time
	ExpiresAt    time.Time
	LastActivity time.Time
}

// TokenStore manages transient authentication tokens in memory.
type TokenStore struct {
	tokens map[string]*TransientToken
	mu     sync.RWMutex
	ttl    time.Duration
}

// NewTokenStore creates a new in-memory token store.
func NewTokenStore(ttl time.Duration) *TokenStore {
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	return &TokenStore{
		tokens: make(map[string]*TransientToken),
		ttl:    ttl,
	}
}

// Create generates a new token for a user.
func (s *TokenStore) Create(userID uuid.UUID) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	tt := &TransientToken{
		Token:        token,
		UserID:       userID,
		IssuedAt:     now,
		ExpiresAt:    now.Add(s.ttl),
		LastActivity: now,
	}

	s.mu.Lock()
	s.tokens[token] = tt
	s.mu.Unlock()

	return token, nil
}

// Validate checks if a token exists and is valid, and updates its last activity.
func (s *TokenStore) Validate(token string) (uuid.UUID, error) {
	s.mu.RLock()
	tt, exists := s.tokens[token]
	s.mu.RUnlock()

	if !exists {
		return uuid.Nil, ErrTokenNotFound
	}

	now := time.Now()
	if now.After(tt.ExpiresAt) {
		s.Invalidate(token)
		return uuid.Nil, ErrTokenExpired
	}

	s.mu.Lock()
	tt.LastActivity = now
	tt.ExpiresAt = now.Add(s.ttl)
	s.mu.Unlock()

	return tt.UserID, nil
}

// Invalidate removes a token from the store.
func (s *TokenStore) Invalidate(token string) {
	s.mu.Lock()
	delete(s.tokens, token)
	s.mu.Unlock()
}

// CleanupExpired removes all expired tokens from the store.
func (s *TokenStore) CleanupExpired() int {
	now := time.Now()
	count := 0

	s.mu.Lock()
	for token, tt := range s.tokens {
		if now.After(tt.ExpiresAt) {
			delete(s.tokens, token)
			count++
		}
	}
	s.mu.Unlock()

	return count
}

// Count returns the number of active tokens.
func (s *TokenStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tokens)
}

// StartCleanup starts a background goroutine that periodically cleans up expired tokens.
// Returns a channel that can be closed to stop the cleanup routine.
func (s *TokenStore) StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count := s.CleanupExpired()
				if count > 0 {
					// Could log here if logger is available
				}
			}
		}
	}()
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
