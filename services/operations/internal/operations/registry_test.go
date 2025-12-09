package operations

import (
	"testing"
)

func TestNormalizeInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase",
			input: "LIST ORDERS",
			want:  "list orders",
		},
		{
			name:  "trimSpaces",
			input: "  list orders  ",
			want:  "list orders",
		},
		{
			name:  "multipleSpaces",
			input: "list    orders",
			want:  "list orders",
		},
		{
			name:  "hyphenToSpace",
			input: "list-orders",
			want:  "list orders",
		},
		{
			name:  "mixedCase",
			input: "LiSt OrDeRs",
			want:  "list orders",
		},
		{
			name:  "emptyString",
			input: "",
			want:  "",
		},
		{
			name:  "onlySpaces",
			input: "   ",
			want:  "",
		},
		{
			name:  "combined",
			input: "  GET-ORDER   123  ",
			want:  "get order 123",
		},
		{
			name:  "multipleHyphens",
			input: "add-item-to-group",
			want:  "add item to group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeInput(tt.input)
			if got != tt.want {
				t.Errorf("normalizeInput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "singleWord",
			input: "help",
			want:  []string{"help"},
		},
		{
			name:  "multipleWords",
			input: "list orders",
			want:  []string{"list", "orders"},
		},
		{
			name:  "withParams",
			input: "get order 123",
			want:  []string{"get", "order", "123"},
		},
		{
			name:  "emptyString",
			input: "",
			want:  []string{},
		},
		{
			name:  "onlySpaces",
			input: "   ",
			want:  []string{},
		},
		{
			name:  "multipleSpaces",
			input: "list   orders   123",
			want:  []string{"list", "orders", "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("tokenize(%q) length = %d, want %d", tt.input, len(got), len(tt.want))
				return
			}
			for i, g := range got {
				if g != tt.want[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, g, tt.want[i])
				}
			}
		})
	}
}

func TestMatchesVariation(t *testing.T) {
	tests := []struct {
		name            string
		inputTokens     []string
		variationTokens []string
		want            bool
	}{
		{
			name:            "exactMatch",
			inputTokens:     []string{"list", "orders"},
			variationTokens: []string{"list", "orders"},
			want:            true,
		},
		{
			name:            "inputHasMore",
			inputTokens:     []string{"list", "orders", "123"},
			variationTokens: []string{"list", "orders"},
			want:            true,
		},
		{
			name:            "variationHasMore",
			inputTokens:     []string{"list"},
			variationTokens: []string{"list", "orders"},
			want:            false,
		},
		{
			name:            "noMatch",
			inputTokens:     []string{"get", "order"},
			variationTokens: []string{"list", "orders"},
			want:            false,
		},
		{
			name:            "emptyInput",
			inputTokens:     []string{},
			variationTokens: []string{"list"},
			want:            false,
		},
		{
			name:            "emptyVariation",
			inputTokens:     []string{"list"},
			variationTokens: []string{},
			want:            true,
		},
		{
			name:            "bothEmpty",
			inputTokens:     []string{},
			variationTokens: []string{},
			want:            true,
		},
		{
			name:            "partialMismatch",
			inputTokens:     []string{"list", "tables"},
			variationTokens: []string{"list", "orders"},
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesVariation(tt.inputTokens, tt.variationTokens)
			if got != tt.want {
				t.Errorf("matchesVariation(%v, %v) = %v, want %v",
					tt.inputTokens, tt.variationTokens, got, tt.want)
			}
		})
	}
}

func TestNewCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry(nil)

	if registry == nil {
		t.Fatal("NewCommandRegistry() returned nil")
	}
	if registry.commands == nil {
		t.Error("commands map is nil")
	}
	if len(registry.commands) == 0 {
		t.Error("no commands registered")
	}

	// Verify some expected commands exist
	expectedCommands := []string{"help", "login", "exit", "list-orders", "list-tables"}
	for _, cmd := range expectedCommands {
		if _, ok := registry.commands[cmd]; !ok {
			t.Errorf("expected command %q not found", cmd)
		}
	}
}

