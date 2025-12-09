package operations

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		wantIsNil bool
	}{
		{
			name: "withValidUserID",
			setupCtx: func() context.Context {
				userID := uuid.New()
				return context.WithValue(context.Background(), contextKeyUserID, userID)
			},
			wantIsNil: false,
		},
		{
			name: "withNilContext",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantIsNil: true,
		},
		{
			name: "withWrongType",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), contextKeyUserID, "not-a-uuid")
			},
			wantIsNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			got := getUserIDFromContext(ctx)

			if tt.wantIsNil && got != uuid.Nil {
				t.Errorf("getUserIDFromContext() = %v, want uuid.Nil", got)
			}
			if !tt.wantIsNil && got == uuid.Nil {
				t.Error("getUserIDFromContext() = uuid.Nil, want valid UUID")
			}
		})
	}
}

func TestGetTokenFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		want     string
	}{
		{
			name: "withValidToken",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), contextKeyToken, "my-test-token")
			},
			want: "my-test-token",
		},
		{
			name: "withEmptyContext",
			setupCtx: func() context.Context {
				return context.Background()
			},
			want: "",
		},
		{
			name: "withWrongType",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), contextKeyToken, 12345)
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			got := getTokenFromContext(ctx)

			if got != tt.want {
				t.Errorf("getTokenFromContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
