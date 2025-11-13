package operations

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyToken  contextKey = "token"
)

func getUserIDFromContext(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value(contextKeyUserID).(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}

func getTokenFromContext(ctx context.Context) string {
	if token, ok := ctx.Value(contextKeyToken).(string); ok {
		return token
	}
	return ""
}
