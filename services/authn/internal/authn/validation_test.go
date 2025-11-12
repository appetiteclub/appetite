package authn

import (
	"context"
	"testing"
	"time"

	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/google/uuid"
)

func TestValidateCreateUser(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New()
	validTime := time.Now()

	tests := []struct {
		name           string
		user           User
		expectedCount  int
		expectedFields []string
	}{
		{
			name: "valid user",
			user: User{
				ID:           validID,
				EmailLookup:  []byte("test@example.com"),
				PasswordHash: []byte("hashed_password"),
				PasswordSalt: []byte("salt"),
				Status:       authpkg.UserStatusActive,
				CreatedAt:    validTime,
				UpdatedAt:    validTime,
			},
			expectedCount: 0,
		},
		{
			name: "missing ID",
			user: User{
				ID:           uuid.Nil,
				EmailLookup:  []byte("test@example.com"),
				PasswordHash: []byte("hashed_password"),
				PasswordSalt: []byte("salt"),
				Status:       authpkg.UserStatusActive,
				CreatedAt:    validTime,
				UpdatedAt:    validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"id"},
		},
		{
			name: "missing email lookup",
			user: User{
				ID:           validID,
				EmailLookup:  []byte{},
				PasswordHash: []byte("hashed_password"),
				PasswordSalt: []byte("salt"),
				Status:       authpkg.UserStatusActive,
				CreatedAt:    validTime,
				UpdatedAt:    validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"email"},
		},
		{
			name: "missing password hash",
			user: User{
				ID:           validID,
				EmailLookup:  []byte("test@example.com"),
				PasswordHash: []byte{},
				PasswordSalt: []byte("salt"),
				Status:       authpkg.UserStatusActive,
				CreatedAt:    validTime,
				UpdatedAt:    validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"password"},
		},
		{
			name: "missing password salt",
			user: User{
				ID:           validID,
				EmailLookup:  []byte("test@example.com"),
				PasswordHash: []byte("hashed_password"),
				PasswordSalt: []byte{},
				Status:       authpkg.UserStatusActive,
				CreatedAt:    validTime,
				UpdatedAt:    validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"password"},
		},
		{
			name: "invalid status",
			user: User{
				ID:           validID,
				EmailLookup:  []byte("test@example.com"),
				PasswordHash: []byte("hashed_password"),
				PasswordSalt: []byte("salt"),
				Status:       authpkg.UserStatus("invalid"),
				CreatedAt:    validTime,
				UpdatedAt:    validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"status"},
		},
		{
			name: "missing created at",
			user: User{
				ID:           validID,
				EmailLookup:  []byte("test@example.com"),
				PasswordHash: []byte("hashed_password"),
				PasswordSalt: []byte("salt"),
				Status:       authpkg.UserStatusActive,
				CreatedAt:    time.Time{}, // zero time
				UpdatedAt:    validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"created_at"},
		},
		{
			name: "missing updated at",
			user: User{
				ID:           validID,
				EmailLookup:  []byte("test@example.com"),
				PasswordHash: []byte("hashed_password"),
				PasswordSalt: []byte("salt"),
				Status:       authpkg.UserStatusActive,
				CreatedAt:    validTime,
				UpdatedAt:    time.Time{}, // zero time
			},
			expectedCount:  1,
			expectedFields: []string{"updated_at"},
		},
		{
			name: "multiple errors",
			user: User{
				ID:           uuid.Nil,
				EmailLookup:  []byte{},
				PasswordHash: []byte{},
				PasswordSalt: []byte{},
				Status:       authpkg.UserStatus("invalid"),
				CreatedAt:    time.Time{},
				UpdatedAt:    time.Time{},
			},
			expectedCount:  7,
			expectedFields: []string{"id", "email", "password", "password", "status", "created_at", "updated_at"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateCreateUser(ctx, tt.user)

			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateCreateUser() returned %d errors, want %d", len(errors), tt.expectedCount)
				for _, err := range errors {
					t.Logf("  - %s: %s", err.Field, err.Message)
				}
			}

			if tt.expectedFields != nil {
				for i, expectedField := range tt.expectedFields {
					if i >= len(errors) {
						t.Errorf("Expected error for field %s but not enough errors returned", expectedField)
						break
					}
					if errors[i].Field != expectedField {
						t.Errorf("Expected error field %s at position %d, got %s", expectedField, i, errors[i].Field)
					}
				}
			}
		})
	}
}

