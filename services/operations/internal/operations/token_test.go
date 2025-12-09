package operations

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewTokenStore(t *testing.T) {
	tests := []struct {
		name        string
		ttl         time.Duration
		wantDefault bool
	}{
		{
			name:        "withCustomTTL",
			ttl:         10 * time.Minute,
			wantDefault: false,
		},
		{
			name:        "withZeroTTL",
			ttl:         0,
			wantDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewTokenStore(tt.ttl)
			if store == nil {
				t.Fatal("NewTokenStore() returned nil")
			}
			if store.tokens == nil {
				t.Error("tokens map is nil")
			}
			if tt.wantDefault && store.ttl != 30*time.Minute {
				t.Errorf("ttl = %v, want %v", store.ttl, 30*time.Minute)
			}
			if !tt.wantDefault && store.ttl != tt.ttl {
				t.Errorf("ttl = %v, want %v", store.ttl, tt.ttl)
			}
		})
	}
}

func TestTokenStoreCreate(t *testing.T) {
	store := NewTokenStore(5 * time.Minute)
	userID := uuid.New()

	token, err := store.Create(userID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if token == "" {
		t.Error("Create() returned empty token")
	}

	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1", store.Count())
	}

	// Verify token is stored correctly
	store.mu.RLock()
	tt := store.tokens[token]
	store.mu.RUnlock()

	if tt == nil {
		t.Fatal("token not found in store")
	}
	if tt.UserID != userID {
		t.Errorf("UserID = %v, want %v", tt.UserID, userID)
	}
	if tt.Token != token {
		t.Errorf("Token = %v, want %v", tt.Token, token)
	}
}

func TestTokenStoreCreateMultiple(t *testing.T) {
	store := NewTokenStore(5 * time.Minute)

	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		token, err := store.Create(uuid.New())
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if tokens[token] {
			t.Errorf("duplicate token generated: %s", token)
		}
		tokens[token] = true
	}

	if store.Count() != 10 {
		t.Errorf("Count() = %d, want 10", store.Count())
	}
}

func TestTokenStoreValidate(t *testing.T) {
	store := NewTokenStore(1 * time.Hour)
	userID := uuid.New()

	token, err := store.Create(userID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantErr   error
		wantValid bool
	}{
		{
			name:      "validToken",
			token:     token,
			wantErr:   nil,
			wantValid: true,
		},
		{
			name:      "nonExistentToken",
			token:     "invalid-token",
			wantErr:   ErrTokenNotFound,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.Validate(tt.token)
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantValid && got != userID {
				t.Errorf("Validate() = %v, want %v", got, userID)
			}
			if !tt.wantValid && got != uuid.Nil {
				t.Errorf("Validate() = %v, want uuid.Nil", got)
			}
		})
	}
}

func TestTokenStoreValidateExpired(t *testing.T) {
	store := NewTokenStore(1 * time.Millisecond)
	userID := uuid.New()

	token, err := store.Create(userID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Wait for token to expire
	time.Sleep(5 * time.Millisecond)

	got, err := store.Validate(token)
	if err != ErrTokenExpired {
		t.Errorf("Validate() error = %v, want ErrTokenExpired", err)
	}
	if got != uuid.Nil {
		t.Errorf("Validate() = %v, want uuid.Nil", got)
	}

	// Token should be removed after validation of expired
	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0 (expired token should be removed)", store.Count())
	}
}

func TestTokenStoreValidateRefreshesExpiry(t *testing.T) {
	store := NewTokenStore(100 * time.Millisecond)
	userID := uuid.New()

	token, err := store.Create(userID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	store.mu.RLock()
	originalExpiry := store.tokens[token].ExpiresAt
	store.mu.RUnlock()

	// Wait a bit and validate to refresh
	time.Sleep(10 * time.Millisecond)

	_, err = store.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	store.mu.RLock()
	newExpiry := store.tokens[token].ExpiresAt
	store.mu.RUnlock()

	if !newExpiry.After(originalExpiry) {
		t.Error("expiry was not refreshed after Validate()")
	}
}

func TestTokenStoreInvalidate(t *testing.T) {
	store := NewTokenStore(5 * time.Minute)

	token, _ := store.Create(uuid.New())
	if store.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", store.Count())
	}

	store.Invalidate(token)

	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0 after Invalidate()", store.Count())
	}

	// Validate should fail
	_, err := store.Validate(token)
	if err != ErrTokenNotFound {
		t.Errorf("Validate() after Invalidate() error = %v, want ErrTokenNotFound", err)
	}
}

