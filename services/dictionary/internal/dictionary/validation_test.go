package dictionary

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestValidateCreateSet(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		set      *Set
		wantErrs int
	}{
		{
			name: "valid set",
			set: &Set{
				Name:  "test_set",
				Label: "Test Set",
			},
			wantErrs: 0,
		},
		{
			name: "missing name",
			set: &Set{
				Label: "Test Set",
			},
			wantErrs: 1,
		},
		{
			name: "missing label",
			set: &Set{
				Name: "test_set",
			},
			wantErrs: 1,
		},
		{
			name:     "missing both",
			set:      &Set{},
			wantErrs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateCreateSet(ctx, tt.set)
			if len(errs) != tt.wantErrs {
				t.Errorf("ValidateCreateSet() got %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestValidateCreateOption(t *testing.T) {
	ctx := context.Background()
	setID := uuid.New()

	tests := []struct {
		name     string
		option   *Option
		wantErrs int
	}{
		{
			name: "valid option",
			option: &Option{
				Set:   setID,
				Key:   "test_key",
				Label: "Test Label",
				Value: "Test Value",
			},
			wantErrs: 0,
		},
		{
			name: "missing set_id",
			option: &Option{
				Key:   "test_key",
				Label: "Test Label",
				Value: "Test Value",
			},
			wantErrs: 1,
		},
		{
			name: "missing key",
			option: &Option{
				Set:   setID,
				Label: "Test Label",
				Value: "Test Value",
			},
			wantErrs: 1,
		},
		{
			name: "missing label",
			option: &Option{
				Set:   setID,
				Key:   "test_key",
				Value: "Test Value",
			},
			wantErrs: 1,
		},
		{
			name: "missing value",
			option: &Option{
				Set:   setID,
				Key:   "test_key",
				Label: "Test Label",
			},
			wantErrs: 1,
		},
		{
			name:     "missing all fields",
			option:   &Option{},
			wantErrs: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateCreateOption(ctx, tt.option)
			if len(errs) != tt.wantErrs {
				t.Errorf("ValidateCreateOption() got %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}