func TestValidateUpdateUser(t *testing.T) {
	ctx := context.Background()
	validID := uuid.New()
	differentID := uuid.New()
	validTime := time.Now()

	tests := []struct {
		name           string
		id             uuid.UUID
		user           User
		expectedCount  int
		expectedFields []string
	}{
		{
			name: "valid update",
			id:   validID,
			user: User{
				ID:          validID,
				EmailLookup: []byte("test@example.com"),
				Status:      authpkg.UserStatusActive,
				UpdatedAt:   validTime,
			},
			expectedCount: 0,
		},
		{
			name: "ID mismatch",
			id:   validID,
			user: User{
				ID:          differentID,
				EmailLookup: []byte("test@example.com"),
				Status:      authpkg.UserStatusActive,
				UpdatedAt:   validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"id"},
		},
		{
			name: "missing email lookup",
			id:   validID,
			user: User{
				ID:          validID,
				EmailLookup: []byte{},
				Status:      authpkg.UserStatusActive,
				UpdatedAt:   validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"email"},
		},
		{
			name: "invalid status",
			id:   validID,
			user: User{
				ID:          validID,
				EmailLookup: []byte("test@example.com"),
				Status:      authpkg.UserStatus("invalid"),
				UpdatedAt:   validTime,
			},
			expectedCount:  1,
			expectedFields: []string{"status"},
		},
		{
			name: "missing updated at",
			id:   validID,
			user: User{
				ID:          validID,
				EmailLookup: []byte("test@example.com"),
				Status:      authpkg.UserStatusActive,
				UpdatedAt:   time.Time{},
			},
			expectedCount:  1,
			expectedFields: []string{"updated_at"},
		},
		{
			name: "multiple errors",
			id:   validID,
			user: User{
				ID:          differentID,
				EmailLookup: []byte{},
				Status:      authpkg.UserStatus("invalid"),
				UpdatedAt:   time.Time{},
			},
			expectedCount:  4,
			expectedFields: []string{"id", "email", "status", "updated_at"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateUpdateUser(ctx, tt.id, tt.user)

			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateUpdateUser() returned %d errors, want %d", len(errors), tt.expectedCount)
				for _, err := range errors {
					t.Logf("  - %s: %s", err.Field, err.Message)
				}
			}

			if tt.expectedFields != nil {
				for i, expectedField := range tt.expectedFields {
					if i >= len(errors) {
						t.Errorf("Expected error for field %s but not enough errors returned", expectedField)
						break
					}
					if errors[i].Field != expectedField {
						t.Errorf("Expected error field %s at position %d, got %s", expectedField, i, errors[i].Field)
					}
				}
			}
		})
	}
}

func TestValidateDeleteUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		id             uuid.UUID
		expectedCount  int
		expectedFields []string
	}{
		{
			name:          "valid ID",
			id:            uuid.New(),
			expectedCount: 0,
		},
		{
			name:           "nil ID",
			id:             uuid.Nil,
			expectedCount:  1,
			expectedFields: []string{"id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateDeleteUser(ctx, tt.id)

			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateDeleteUser() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			if tt.expectedFields != nil {
				for i, expectedField := range tt.expectedFields {
					if i >= len(errors) {
						t.Errorf("Expected error for field %s but not enough errors returned", expectedField)
						break
					}
					if errors[i].Field != expectedField {
						t.Errorf("Expected error field %s at position %d, got %s", expectedField, i, errors[i].Field)
					}
				}
			}
		})
	}
}

