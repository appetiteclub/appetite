package operations

import (
	"testing"

	"github.com/google/uuid"
)

func TestFormatError(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "simpleMessage",
			message: "Authentication failed",
		},
		{
			name:    "emptyMessage",
			message: "",
		},
		{
			name:    "messageWithSpecialChars",
			message: "Error: <script>alert('xss')</script>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatError(tt.message)
			if result == "" {
				t.Error("formatError() returned empty string")
			}
			// Should contain the error class/styling
			if len(result) < 50 {
				t.Errorf("formatError() result too short, got %d chars", len(result))
			}
		})
	}
}

func TestParseUUID(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "validUUID",
			input:   validUUID.String(),
			wantErr: false,
		},
		{
			name:    "emptyString",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalidFormat",
			input:   "not-a-uuid",
			wantErr: true,
		},
		{
			name:    "nilUUID",
			input:   uuid.Nil.String(),
			wantErr: false,
		},
		{
			name:    "partialUUID",
			input:   "123e4567-e89b-12d3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUUID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseUUID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.input == validUUID.String() && got != validUUID {
				t.Errorf("parseUUID(%q) = %v, want %v", tt.input, got, validUUID)
			}
		})
	}
}
