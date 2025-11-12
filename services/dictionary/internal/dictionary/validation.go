package dictionary

import (
	"context"

	"github.com/google/uuid"
)

// ValidateCreateSet validates a Set for creation.
func ValidateCreateSet(ctx context.Context, set *Set) []string {
	var errors []string

	if set.Name == "" {
		errors = append(errors, "name is required")
	}

	if set.Label == "" {
		errors = append(errors, "label is required")
	}

	return errors
}

// ValidateUpdateSet validates a Set for update.
func ValidateUpdateSet(ctx context.Context, id uuid.UUID, set *Set) []string {
	var errors []string

	if id == uuid.Nil {
		errors = append(errors, "id is required")
	}

	if set.Name == "" {
		errors = append(errors, "name is required")
	}

	if set.Label == "" {
		errors = append(errors, "label is required")
	}

	return errors
}

// ValidateDeleteSet validates a Set for deletion.
func ValidateDeleteSet(ctx context.Context, id uuid.UUID) []string {
	var errors []string

	if id == uuid.Nil {
		errors = append(errors, "id is required")
	}

	return errors
}

// ValidateCreateOption validates an Option for creation.
func ValidateCreateOption(ctx context.Context, option *Option) []string {
	var errors []string

	if option.Set == uuid.Nil {
		errors = append(errors, "set_id is required")
	}

	if option.Key == "" {
		errors = append(errors, "key is required")
	}

	if option.Label == "" {
		errors = append(errors, "label is required")
	}

	if option.Value == "" {
		errors = append(errors, "value is required")
	}

	return errors
}

// ValidateUpdateOption validates an Option for update.
func ValidateUpdateOption(ctx context.Context, id uuid.UUID, option *Option) []string {
	var errors []string

	if id == uuid.Nil {
		errors = append(errors, "id is required")
	}

	if option.Set == uuid.Nil {
		errors = append(errors, "set_id is required")
	}

	if option.Key == "" {
		errors = append(errors, "key is required")
	}

	if option.Label == "" {
		errors = append(errors, "label is required")
	}

	if option.Value == "" {
		errors = append(errors, "value is required")
	}

	return errors
}

// ValidateDeleteOption validates an Option for deletion.
func ValidateDeleteOption(ctx context.Context, id uuid.UUID) []string {
	var errors []string

	if id == uuid.Nil {
		errors = append(errors, "id is required")
	}

	return errors
}
