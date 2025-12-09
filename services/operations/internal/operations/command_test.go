package operations

import (
	"context"
	"testing"
)

func TestNewDeterministicParser(t *testing.T) {
	parser := NewDeterministicParser(nil, nil, nil, nil)

	if parser == nil {
		t.Fatal("NewDeterministicParser() returned nil")
	}
	if parser.registry == nil {
		t.Error("registry is nil")
	}
}

func TestDeterministicParserFormatUnknownCommand(t *testing.T) {
	parser := NewDeterministicParser(nil, nil, nil, nil)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simpleCommand",
			input: "unknown",
		},
		{
			name:  "emptyCommand",
			input: "",
		},
		{
			name:  "commandWithSpaces",
			input: "some unknown command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.formatUnknownCommand(tt.input)
			if result == "" {
				t.Error("formatUnknownCommand() returned empty string")
			}
			// Should contain help instruction
			if len(result) < 20 {
				t.Errorf("formatUnknownCommand() result too short")
			}
		})
	}
}

func TestDeterministicParserFormatInvalidParams(t *testing.T) {
	parser := NewDeterministicParser(nil, nil, nil, nil)

	tests := []struct {
		name      string
		cmd       *CommandDefinition
		gotParams int
	}{
		{
			name: "fixedParams",
			cmd: &CommandDefinition{
				Canonical:   "test-cmd",
				MinParams:   2,
				MaxParams:   2,
				Description: "Test command",
			},
			gotParams: 1,
		},
		{
			name: "rangeParams",
			cmd: &CommandDefinition{
				Canonical:   "test-cmd",
				MinParams:   1,
				MaxParams:   3,
				Description: "Test command with range",
			},
			gotParams: 0,
		},
		{
			name: "zeroParams",
			cmd: &CommandDefinition{
				Canonical:   "test-cmd",
				MinParams:   0,
				MaxParams:   0,
				Description: "No params command",
			},
			gotParams: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.formatInvalidParams(tt.cmd, tt.gotParams)
			if result == "" {
				t.Error("formatInvalidParams() returned empty string")
			}
		})
	}
}

func TestDeterministicParserProcessUnknownCommand(t *testing.T) {
	parser := NewDeterministicParser(nil, nil, nil, nil)

	result, err := parser.Process(context.Background(), "completely-unknown-xyz")
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result == nil {
		t.Fatal("Process() returned nil result")
	}
	if result.Success {
		t.Error("Process() should return Success=false for unknown command")
	}
	if result.Message != "Command not recognized" {
		t.Errorf("Message = %q, want %q", result.Message, "Command not recognized")
	}
}

func TestDeterministicParserProcessEmptyInput(t *testing.T) {
	parser := NewDeterministicParser(nil, nil, nil, nil)

	result, err := parser.Process(context.Background(), "")
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result == nil {
		t.Fatal("Process() returned nil result")
	}
	if result.Success {
		t.Error("Process() should return Success=false for empty input")
	}
}

func TestDeterministicParserProcessInvalidParamCount(t *testing.T) {
	parser := NewDeterministicParser(nil, nil, nil, nil)

	// "get order" requires 1 param, providing 0
	result, err := parser.Process(context.Background(), "get order")
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result == nil {
		t.Fatal("Process() returned nil result")
	}
	if result.Success {
		t.Error("Process() should return Success=false for invalid param count")
	}
	if result.Message != "Invalid parameter count" {
		t.Errorf("Message = %q, want %q", result.Message, "Invalid parameter count")
	}
}

func TestDeterministicParserProcessHelp(t *testing.T) {
	parser := NewDeterministicParser(nil, nil, nil, nil)

	result, err := parser.Process(context.Background(), "help")
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result == nil {
		t.Fatal("Process() returned nil result")
	}
	if !result.Success {
		t.Error("Process('help') should return Success=true")
	}
	if result.HTML == "" {
		t.Error("Process('help') should return non-empty HTML")
	}
}

func TestCommandResponseFields(t *testing.T) {
	resp := &CommandResponse{
		HTML:    "<p>Test</p>",
		Success: true,
		Message: "Test message",
	}

	if resp.HTML != "<p>Test</p>" {
		t.Errorf("HTML = %q, want %q", resp.HTML, "<p>Test</p>")
	}
	if !resp.Success {
		t.Error("Success should be true")
	}
	if resp.Message != "Test message" {
		t.Errorf("Message = %q, want %q", resp.Message, "Test message")
	}
}
