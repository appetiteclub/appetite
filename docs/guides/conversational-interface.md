# Conversational Interface System

**Type:** Guide
**Date:** 2025-11-12
**Version:** 1.0
**Status:** Active

---

## Purpose

This guide explains the conversational interface system in the Operations service. The system provides a terminal-style chat interface where operators interact with the restaurant management system using natural language commands.

---

## System Overview

The conversational interface is an **experimental first approach** for system interaction. Operators interact through an **agent-based conversational system** using designated internal terminals. The system will eventually operate alongside a traditional web interface (planned but not yet implemented) and is designed to support voice integration for simplified operations.

**Current Implementation:** Deterministic command parser (Phase 1)
**Future Evolution:** LLM-powered natural language processing with tutor capabilities

**Key Characteristics:**
- Agent-mediated interactions (not direct system access)
- Lightweight PIN authentication for internal terminals only
- Session-based operation with terminal sharing capability
- Mandatory authentication for destructive/responsible operations

---

## Core Concepts

### 1. Deterministic Parser

The current system uses pattern matching and command registry to interpret user input.

**Characteristics:**
- Predictable behavior
- Fast response times
- No ambiguity in command interpretation
- Multi-language support (currently implementing: English, Spanish, Polish)
- Extensible to additional languages as needed
- Handles variations and short forms

**Example Flow:**
```
User types: "open order 5"
         ↓
System normalizes: "open order 5"
         ↓
Matches canonical: "open-order"
         ↓
Extracts parameter: table_id=5
         ↓
Executes: handleOpenOrder(5)
         ↓
Backend generates conversational ID: 47 (atomic counter)
         ↓
Returns: "Order 47 created for Table 5"
```

### 2. Command Forms

Each command supports three input forms:

**Natural Language** (primary display):
```
open order 5        → system returns: 47
add item 47 BURGER 2
send to kitchen 47
```

**Short Form** (for speed):
```
oo 5                → returns: 47
ai 47 BURGER 2
sk 47
```

**Canonical** (internal representation):
```
open-order
add-item
send-to-kitchen
```

### 3. Multi-Language Support

Commands work in multiple languages without translation layers. The system architecture supports language extensibility.

**Currently Implemented Languages:**

**English:**
```
list tables
open order 5
seat party 3 4
```

**Spanish:**
```
listar mesas
abrir orden 5
sentar 3 4
```

**Polish:**
```
lista stolików
otwórz zamówienie 5
posadź gości 3 4
```

**Future Languages:**
The command registry architecture allows adding new languages by registering command variations. Additional languages can be implemented as operational needs require.

---

## Authentication and Scope

### PIN-Based Authentication

The system uses a lightweight PIN-based authentication designed for **internal designated terminals only**. Each user has a unique 6-character alphanumeric PIN (case-insensitive, typically entered in lowercase) that determines their access scope.

**PIN Format:**
- 6 alphanumeric characters
- Case-insensitive (internally normalized)
- Natural entry in lowercase (e.g., `abc123`, `w4t5r1`)

**Login Process:**

Users can authenticate using:
```
login          → System prompts for PIN
login abc123   → Direct login with PIN
abc123         → Eventually, PIN alone (planned)
```

**Session Lifecycle:**
```
1. User: login abc123 → Agent validates → Session created
2. All commands execute within user's scope
3. Destructive/responsible operations require authentication
4. User: exit → Session closed, terminal available for next user
```

**Terminal Sharing:**
- Terminals are shared resources in the operation
- `exit` command releases terminal for next operator
- No user-specific terminal assignment
- Quick session switching between operators

**Scope Levels:**
- **Waiter:** Access to assigned tables and orders
- **Manager:** Access to all tables, plus management commands
- **Admin:** Full system access including configuration

**Authentication Requirements:**

Operations requiring PIN authentication:
- Destructive operations (delete, cancel)
- Financially responsible operations (discounts, void)
- Status changes with business impact
- Most operations for accountability

### Command-Level PIN Override (Planned)

**Note:** This feature is not yet implemented.

Future capability will allow quick command execution for another user without changing session:

```
close order 47 PIN:mgr001         → Execute as user with PIN mgr001
apply discount 47 10% PIN:mgr001  → Manager approval inline
```

**Planned Use Cases:**
- Manager approval without session switch
- Supervisor override during busy service
- Cross-scope operations without logout/login

**Planned Security:**
- Both executing user and authorizing PIN logged
- Explicit permission configuration required
- Failed override attempts trigger alerts

---

## Interface Design

### Terminal-Style Chat

The interface mimics a terminal for familiarity with technical operators. Interactions occur through an **agent** that processes commands and communicates with backend services.

**Visual Characteristics:**
- Monospace font
- Command history scrollback
- User input at bottom
- Agent responses above
- No bubble-style messaging

**Interaction Pattern:**
```
Agent:  Ready. Type 'help' for command reference or 'login' to authenticate.
User:   login maria1
Agent:  ✓ Session started. Welcome, Maria (Waiter - Tables: 3, 5, 7)
User:   list tables
Agent:  ✓ 12 tables found
        Table 3 - Occupied (2 guests) - Order 47
        Table 5 - Available
        Table 7 - Reserved (Smith, 18:00)
        ...
User:   open order 3
Agent:  ✓ Order 47 created for Table 3
        Use 'add item 47 [item] [qty]' to add items
User:   exit
Agent:  ✓ Session closed. Terminal ready for next user.
```

### Response Format

Agent responses follow consistent patterns:

**Success:**
```
✓ [Operation] successful
  [Key details]
  [Next suggested action]
```

**Error:**
```
✗ [Operation] failed
  [Error reason]
  [Corrective suggestion]
```

**Information:**
```
[Icon] [Title]
  [Structured data]
  [Contextual help]
```

---

## Command Categories

### Order Management

**Query Commands:**
- `list orders` - Show all orders
- `active orders` - Show active orders only
- `get order 47` - Show order details
- `order items 47` - Show order items
- `order status 47` - Show order status

**Action Commands:**
- `open order 5` - Create order for table 5 (returns conversational ID, e.g., 47)
- `add item 47 BURGER 2` - Add items to order
- `send to kitchen 47` - Submit order to kitchen
- `mark ready 47` - Mark order ready for serving
- `close order 47` - Complete and close order

**Advanced Commands:**
- `split order 47 by-person` - Split order for separate bills
- `merge orders 47 52` - Combine multiple orders
- `apply discount 47 10%` - Apply discount to order
- `transfer order 47 8` - Move order to different table

### Table Management

**Query Commands:**
- `list tables` - Show all tables
- `available tables` - Show available tables only
- `get table 5` - Show table details
- `table status 5` - Show table status

**Seating Commands:**
- `seat party 3 4` - Seat 4 guests at table 3
- `release table 3` - Clear and free table 3
- `reserve table 5 "Smith"` - Create reservation

**Management Commands:**
- `assign waiter 5 USR-123` - Assign waiter to table
- `clean table 5` - Mark table as clean
- `dirty table 5` - Mark table needs cleaning
- `merge tables 3 4` - Combine tables for large party

---

## Typical Workflows

### Session Start

```
1. login maria1  → Authenticate with PIN
   Agent shows: Welcome, Maria (Waiter - Tables: 3, 5, 7)
2. Commands now execute within Maria's scope
```

### Opening a Table

```
1. sp 3 4        → Seat 4 guests at table 3
2. oo 3          → Open order (system returns: 47)
3. ai 47 BURGER 2 → Add 2 burgers
4. ai 47 COFFEE 4 → Add 4 coffees
5. sk 47         → Send to kitchen
```

### Checking Order Status

```
1. lao           → List active orders
   System shows: 47, 52, 58
2. gs 47         → Get status of order 47
   System shows: In Kitchen, 5 minutes
3. mr 47         → Mark ready when done
```

### Closing Out

```
1. co 47         → Close order (payment processed)
2. rt 3          → Release table 3
3. mtd 3         → Mark table dirty for cleaning
4. exit          → End session, terminal available for next user
```

### Handling Special Cases

