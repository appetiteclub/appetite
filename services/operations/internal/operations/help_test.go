package operations

import (
	"context"
	"strings"
	"testing"
)

func TestHandleHelp(t *testing.T) {
	parser := &DeterministicParser{}

	tests := []struct {
		name        string
		params      []string
		wantSuccess bool
		wantContent []string
	}{
		{
			name:        "noParams",
			params:      []string{},
			wantSuccess: true,
			wantContent: []string{
				"Command Reference Guide",
				"Authentication",
				"Order Management",
				"Table Management",
				"Menu Management",
				"Utility Commands",
			},
		},
		{
			name:        "withParams",
			params:      []string{"some", "params"},
			wantSuccess: true,
			wantContent: []string{"Command Reference Guide"},
		},
		{
			name:        "nilParams",
			params:      nil,
			wantSuccess: true,
			wantContent: []string{"help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := parser.handleHelp(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("handleHelp() error = %v", err)
			}
			if resp == nil {
				t.Fatal("handleHelp() returned nil response")
			}
			if resp.Success != tt.wantSuccess {
				t.Errorf("handleHelp() Success = %v, want %v", resp.Success, tt.wantSuccess)
			}
			for _, content := range tt.wantContent {
				if !strings.Contains(resp.HTML, content) {
					t.Errorf("handleHelp() HTML should contain %q", content)
				}
			}
		})
	}
}

func TestHandleHelpContainsLanguageSupport(t *testing.T) {
	parser := &DeterministicParser{}
	resp, err := parser.handleHelp(context.Background(), nil)
	if err != nil {
		t.Fatalf("handleHelp() error = %v", err)
	}

	languages := []string{"English", "Español", "Polski"}
	for _, lang := range languages {
		if !strings.Contains(resp.HTML, lang) {
			t.Errorf("handleHelp() should mention language support for %s", lang)
		}
	}
}

func TestHandleHelpContainsCommandExamples(t *testing.T) {
	parser := &DeterministicParser{}
	resp, err := parser.handleHelp(context.Background(), nil)
	if err != nil {
		t.Fatalf("handleHelp() error = %v", err)
	}

	// Check for example commands
	examples := []string{
		"login",
		"list orders",
		"list tables",
		"seat party",
		"open order",
		"close order",
		"add item",
	}
	for _, example := range examples {
		if !strings.Contains(resp.HTML, example) {
			t.Errorf("handleHelp() should contain example command %q", example)
		}
	}
}

func TestHandleHelpMessageField(t *testing.T) {
	parser := &DeterministicParser{}
	resp, err := parser.handleHelp(context.Background(), nil)
	if err != nil {
		t.Fatalf("handleHelp() error = %v", err)
	}

	if resp.Message != "Help displayed" {
		t.Errorf("handleHelp() Message = %q, want %q", resp.Message, "Help displayed")
	}
}