func TestTokenStoreInvalidateNonExistent(t *testing.T) {
	store := NewTokenStore(5 * time.Minute)

	// Should not panic
	store.Invalidate("non-existent-token")
	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}
}

func TestTokenStoreCleanupExpired(t *testing.T) {
	store := NewTokenStore(1 * time.Millisecond)

	// Create 5 tokens
	for i := 0; i < 5; i++ {
		store.Create(uuid.New())
	}

	if store.Count() != 5 {
		t.Fatalf("Count() = %d, want 5", store.Count())
	}

	// Wait for all to expire
	time.Sleep(5 * time.Millisecond)

	removed := store.CleanupExpired()
	if removed != 5 {
		t.Errorf("CleanupExpired() = %d, want 5", removed)
	}

	if store.Count() != 0 {
		t.Errorf("Count() after cleanup = %d, want 0", store.Count())
	}
}

func TestTokenStoreCleanupExpiredPartial(t *testing.T) {
	// Create store with longer TTL
	store := NewTokenStore(1 * time.Hour)

	// Create 3 tokens that won't expire
	for i := 0; i < 3; i++ {
		store.Create(uuid.New())
	}

	// Manually expire 2 of them
	store.mu.Lock()
	count := 0
	for _, tt := range store.tokens {
		if count < 2 {
			tt.ExpiresAt = time.Now().Add(-1 * time.Hour)
			count++
		}
	}
	store.mu.Unlock()

	removed := store.CleanupExpired()
	if removed != 2 {
		t.Errorf("CleanupExpired() = %d, want 2", removed)
	}

	if store.Count() != 1 {
		t.Errorf("Count() after partial cleanup = %d, want 1", store.Count())
	}
}

func TestTokenStoreCount(t *testing.T) {
	store := NewTokenStore(5 * time.Minute)

	if store.Count() != 0 {
		t.Errorf("Count() empty store = %d, want 0", store.Count())
	}

	store.Create(uuid.New())
	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1", store.Count())
	}

	store.Create(uuid.New())
	if store.Count() != 2 {
		t.Errorf("Count() = %d, want 2", store.Count())
	}
}

func TestTokenStoreStartCleanup(t *testing.T) {
	store := NewTokenStore(1 * time.Millisecond)

	// Create token
	store.Create(uuid.New())
	if store.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", store.Count())
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start cleanup with short interval
	store.StartCleanup(ctx, 5*time.Millisecond)

	// Wait for cleanup to run
	time.Sleep(20 * time.Millisecond)

	// Token should be cleaned up
	if store.Count() != 0 {
		t.Errorf("Count() after cleanup = %d, want 0", store.Count())
	}

	// Stop cleanup
	cancel()
}

func TestTokenStoreStartCleanupCancellation(t *testing.T) {
	store := NewTokenStore(1 * time.Hour)

	ctx, cancel := context.WithCancel(context.Background())

	// Start cleanup
	store.StartCleanup(ctx, 1*time.Second)

	// Immediately cancel
	cancel()

	// Create a token after cancellation
	store.Create(uuid.New())

	// Give some time for potential cleanup
	time.Sleep(10 * time.Millisecond)

	// Token should still exist (cleanup was cancelled)
	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1 (cleanup should have been cancelled)", store.Count())
	}
}

func TestTokenStoreConcurrency(t *testing.T) {
	store := NewTokenStore(5 * time.Minute)
	var wg sync.WaitGroup
	iterations := 100

	// Create tokens concurrently
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			store.Create(uuid.New())
		}()
	}
	wg.Wait()

	if store.Count() != iterations {
		t.Errorf("Count() = %d, want %d", store.Count(), iterations)
	}

	// Validate and invalidate concurrently
	store.mu.RLock()
	var tokens []string
	for token := range store.tokens {
		tokens = append(tokens, token)
	}
	store.mu.RUnlock()

	wg.Add(len(tokens) * 2)
	for _, token := range tokens {
		tok := token
		go func() {
			defer wg.Done()
			store.Validate(tok)
		}()
		go func() {
			defer wg.Done()
			store.Invalidate(tok)
		}()
	}
	wg.Wait()

	// No panics means success - final count is indeterminate due to race
}

func TestGenerateToken(t *testing.T) {
	tokens := make(map[string]bool)

	// Generate multiple tokens and check uniqueness
	for i := 0; i < 100; i++ {
		token, err := generateToken()
		if err != nil {
			t.Fatalf("generateToken() error = %v", err)
		}
		if token == "" {
			t.Error("generateToken() returned empty string")
		}
		if len(token) < 32 {
			t.Errorf("generateToken() length = %d, want >= 32", len(token))
		}
		if tokens[token] {
			t.Errorf("duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}