```
# Split bill by person
so 47 by-person

# Apply manager discount (requires manager PIN)
ad 47 10% PIN:mgr001          (planned - not yet implemented)

# Transfer to larger table
to 47 8

# Merge small tables
mt 3 4
```

### Terminal Handoff

```
# Operator Maria finishing shift
1. exit          → Close Maria's session

# Operator Juan starting
2. login juan01  → Juan's session begins
3. lao           → Check active orders
```

---

## Future Evolution

### Phase 2: LLM Integration

The system will incorporate LLM capabilities while maintaining deterministic fallback:

**Natural Language Understanding:**
```
User: "Table 5 wants to move to a bigger table"
LLM:  Understands intent → Maps to: transfer-order ORD-5X [table]
      Asks: "Which table should I transfer to?"
```

**Context Awareness:**
```
User: "Add 2 more burgers"
LLM:  Knows active order → Maps to: add-item ORD-5X BURGER 2
```

**Tutor Mode:**
```
User: "How do I split a bill?"
LLM:  Explains split-order command
      Shows examples
      Offers to execute with confirmation
```

**Multi-Step Operations:**
```
User: "Set up table 3 for 4 people and start their order"
LLM:  Executes: seat-party 3 4
      Then: open-order 3
      Confirms: "Table 3 ready for ordering (ORD-3X)"
```

### Voice Integration

Voice commands will map to the same command system:

**Voice Input Processing:**
```
Voice: "Open order for table five"
  ↓ STT
Text: "open order for table five"
  ↓ Parser/LLM
Command: open-order 5
  ↓ Execute
Response: "Order opened for table five"
  ↓ TTS
Voice: [Speaks confirmation]
```

**Ambient Noise Handling:**
- Push-to-talk for busy environments
- Command confirmation for critical operations
- Visual display always shows text interpretation

### Traditional Web Interface (Planned)

The command system architecture is designed to power both chat and web interfaces:

**Chat Interface (Current):**
```
Input: open order 5
Process: Direct text command
```

**Web Interface (Planned):**
```
UI: [Button: Open Order] [Input: Table 5]
Process: POST /commands/open-order {table_id: 5}
Backend: Same handler as chat command
```

**Design Goals:**
Both interfaces will:
- Share authentication/authorization
- Use identical command handlers
- Provide consistent behavior
- Support same business logic

The chat interface serves as the foundation, with traditional web UI to be implemented later using the same command processing backend.

---

## Technical Architecture

### Request Flow

```
1. User Input (via designated terminal)
   ↓
2. WebSocket/HTTP to /chat endpoint
   ↓
3. Agent receives command
   ↓
4. Session validation (PIN-based)
   ↓
5. Command normalization
   ↓
6. Pattern matching → Command identification
   ↓
7. Parameter extraction and validation
   ↓
8. Authorization check (scope + permissions)
   ↓
9. Handler execution
   ↓
10. Service layer operation
    ↓
11. Response formatting (HTML)
    ↓
12. Agent response via HTMX update
```

### Components

**Command Processor** (`command.go`):
- Interface definition
- Generic processing logic
- Error handling structure

**Command Registry** (`registry.go`):
- Command definitions
- Variation mappings
- Pattern specifications

**Command Handlers** (`orders.go`, `tables.go`):
- Business logic implementation
- Service integration
- Response generation

**Authentication** (`auth.go`):
- PIN validation
- Session management
- Scope determination

**Chat Endpoint** (`chat.go`):
- HTTP handler
- Session binding
- Response rendering

### Data Flow

**Session State:**
```go
type Session struct {
    UserID    string
    PIN       string
    Scope     string    // waiter, manager, admin
    TableIDs  []int     // For waiter scope
    ExpiresAt time.Time
}
```

**Command Context:**
```go
type CommandContext struct {
    UserID      string
    Scope       string
    TableAccess []int
    Permissions []string
}
```

**Command Response:**
```go
type CommandResponse struct {
    HTML    string  // Formatted UI content
    Success bool
    Message string
    Error   error
}
```

---

## Implementation Guidelines

### Adding New Commands