func TestCommandRegistryFindCommand(t *testing.T) {
	registry := NewCommandRegistry(nil)

	tests := []struct {
		name          string
		input         string
		wantFound     bool
		wantCanonical string
		wantParams    []string
	}{
		{
			name:          "exactCanonical",
			input:         "help",
			wantFound:     true,
			wantCanonical: "help",
			wantParams:    []string{},
		},
		{
			name:          "shortForm",
			input:         "h",
			wantFound:     true,
			wantCanonical: "help",
			wantParams:    []string{},
		},
		{
			name:          "variation",
			input:         "ayuda",
			wantFound:     true,
			wantCanonical: "help",
			wantParams:    []string{},
		},
		{
			name:          "twoWordCommand",
			input:         "list orders",
			wantFound:     true,
			wantCanonical: "list-orders",
			wantParams:    []string{},
		},
		{
			name:          "commandWithParams",
			input:         "get order 123",
			wantFound:     true,
			wantCanonical: "get-order",
			wantParams:    []string{"123"},
		},
		{
			name:          "loginDotNotation",
			input:         ".1234",
			wantFound:     true,
			wantCanonical: "login",
			wantParams:    []string{"1234"},
		},
		{
			name:          "logoutDotNotation",
			input:         ".",
			wantFound:     true,
			wantCanonical: "exit",
			wantParams:    []string{},
		},
		{
			name:          "caseInsensitive",
			input:         "HELP",
			wantFound:     true,
			wantCanonical: "help",
			wantParams:    []string{},
		},
		{
			name:          "hyphenatedInput",
			input:         "list-orders",
			wantFound:     true,
			wantCanonical: "list-orders",
			wantParams:    []string{},
		},
		{
			name:          "unknownCommand",
			input:         "unknown-command",
			wantFound:     false,
			wantCanonical: "",
			wantParams:    nil,
		},
		{
			name:          "emptyInput",
			input:         "",
			wantFound:     false,
			wantCanonical: "",
			wantParams:    nil,
		},
		{
			name:          "onlySpaces",
			input:         "   ",
			wantFound:     false,
			wantCanonical: "",
			wantParams:    nil,
		},
		{
			name:          "listTablesShort",
			input:         "lt",
			wantFound:     true,
			wantCanonical: "list-tables",
			wantParams:    []string{},
		},
		{
			name:          "spanishVariation",
			input:         "mesas",
			wantFound:     true,
			wantCanonical: "list-tables",
			wantParams:    []string{},
		},
		{
			name:          "polishVariation",
			input:         "stoliki",
			wantFound:     true,
			wantCanonical: "list-tables",
			wantParams:    []string{},
		},
		{
			name:          "questionMark",
			input:         "?",
			wantFound:     true,
			wantCanonical: "help",
			wantParams:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, params, found := registry.FindCommand(tt.input)
			if found != tt.wantFound {
				t.Errorf("FindCommand(%q) found = %v, want %v", tt.input, found, tt.wantFound)
				return
			}
			if !found {
				return
			}
			if cmd.Canonical != tt.wantCanonical {
				t.Errorf("FindCommand(%q) canonical = %q, want %q", tt.input, cmd.Canonical, tt.wantCanonical)
			}
			if len(params) != len(tt.wantParams) {
				t.Errorf("FindCommand(%q) params = %v, want %v", tt.input, params, tt.wantParams)
				return
			}
			for i, p := range params {
				if p != tt.wantParams[i] {
					t.Errorf("FindCommand(%q) params[%d] = %q, want %q", tt.input, i, p, tt.wantParams[i])
				}
			}
		})
	}
}

