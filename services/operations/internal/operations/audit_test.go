package operations

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewAuditLogger(t *testing.T) {
	tests := []struct {
		name   string
		logger bool
	}{
		{
			name:   "withNilLogger",
			logger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewAuditLogger(nil)
			if logger == nil {
				t.Fatal("NewAuditLogger() returned nil")
			}
			if logger.logger == nil {
				t.Error("logger should default to noop logger")
			}
		})
	}
}

func TestAuditLoggerLog(t *testing.T) {
	logger := NewAuditLogger(nil)
	ctx := context.Background()

	tests := []struct {
		name  string
		entry AuditEntry
	}{
		{
			name: "basicEntry",
			entry: AuditEntry{
				UserID:  uuid.New(),
				Action:  "test-action",
				Target:  "test-target",
				Success: true,
			},
		},
		{
			name: "entryWithTimestamp",
			entry: AuditEntry{
				UserID:    uuid.New(),
				Action:    "test-action",
				Target:    "test-target",
				Success:   true,
				Timestamp: time.Now(),
			},
		},
		{
			name: "failedEntry",
			entry: AuditEntry{
				UserID:  uuid.New(),
				Action:  "failed-action",
				Target:  "test-target",
				Success: false,
				Error:   "test error message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			logger.Log(ctx, tt.entry)
		})
	}
}

func TestAuditLoggerLogCommand(t *testing.T) {
	logger := NewAuditLogger(nil)
	ctx := context.Background()

	tests := []struct {
		name     string
		userID   uuid.UUID
		command  string
		params   []string
		success  bool
		errorMsg string
	}{
		{
			name:     "successfulCommand",
			userID:   uuid.New(),
			command:  "list-orders",
			params:   []string{},
			success:  true,
			errorMsg: "",
		},
		{
			name:     "commandWithParams",
			userID:   uuid.New(),
			command:  "get-order",
			params:   []string{"123"},
			success:  true,
			errorMsg: "",
		},
		{
			name:     "failedCommand",
			userID:   uuid.New(),
			command:  "invalid-cmd",
			params:   []string{},
			success:  false,
			errorMsg: "command not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			logger.LogCommand(ctx, tt.userID, tt.command, tt.params, tt.success, tt.errorMsg)
		})
	}
}

func TestAuditLoggerLogLogin(t *testing.T) {
	logger := NewAuditLogger(nil)
	ctx := context.Background()

	userID := uuid.New()

	// Should not panic
	logger.LogLogin(ctx, userID)
}

func TestAuditLoggerLogLogout(t *testing.T) {
	logger := NewAuditLogger(nil)
	ctx := context.Background()

	userID := uuid.New()

	// Should not panic
	logger.LogLogout(ctx, userID)
}

func TestAuditEntryFields(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	entry := AuditEntry{
		UserID:    userID,
		Action:    "test-action",
		Target:    "test-target",
		Payload:   []byte(`{"key": "value"}`),
		Timestamp: now,
		Success:   true,
		Error:     "",
	}

	if entry.UserID != userID {
		t.Errorf("UserID = %v, want %v", entry.UserID, userID)
	}
	if entry.Action != "test-action" {
		t.Errorf("Action = %q, want %q", entry.Action, "test-action")
	}
	if entry.Target != "test-target" {
		t.Errorf("Target = %q, want %q", entry.Target, "test-target")
	}
	if !entry.Success {
		t.Error("Success should be true")
	}
	if entry.Timestamp != now {
		t.Errorf("Timestamp = %v, want %v", entry.Timestamp, now)
	}
}