1. **Define in registry** (`registry.go`):
```go
"new-command": {
    Canonical:  "new-command",
    Variations: []string{"new command", "nc"},
    Pattern:    regexp.MustCompile(`^(new|nc)[\s-]*(command)?\s+(.+)$`),
    Handler:    p.handleNewCommand,
    MinParams:  1,
    MaxParams:  2,
}
```

2. **Implement handler**:
```go
func (p *Parser) handleNewCommand(ctx context.Context, params []string) (*CommandResponse, error) {
    // Validate params
    // Call service
    // Format response
    return &CommandResponse{
        HTML:    formatHTML(...),
        Success: true,
        Message: "Command executed",
    }, nil
}
```

3. **Add to help** (`help.go`):
```html
<tr>
    <td><code>new command</code></td>
    <td><code>nc</code></td>
    <td>nc param | new command param</td>
</tr>
```

### Response Formatting

Use consistent HTML structure:

```go
html := fmt.Sprintf(`
    <p>✓ <strong>%s</strong></p>
    <ul>
        <li><strong>Field:</strong> %s</li>
    </ul>
    <p><em>%s</em></p>
`, title, value, nextAction)
```

### Error Handling

Provide actionable error messages:

```go
if err != nil {
    return &CommandResponse{
        HTML: fmt.Sprintf(`
            <p>✗ <strong>Operation Failed</strong></p>
            <p>%s</p>
            <p><em>Try: %s</em></p>
        `, err.Error(), suggestion),
        Success: false,
        Error:   err,
    }, nil
}
```

---

## Operational Benefits

### Speed

**Short forms** reduce typing during busy service:
- `oo 5` vs `open order 5` (60% faster)
- `ai ORD-5X BURGER 2` vs full form (40% faster)
- Command history with arrow keys

### Flexibility

**Multiple input methods** for different preferences:
- Natural language for trainees
- Short forms for experienced staff
- Native language for international teams

### Auditability

**Complete command logging:**
- Every command timestamped
- User identification via PIN
- Parameter capture
- Success/failure tracking

### Training

**Built-in help system:**
- `help` shows full command reference
- Suggestions after each command
- Error messages include corrective hints
- Multi-language examples

---

## Security Considerations

### Designated Terminal Access

- System accessible only from **designated internal terminals**
- Physical access control to terminal locations
- No remote access to conversational interface
- Terminals identified and registered in system

### Input Validation

- All parameters sanitized before service calls
- Command injection prevention
- SQL injection prevention (parameterized queries)
- XSS prevention (HTML escaping)

### Authorization

- PIN authentication required for session start
- Destructive/responsible operations require authentication
- Every command checks user scope
- Table-level access control for waiters
- Audit trail for all operations

### Session Management

- 8-hour session timeout
- Explicit logout via `exit` command
- Session invalidation on security events
- Terminal remains available for next user after `exit`
- Sessions tied to terminal, not user device

### Rate Limiting

- Max 60 commands per minute per session
- Throttling on authentication attempts (PIN)
- Protection against command spam
- Failed login attempt monitoring

---

## Monitoring and Metrics

### Key Metrics

**Usage:**
- Commands per minute
- Most used commands
- Average response time
- Error rate by command

**User Behavior:**
- Short form vs natural language ratio
- Command success rate
- Session duration
- Commands per session

**Performance:**
- Command processing time
- Service call latency
- Pattern matching efficiency
- Cache hit rates

### Logging

**Command Execution:**
```json
{
  "timestamp": "2025-11-12T15:30:00Z",
  "user_id": "USR-123",
  "command": "open-order",
  "params": ["5"],
  "success": true,
  "duration_ms": 45
}
```

**Authentication Events:**
```json
{
  "timestamp": "2025-11-12T15:00:00Z",
  "event": "pin_login",
  "user_id": "USR-123",
  "scope": "waiter",
  "table_access": [3, 5, 7]
}
```

---

## Related Documentation

- **Architecture:** `docs/drafts/command-parser-architecture.md`
- **API Endpoints:** `docs/api/operations-service.md` (when created)
- **Command Reference:** Built-in `help` command
- **Authorization:** `docs/draft/001-authz-system-overview.md`

---

**Last Updated:** 2025-11-12
**Maintained By:** Operations Team
**Review Schedule:** After each phase completion