func TestValidateSignUpRequest(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		password       string
		username       string
		displayName    string
		expectedCount  int
		expectedFields []string
	}{
		{
			name:          "valid signup",
			email:         "test@example.com",
			password:      "ValidPassword123!",
			username:      "testuser",
			displayName:   "Test User",
			expectedCount: 0,
		},
		{
			name:           "empty email",
			email:          "",
			password:       "ValidPassword123!",
			username:       "validuser",
			displayName:    "Valid Name",
			expectedCount:  1,
			expectedFields: []string{"email"},
		},
		{
			name:           "invalid email format",
			email:          "not-an-email",
			password:       "ValidPassword123!",
			username:       "validuser",
			displayName:    "Valid Name",
			expectedCount:  1,
			expectedFields: []string{"email"},
		},
		{
			name:           "empty password",
			email:          "test@example.com",
			password:       "",
			username:       "validuser",
			displayName:    "Valid Name",
			expectedCount:  1,
			expectedFields: []string{"password"},
		},
		{
			name:           "weak password",
			email:          "test@example.com",
			password:       "123",
			username:       "validuser",
			displayName:    "Valid Name",
			expectedCount:  4,
			expectedFields: []string{"password", "password", "password", "password"},
		},
		{
			name:           "both invalid",
			email:          "not-an-email",
			password:       "weak",
			username:       "validuser",
			displayName:    "Valid Name",
			expectedCount:  5,
			expectedFields: []string{"email", "password", "password", "password", "password"},
		},
		{
			name:           "invalid username",
			email:          "ok@example.com",
			password:       "ValidPassword123!",
			username:       "ab",
			displayName:    "Valid Name",
			expectedCount:  1,
			expectedFields: []string{"username"},
		},
		{
			name:           "invalid name",
			email:          "ok@example.com",
			password:       "ValidPassword123!",
			username:       "validuser",
			displayName:    "a",
			expectedCount:  1,
			expectedFields: []string{"name"},
		},
		{
			name:           "missing name",
			email:          "ok@example.com",
			password:       "ValidPassword123!",
			username:       "validuser",
			displayName:    "",
			expectedCount:  1,
			expectedFields: []string{"name"},
		},
		{
			name:           "missing username",
			email:          "ok@example.com",
			password:       "ValidPassword123!",
			username:       "",
			displayName:    "Valid Name",
			expectedCount:  1,
			expectedFields: []string{"username"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateSignUpRequest(tt.email, tt.password, tt.username, tt.displayName)

			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateSignUpRequest() returned %d errors, want %d", len(errors), tt.expectedCount)
				for _, err := range errors {
					t.Logf("  - %s: %s", err.Field, err.Message)
				}
			}

			if tt.expectedFields != nil {
				fieldCount := make(map[string]int)
				for _, err := range errors {
					fieldCount[err.Field]++
				}

				expectedFieldCount := make(map[string]int)
				for _, field := range tt.expectedFields {
					expectedFieldCount[field]++
				}

				for field, expectedCount := range expectedFieldCount {
					if fieldCount[field] != expectedCount {
						t.Errorf("Expected %d error(s) for field %s, got %d", expectedCount, field, fieldCount[field])
					}
				}
			}
		})
	}
}

func TestValidateSignInRequest(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		password       string
		expectedCount  int
		expectedFields []string
	}{
		{
			name:          "valid signin",
			email:         "test@example.com",
			password:      "any-password",
			expectedCount: 0,
		},
		{
			name:          "valid signin with simple password",
			email:         "test@example.com",
			password:      "123", // signin is less strict
			expectedCount: 0,
		},
		{
			name:           "empty email",
			email:          "",
			password:       "password",
			expectedCount:  1,
			expectedFields: []string{"email"},
		},
		{
			name:           "empty password",
			email:          "test@example.com",
			password:       "",
			expectedCount:  1,
			expectedFields: []string{"password"},
		},
		{
			name:           "both empty",
			email:          "",
			password:       "",
			expectedCount:  2,
			expectedFields: []string{"email", "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateSignInRequest(tt.email, tt.password)

			if len(errors) != tt.expectedCount {
				t.Errorf("ValidateSignInRequest() returned %d errors, want %d", len(errors), tt.expectedCount)
				for _, err := range errors {
					t.Logf("  - %s: %s", err.Field, err.Message)
				}
			}

			if tt.expectedFields != nil {
				for i, expectedField := range tt.expectedFields {
					if i >= len(errors) {
						t.Errorf("Expected error for field %s but not enough errors returned", expectedField)
						break
					}
					if errors[i].Field != expectedField {
						t.Errorf("Expected error field %s at position %d, got %s", expectedField, i, errors[i].Field)
					}
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	// Test ValidationError struct
	err := ValidationError{
		Field:   "test_field",
		Message: "test_message",
	}

	if err.Field != "test_field" {
		t.Errorf("ValidationError.Field = %v, want %v", err.Field, "test_field")
	}

	if err.Message != "test_message" {
		t.Errorf("ValidationError.Message = %v, want %v", err.Message, "test_message")
	}
}
