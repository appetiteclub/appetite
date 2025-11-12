package operations

import (
	"context"
	"fmt"
)

// CommandProcessor defines the interface for processing user commands
type CommandProcessor interface {
	Process(ctx context.Context, input string) (*CommandResponse, error)
}

// CommandResponse represents the structured response from command processing
type CommandResponse struct {
	HTML    string
	Success bool
	Message string
}

// DeterministicParser implements CommandProcessor using pattern matching
type DeterministicParser struct {
	tableClient *ServiceClientWrapper
	orderClient *ServiceClientWrapper
	registry    *CommandRegistry
}

type ServiceClientWrapper struct {
	baseURL string
}

// NewDeterministicParser creates a new deterministic command parser
func NewDeterministicParser(tableClient, orderClient *ServiceClientWrapper) *DeterministicParser {
	parser := &DeterministicParser{
		tableClient: tableClient,
		orderClient: orderClient,
	}
	parser.registry = NewCommandRegistry(parser)
	return parser
}

// Process implements CommandProcessor interface
func (p *DeterministicParser) Process(ctx context.Context, input string) (*CommandResponse, error) {
	// Find matching command
	cmd, params, found := p.registry.FindCommand(input)
	if !found {
		return &CommandResponse{
			HTML:    p.formatUnknownCommand(input),
			Success: false,
			Message: "Command not recognized",
		}, nil
	}

	// Validate parameter count
	if len(params) < cmd.MinParams || len(params) > cmd.MaxParams {
		return &CommandResponse{
			HTML:    p.formatInvalidParams(cmd, len(params)),
			Success: false,
			Message: "Invalid parameter count",
		}, nil
	}

	// Execute command handler
	return cmd.Handler(ctx, params)
}

func (p *DeterministicParser) formatUnknownCommand(input string) string {
	return fmt.Sprintf(`
		<p>⚠️ Command not recognized: <code>%s</code></p>
		<p>Type <code>help</code> to see available commands.</p>
	`, input)
}

func (p *DeterministicParser) formatInvalidParams(cmd *CommandDefinition, got int) string {
	expected := fmt.Sprintf("%d", cmd.MinParams)
	if cmd.MaxParams != cmd.MinParams {
		expected = fmt.Sprintf("%d-%d", cmd.MinParams, cmd.MaxParams)
	}
	return fmt.Sprintf(`
		<p>⚠️ Invalid parameters for <code>%s</code></p>
		<p><strong>Expected:</strong> %s parameters</p>
		<p><strong>Got:</strong> %d parameters</p>
		<p><strong>Description:</strong> %s</p>
	`, cmd.Canonical, expected, got, cmd.Description)
}
