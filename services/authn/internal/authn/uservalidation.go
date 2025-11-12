package authn

import (
	"context"
	"strings"

	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/google/uuid"
)

// UserCreateRequest represents the payload for creating a user
type UserCreateRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Status   string `json:"status,omitempty"`
}

// UserUpdateRequest represents the payload for updating a user
type UserUpdateRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Status   string `json:"status,omitempty"`
}

// ValidateCreateUserRequest validates the incoming request for creating a user
func ValidateCreateUserRequest(ctx context.Context, req UserCreateRequest) []string {
	var errors []string

	// Name validation
	if strings.TrimSpace(req.Name) == "" {
		errors = append(errors, "name is required")
	} else if len(req.Name) > 100 {
		errors = append(errors, "name must be 100 characters or less")
	}

	// Username validation
	if strings.TrimSpace(req.Username) == "" {
		errors = append(errors, "username is required")
	} else if msg := validateUsername(req.Username); msg != "" {
		errors = append(errors, msg)
	}

	// Email validation
	if strings.TrimSpace(req.Email) == "" {
		errors = append(errors, "email is required")
	} else if len(req.Email) > 255 {
		errors = append(errors, "email must be 255 characters or less")
	} else if !isValidEmail(req.Email) {
		errors = append(errors, "email format is invalid")
	}

	// Password validation
	if strings.TrimSpace(req.Password) == "" {
		errors = append(errors, "password is required")
	} else if len(req.Password) < 8 {
		errors = append(errors, "password must be at least 8 characters")
	} else if len(req.Password) > 255 {
		errors = append(errors, "password must be 255 characters or less")
	}

	// Status validation (optional, default to "active")
	if req.Status != "" {
		validStatuses := []string{"active", "inactive", "pending"}
		valid := false
		for _, status := range validStatuses {
			if req.Status == status {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, "status must be one of: active, inactive, pending")
		}
	}

	return errors
}

// ValidateUpdateUserRequest validates the incoming request for updating a user
func ValidateUpdateUserRequest(ctx context.Context, id uuid.UUID, req UserUpdateRequest) []string {
	var errors []string

	// ID validation
	if id == uuid.Nil {
		errors = append(errors, "id is required")
	}

	// Name validation (required for updates)
	if strings.TrimSpace(req.Name) == "" {
		errors = append(errors, "name is required")
	} else if len(req.Name) > 100 {
		errors = append(errors, "name must be 100 characters or less")
	}

	// Username validation (required)
	if strings.TrimSpace(req.Username) == "" {
		errors = append(errors, "username is required")
	} else if msg := validateUsername(req.Username); msg != "" {
		errors = append(errors, msg)
	}

	// Email validation (required for updates)
	if strings.TrimSpace(req.Email) == "" {
		errors = append(errors, "email is required")
	} else if len(req.Email) > 255 {
		errors = append(errors, "email must be 255 characters or less")
	} else if !isValidEmail(req.Email) {
		errors = append(errors, "email format is invalid")
	}

	// Password validation (optional for updates)
	if req.Password != "" {
		if len(req.Password) < 8 {
			errors = append(errors, "password must be at least 8 characters")
		} else if len(req.Password) > 255 {
			errors = append(errors, "password must be 255 characters or less")
		}
	}

	// Status validation (optional)
	if req.Status != "" {
		validStatuses := []string{"active", "inactive", "pending"}
		valid := false
		for _, status := range validStatuses {
			if req.Status == status {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, "status must be one of: active, inactive, pending")
		}
	}

	return errors
}

// Helper function for email validation
func isValidEmail(email string) bool {
	// Basic email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	local := parts[0]
	domain := parts[1]

	if len(local) == 0 || len(domain) == 0 {
		return false
	}

	if !strings.Contains(domain, ".") {
		return false
	}

	return true
}

// Convert request to User entity
func (req UserCreateRequest) ToUser() User {
	user := User{}
	user.Name = strings.TrimSpace(req.Name)
	if normalized, err := normalizeUsername(req.Username); err == nil {
		user.Username = normalized
	}

	// Set status with proper type conversion
	if req.Status != "" {
		user.Status = authpkg.UserStatus(req.Status)
	} else {
		user.Status = authpkg.UserStatusActive
	}

	return user
}

// Convert request to User entity for updates
func (req UserUpdateRequest) ToUser() User {
	user := User{}
	user.Name = strings.TrimSpace(req.Name)
	if normalized, err := normalizeUsername(req.Username); err == nil {
		user.Username = normalized
	}

	// Set status with proper type conversion
	if req.Status != "" {
		user.Status = authpkg.UserStatus(req.Status)
	} else {
		user.Status = authpkg.UserStatusActive
	}

	return user
}
