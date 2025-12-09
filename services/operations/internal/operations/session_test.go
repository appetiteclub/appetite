package operations

import (
	"sync"
	"testing"
	"time"
)

func TestNewSessionStore(t *testing.T) {
	secret := []byte("test-secret")
	ttl := 30 * time.Minute

	store := NewSessionStore(secret, ttl)
	if store == nil {
		t.Fatal("NewSessionStore() returned nil")
	}
	if store.sessions == nil {
		t.Error("sessions map is nil")
	}
	if store.ttl != ttl {
		t.Errorf("ttl = %v, want %v", store.ttl, ttl)
	}
}

func TestSessionStoreSave(t *testing.T) {
	store := NewSessionStore([]byte("secret"), time.Hour)

	tests := []struct {
		name    string
		session *Session
		wantErr bool
	}{
		{
			name: "validSession",
			session: &Session{
				ID:        "session-1",
				UserID:    "user-1",
				Username:  "testuser",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(time.Hour),
			},
			wantErr: false,
		},
		{
			name:    "nilSession",
			session: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Save(tt.session)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSessionStoreGet(t *testing.T) {
	store := NewSessionStore([]byte("secret"), time.Hour)

	// Save a valid session
	validSession := &Session{
		ID:        "valid-session",
		UserID:    "user-1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	store.Save(validSession)

	// Save an expired session
	expiredSession := &Session{
		ID:        "expired-session",
		UserID:    "user-2",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	store.Save(expiredSession)

	tests := []struct {
		name      string
		sessionID string
		wantErr   bool
	}{
		{
			name:      "existingSession",
			sessionID: "valid-session",
			wantErr:   false,
		},
		{
			name:      "nonExistentSession",
			sessionID: "non-existent",
			wantErr:   true,
		},
		{
			name:      "expiredSession",
			sessionID: "expired-session",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.Get(tt.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("Get() returned nil for valid session")
			}
		})
	}
}

func TestSessionStoreDelete(t *testing.T) {
	store := NewSessionStore([]byte("secret"), time.Hour)

	session := &Session{
		ID:        "session-to-delete",
		UserID:    "user-1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	store.Save(session)

	// Verify session exists
	_, err := store.Get("session-to-delete")
	if err != nil {
		t.Fatalf("session not found after Save(): %v", err)
	}

	// Delete session
	store.Delete("session-to-delete")

	// Verify session is deleted
	_, err = store.Get("session-to-delete")
	if err == nil {
		t.Error("session still found after Delete()")
	}
}

func TestSessionStoreDeleteNonExistent(t *testing.T) {
	store := NewSessionStore([]byte("secret"), time.Hour)

	// Should not panic
	store.Delete("non-existent-session")
}

func TestSessionStoreConcurrency(t *testing.T) {
	store := NewSessionStore([]byte("secret"), time.Hour)
	var wg sync.WaitGroup
	iterations := 50

	// Concurrent saves
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(id int) {
			defer wg.Done()
			session := &Session{
				ID:        string(rune('a' + id%26)),
				UserID:    "user",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(time.Hour),
			}
			store.Save(session)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(id int) {
			defer wg.Done()
			store.Get(string(rune('a' + id%26)))
		}(i)
	}
	wg.Wait()

	// No panics means success
}

func TestSessionFields(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:        "test-id",
		UserID:    "user-123",
		Username:  "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}

	if session.ID != "test-id" {
		t.Errorf("ID = %q, want %q", session.ID, "test-id")
	}
	if session.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", session.UserID, "user-123")
	}
	if session.Username != "testuser" {
		t.Errorf("Username = %q, want %q", session.Username, "testuser")
	}
	if session.Name != "Test User" {
		t.Errorf("Name = %q, want %q", session.Name, "Test User")
	}
	if session.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", session.Email, "test@example.com")
	}
}
