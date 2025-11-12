package authn

import (
	"context"
	"strings"

	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/google/uuid"
)

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidateCreateUser validates a User entity before creation.
func ValidateCreateUser(ctx context.Context, user User) []ValidationError {
	var errors []ValidationError

	// ID should be set
	if user.ID == uuid.Nil {
		errors = append(errors, ValidationError{
			Field:   "id",
			Message: "ID is required",
		})
	}

	// Email lookup should be set (computed from email)
	if len(user.EmailLookup) == 0 {
		errors = append(errors, ValidationError{
			Field:   "email",
			Message: "Email lookup is required",
		})
	}

	// Password hash should be set
	if len(user.PasswordHash) == 0 {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Password hash is required",
		})
	}

	// Password salt should be set
	if len(user.PasswordSalt) == 0 {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Password salt is required",
		})
	}

	// Status should be valid
	if statusErrors := authpkg.ValidateUserStatus(authpkg.UserStatus(user.Status)); len(statusErrors) > 0 {
		errors = append(errors, ValidationError{
			Field:   "status",
			Message: "Invalid user status",
		})
	}

	// CreatedAt should be set
	if user.CreatedAt.IsZero() {
		errors = append(errors, ValidationError{
			Field:   "created_at",
			Message: "Created timestamp is required",
		})
	}

	// UpdatedAt should be set
	if user.UpdatedAt.IsZero() {
		errors = append(errors, ValidationError{
			Field:   "updated_at",
			Message: "Updated timestamp is required",
		})
	}

	return errors
}

// ValidateUpdateUser validates a User entity before update.
func ValidateUpdateUser(ctx context.Context, id uuid.UUID, user User) []ValidationError {
	var errors []ValidationError

	// ID should match parameter
	if user.ID != id {
		errors = append(errors, ValidationError{
			Field:   "id",
			Message: "ID mismatch",
		})
	}

	// Email lookup should be set (for existing user)
	if len(user.EmailLookup) == 0 {
		errors = append(errors, ValidationError{
			Field:   "email",
			Message: "Email lookup is required",
		})
	}

	// Status should be valid
	if statusErrors := authpkg.ValidateUserStatus(authpkg.UserStatus(user.Status)); len(statusErrors) > 0 {
		errors = append(errors, ValidationError{
			Field:   "status",
			Message: "Invalid user status",
		})
	}

	// UpdatedAt should be set
	if user.UpdatedAt.IsZero() {
		errors = append(errors, ValidationError{
			Field:   "updated_at",
			Message: "Updated timestamp is required",
		})
	}

	return errors
}

// ValidateDeleteUser validates a User entity before deletion.
func ValidateDeleteUser(ctx context.Context, id uuid.UUID) []ValidationError {
	var errors []ValidationError

	// ID should be valid
	if id == uuid.Nil {
		errors = append(errors, ValidationError{
			Field:   "id",
			Message: "Valid ID is required for deletion",
		})
	}

	return errors
}

// ValidateSignUpRequest validates signup request data.
func ValidateSignUpRequest(email, password, username, name string) []ValidationError {
	var errors []ValidationError

	// Validate email using pure function
	if emailErrors := authpkg.ValidateEmail(email); len(emailErrors) > 0 {
		for _, err := range emailErrors {
			errors = append(errors, ValidationError{
				Field:   "email",
				Message: err.Message,
			})
		}
	}

	// Validate password using pure function
	if passwordErrors := authpkg.ValidatePassword(password); len(passwordErrors) > 0 {
		for _, err := range passwordErrors {
			errors = append(errors, ValidationError{
				Field:   "password",
				Message: err.Message,
			})
		}
	}

	if err := validateUsername(username); err != "" {
		errors = append(errors, ValidationError{Field: "username", Message: err})
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		errors = append(errors, ValidationError{Field: "name", Message: "Name is required"})
	} else {
		if len([]rune(trimmedName)) < 2 {
			errors = append(errors, ValidationError{Field: "name", Message: "Name must be at least 2 characters"})
		}
		if len([]rune(trimmedName)) > 120 {
			errors = append(errors, ValidationError{Field: "name", Message: "Name must be 120 characters or less"})
		}
	}

	return errors
}

// ValidateSignInRequest validates signin request data.
func ValidateSignInRequest(email, password string) []ValidationError {
	var errors []ValidationError

	// Basic email validation (less strict for signin)
	if email == "" {
		errors = append(errors, ValidationError{
			Field:   "email",
			Message: "Email is required",
		})
	}

	// Basic password validation (less strict for signin)
	if password == "" {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Password is required",
		})
	}

	return errors
}

func validateUsername(value string) string {
	if _, err := normalizeUsername(value); err != nil {
		return err.Error()
	}
	return ""
}
