package operations

import (
	"testing"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want string
	}{
		{
			name: "emptyMap",
			data: map[string]interface{}{},
			want: "{}",
		},
		{
			name: "simpleMap",
			data: map[string]interface{}{
				"key": "value",
			},
			want: `{"key":"value"}`,
		},
		{
			name: "nestedMap",
			data: map[string]interface{}{
				"name": "test",
				"nested": map[string]interface{}{
					"inner": "value",
				},
			},
		},
		{
			name: "nilMap",
			data: nil,
			want: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toJSON(tt.data)
			if tt.want != "" && got != tt.want {
				t.Errorf("toJSON() = %q, want %q", got, tt.want)
			}
			if got == "" {
				t.Error("toJSON() returned empty string")
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{
			name:  "integer",
			input: "10",
			want:  10.0,
		},
		{
			name:  "decimal",
			input: "15.99",
			want:  15.99,
		},
		{
			name:  "zero",
			input: "0",
			want:  0.0,
		},
		{
			name:  "negativeValue",
			input: "-5.50",
			want:  -5.50,
		},
		{
			name:  "invalidString",
			input: "not-a-number",
			want:  0.0,
		},
		{
			name:  "emptyString",
			input: "",
			want:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFloat(tt.input)
			if got != tt.want {
				t.Errorf("parseFloat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
