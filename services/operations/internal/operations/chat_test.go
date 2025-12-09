package operations

import (
	"testing"
)

func TestCommandRequiresAuth(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "loginCommand",
			message: "login abc123",
			want:    false,
		},
		{
			name:    "loginUppercase",
			message: "LOGIN abc123",
			want:    false,
		},
		{
			name:    "dotCommand",
			message: ".abc123",
			want:    false,
		},
		{
			name:    "exitCommand",
			message: "exit",
			want:    false,
		},
		{
			name:    "exitUppercase",
			message: "EXIT",
			want:    false,
		},
		{
			name:    "helpCommand",
			message: "help",
			want:    false,
		},
		{
			name:    "helpUppercase",
			message: "HELP",
			want:    false,
		},
		{
			name:    "listOrdersCommand",
			message: "list orders",
			want:    true,
		},
		{
			name:    "listTablesCommand",
			message: "list tables",
			want:    true,
		},
		{
			name:    "openOrderCommand",
			message: "open order 5",
			want:    true,
		},
		{
			name:    "seatPartyCommand",
			message: "seat party 3 4",
			want:    true,
		},
		{
			name:    "emptyMessage",
			message: "",
			want:    true,
		},
		{
			name:    "whitespaceMessage",
			message: "   ",
			want:    true,
		},
		{
			name:    "loginWithSpaces",
			message: "  login  abc123 ",
			want:    false,
		},
		{
			name:    "dotWithSpaces",
			message: "  .abc123 ",
			want:    false,
		},
		{
			name:    "helpWithMixedCase",
			message: "  HeLp  ",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.commandRequiresAuth(tt.message)
			if got != tt.want {
				t.Errorf("commandRequiresAuth(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}

func TestChatMessageRequestFields(t *testing.T) {
	req := ChatMessageRequest{
		Message: "help",
		Token:   "test-token",
	}

	if req.Message != "help" {
		t.Errorf("Message = %q, want %q", req.Message, "help")
	}
	if req.Token != "test-token" {
		t.Errorf("Token = %q, want %q", req.Token, "test-token")
	}
}

func TestChatMessageResponseFields(t *testing.T) {
	resp := ChatMessageResponse{
		HTML:    "<p>Test</p>",
		Success: true,
		Message: "Success",
	}

	if resp.HTML != "<p>Test</p>" {
		t.Errorf("HTML = %q, want %q", resp.HTML, "<p>Test</p>")
	}
	if !resp.Success {
		t.Error("Success should be true")
	}
	if resp.Message != "Success" {
		t.Errorf("Message = %q, want %q", resp.Message, "Success")
	}
}
