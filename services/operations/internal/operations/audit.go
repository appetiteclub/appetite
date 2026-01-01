package operations

import (
	"context"
	"encoding/json"
	"time"

	"github.com/appetiteclub/apt"
	"github.com/google/uuid"
)

// AuditEntry represents a single audit log entry for a user action.
type AuditEntry struct {
	UserID    uuid.UUID       `json:"user_id"`
	Action    string          `json:"action"`
	Target    string          `json:"target"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	Success   bool            `json:"success"`
	Error     string          `json:"error,omitempty"`
}

// AuditLogger handles logging of user actions for operational transparency.
type AuditLogger struct {
	logger apt.Logger
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(logger apt.Logger) *AuditLogger {
	if logger == nil {
		logger = apt.NewNoopLogger()
	}
	return &AuditLogger{logger: logger}
}

// Log records an audit entry.
func (a *AuditLogger) Log(ctx context.Context, entry AuditEntry) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	a.logger.Info("audit",
		"user_id", entry.UserID.String(),
		"action", entry.Action,
		"target", entry.Target,
		"success", entry.Success,
		"timestamp", entry.Timestamp.Format(time.RFC3339),
		"error", entry.Error,
	)
}

// LogCommand logs a command execution with its result.
func (a *AuditLogger) LogCommand(ctx context.Context, userID uuid.UUID, command string, params []string, success bool, errorMsg string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"command": command,
		"params":  params,
	})

	entry := AuditEntry{
		UserID:    userID,
		Action:    "execute-command",
		Target:    command,
		Payload:   payload,
		Timestamp: time.Now(),
		Success:   success,
		Error:     errorMsg,
	}

	a.Log(ctx, entry)
}

// LogLogin logs a successful login.
func (a *AuditLogger) LogLogin(ctx context.Context, userID uuid.UUID) {
	entry := AuditEntry{
		UserID:    userID,
		Action:    "login",
		Target:    "auth",
		Timestamp: time.Now(),
		Success:   true,
	}

	a.Log(ctx, entry)
}

// LogLogout logs a logout action.
func (a *AuditLogger) LogLogout(ctx context.Context, userID uuid.UUID) {
	entry := AuditEntry{
		UserID:    userID,
		Action:    "logout",
		Target:    "auth",
		Timestamp: time.Now(),
		Success:   true,
	}

	a.Log(ctx, entry)
}
