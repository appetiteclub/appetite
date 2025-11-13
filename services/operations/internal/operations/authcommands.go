package operations

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (p *DeterministicParser) handleLogin(ctx context.Context, params []string) (*CommandResponse, error) {
	if len(params) == 0 {
		return &CommandResponse{
			HTML: `
				<div style="padding: 1rem; background: #eff6ff; border-radius: 0.5rem; border-left: 4px solid #3b82f6;">
					<p style="margin: 0;"><strong>🔐 PIN Required</strong></p>
					<p style="margin: 0.5rem 0 0 0; font-size: 0.9em;">Enter your PIN to authenticate.</p>
					<p style="margin: 0.5rem 0 0 0; font-size: 0.85em; color: #666;"><em>Example: login abc123 or .abc123</em></p>
				</div>
			`,
			Success: false,
			Message: "PIN required",
		}, nil
	}

	pin := params[0]

	if p.handler == nil || p.handler.authnClient == nil {
		return &CommandResponse{
			HTML:    formatError("Authentication service unavailable"),
			Success: false,
			Message: "Service unavailable",
		}, nil
	}

	type pinLoginRequest struct {
		PIN string `json:"pin"`
	}

	reqBody := pinLoginRequest{PIN: pin}

	resp, err := p.handler.authnClient.Request(ctx, "POST", "/authn/pin-login", reqBody)
	if err != nil {
		return &CommandResponse{
			HTML:    formatError("Authentication failed. Please check your PIN."),
			Success: false,
			Message: "Authentication failed",
		}, nil
	}

	if resp == nil || resp.Data == nil {
		return &CommandResponse{
			HTML:    formatError("Invalid PIN. Please try again."),
			Success: false,
			Message: "Invalid PIN",
		}, nil
	}

	// Extract user_id from response data
	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		return &CommandResponse{
			HTML:    formatError("Invalid response format"),
			Success: false,
			Message: "Invalid response",
		}, nil
	}

	userIDStr, ok := dataMap["user_id"].(string)
	if !ok {
		return &CommandResponse{
			HTML:    formatError("Invalid user ID in response"),
			Success: false,
			Message: "Invalid response",
		}, nil
	}
	userID, err := parseUUID(userIDStr)
	if err != nil {
		return &CommandResponse{
			HTML:    formatError("Invalid user ID received"),
			Success: false,
			Message: "Invalid response",
		}, nil
	}

	token, err := p.handler.tokenStore.Create(userID)
	if err != nil {
		return &CommandResponse{
			HTML:    formatError("Failed to create session"),
			Success: false,
			Message: "Session creation failed",
		}, nil
	}

	// Log successful login
	if p.handler.auditLogger != nil {
		p.handler.auditLogger.LogLogin(ctx, userID)
	}

	return &CommandResponse{
		HTML: fmt.Sprintf(`
			<div style="padding: 1rem; background: #f0fdf4; border-radius: 0.5rem; border-left: 4px solid #10b981;">
				<p style="margin: 0;"><strong>✓ Authenticated</strong></p>
				<p style="margin: 0.5rem 0 0 0; font-size: 0.9em;">You are now logged in.</p>
			</div>
			<script>
				sessionStorage.setItem('ops_token', '%s');
				sessionStorage.setItem('ops_user_id', '%s');
			</script>
		`, token, userIDStr),
		Success: true,
		Message: "Login successful",
	}, nil
}

func (p *DeterministicParser) handleLogout(ctx context.Context, params []string) (*CommandResponse, error) {
	if p.handler == nil || p.handler.tokenStore == nil {
		return &CommandResponse{
			HTML:    formatError("Session service unavailable"),
			Success: false,
			Message: "Service unavailable",
		}, nil
	}

	// Extract token and userID from context
	token := getTokenFromContext(ctx)
	userID := getUserIDFromContext(ctx)

	if token != "" {
		p.handler.tokenStore.Invalidate(token)
	}

	// Log logout
	if p.handler.auditLogger != nil && userID != uuid.Nil {
		p.handler.auditLogger.LogLogout(ctx, userID)
	}

	return &CommandResponse{
		HTML: `
			<div style="padding: 1rem; background: #fef3c7; border-radius: 0.5rem; border-left: 4px solid #f59e0b;">
				<p style="margin: 0;"><strong>👋 Logged Out</strong></p>
				<p style="margin: 0.5rem 0 0 0; font-size: 0.9em;">You have been logged out successfully.</p>
			</div>
			<div style="padding: 1rem; background: #eff6ff; border-radius: 0.5rem; border-left: 4px solid #3b82f6; margin-top: 1rem;">
				<p style="margin: 0;"><strong>🔐 Enter PIN to Login</strong></p>
				<p style="margin: 0.5rem 0 0 0; font-size: 0.9em;">Type your PIN to authenticate.</p>
				<p style="margin: 0.5rem 0 0 0; font-size: 0.85em; color: #666;"><em>Example: .abc123</em></p>
			</div>
			<script>
				sessionStorage.removeItem('ops_token');
				sessionStorage.removeItem('ops_user_id');
			</script>
		`,
		Success: true,
		Message: "Logout successful",
	}, nil
}

func formatError(message string) string {
	return fmt.Sprintf(`
		<div style="padding: 1rem; background: #fef2f2; border-radius: 0.5rem; border-left: 4px solid #ef4444;">
			<p style="margin: 0;"><strong>⚠️ Error</strong></p>
			<p style="margin: 0.5rem 0 0 0; font-size: 0.9em;">%s</p>
		</div>
	`, message)
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