func TestCommandRegistryFindCommandMultipleParams(t *testing.T) {
	registry := NewCommandRegistry(nil)

	// Note: normalizeInput converts hyphens to spaces, so IDs with hyphens get tokenized
	tests := []struct {
		name       string
		input      string
		wantParams []string
	}{
		{
			name:       "addItemThreeParams",
			input:      "add item order123 pizza 2",
			wantParams: []string{"order123", "pizza", "2"},
		},
		{
			name:       "seatPartyTwoParams",
			input:      "seat party table1 4",
			wantParams: []string{"table1", "4"},
		},
		{
			name:       "transferOrderTwoParams",
			input:      "transfer order order123 table5",
			wantParams: []string{"order123", "table5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, params, found := registry.FindCommand(tt.input)
			if !found {
				t.Fatalf("FindCommand(%q) not found", tt.input)
			}
			if len(params) != len(tt.wantParams) {
				t.Errorf("FindCommand(%q) params = %v, want %v", tt.input, params, tt.wantParams)
				return
			}
			for i, p := range params {
				if p != tt.wantParams[i] {
					t.Errorf("FindCommand(%q) params[%d] = %q, want %q", tt.input, i, p, tt.wantParams[i])
				}
			}
		})
	}
}

func TestCommandRegistryRegister(t *testing.T) {
	registry := NewCommandRegistry(nil)
	initialCount := len(registry.commands)

	// Register a new command
	registry.register("test-cmd", &CommandDefinition{
		Canonical:   "test-cmd",
		Variations:  []string{"test"},
		Description: "Test command",
	})

	if len(registry.commands) != initialCount+1 {
		t.Errorf("commands count = %d, want %d", len(registry.commands), initialCount+1)
	}

	if _, ok := registry.commands["test-cmd"]; !ok {
		t.Error("registered command not found")
	}
}

func TestCommandDefinitionFields(t *testing.T) {
	registry := NewCommandRegistry(nil)

	// Test help command definition
	helpCmd := registry.commands["help"]
	if helpCmd == nil {
		t.Fatal("help command not found")
	}

	if helpCmd.Canonical != "help" {
		t.Errorf("Canonical = %q, want %q", helpCmd.Canonical, "help")
	}
	if helpCmd.Description == "" {
		t.Error("Description is empty")
	}
	if len(helpCmd.Variations) == 0 {
		t.Error("Variations is empty")
	}
	if len(helpCmd.ShortForms) == 0 {
		t.Error("ShortForms is empty")
	}

	// Test add-item command (has params)
	addItemCmd := registry.commands["add-item"]
	if addItemCmd == nil {
		t.Fatal("add-item command not found")
	}
	if addItemCmd.MinParams != 3 {
		t.Errorf("MinParams = %d, want 3", addItemCmd.MinParams)
	}
	if addItemCmd.MaxParams != 3 {
		t.Errorf("MaxParams = %d, want 3", addItemCmd.MaxParams)
	}
}

func TestFindCommandAllShortForms(t *testing.T) {
	registry := NewCommandRegistry(nil)

	// Note: "sp" is ambiguous (seat-party and set-price both use it)
	// Testing only unambiguous short forms
	shortForms := map[string]string{
		"h":   "help",
		"lo":  "list-orders",
		"lao": "list-active-orders",
		"go":  "get-order",
		"gi":  "get-order-items",
		"gs":  "get-order-status",
		"oo":  "open-order",
		"co":  "close-order",
		"xo":  "cancel-order",
		"ai":  "add-item",
		"ri":  "remove-item",
		"ui":  "update-item",
		"sk":  "send-to-kitchen",
		"mr":  "mark-ready",
		"lt":  "list-tables",
		"lat": "list-available-tables",
		"lot": "list-occupied-tables",
		"gt":  "get-table",
		"rt":  "release-table",
		"rv":  "reserve-table",
		"cr":  "cancel-reservation",
	}

	for short, canonical := range shortForms {
		t.Run(short, func(t *testing.T) {
			cmd, _, found := registry.FindCommand(short)
			if !found {
				t.Fatalf("FindCommand(%q) not found", short)
			}
			if cmd.Canonical != canonical {
				t.Errorf("FindCommand(%q) = %q, want %q", short, cmd.Canonical, canonical)
			}
		})
	}
}
