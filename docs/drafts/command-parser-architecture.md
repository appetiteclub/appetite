# Command Parser Architecture

**Status:** Draft
**Date:** 2025-11-12
**Scope:** Operations Service - Conversational Interface

---

## Overview

The command parser bridges natural language user input with structured backend operations. It supports multiple input forms (canonical, variations, short codes) and normalizes them into executable commands.

**Key principle:** Minimize typing, maximize flexibility.

---

## Architecture Layers

### 1. Generic Layer (Candidate for `aqm` lib)

**Responsibilities:**
- Command normalization (case, spacing, hyphens)
- Pattern matching and parameter extraction
- Command routing
- Error handling structure

**Components:**

```go
type CommandProcessor interface {
    Process(ctx, input string) (*Response, error)
}

type CommandRegistry struct {
    commands map[string]*CommandDefinition
}

type CommandDefinition struct {
    Canonical   string              // "open-order"
    Variations  []string            // ["open order", "openorder", "oo"]
    Pattern     *regexp.Regexp      // Parameter extraction
    Handler     CommandHandler      // Execution function
    MinParams   int                 // Validation
    MaxParams   int
}

func Normalize(input string) string {
    // lowercase → remove extra spaces → standardize hyphens
}
```

---

### 2. Specific Layer (Operations service)

**Responsibilities:**
- Domain-specific command definitions
- Integration with table/order services
- Response formatting (HTML for chat UI)
- Business logic and validation

**Command Registry Example:**

```go
registry := &CommandRegistry{
    commands: map[string]*CommandDefinition{
        "open-order": {
            Canonical:  "open-order",
            Variations: []string{"open order", "openorder", "oo"},
            Pattern:    regexp.MustCompile(`^(open|oo)[\s-]*(order)?\s+(.+)$`),
            Handler:    h.handleOpenOrder,
            MinParams:  1, // table_id required
            MaxParams:  1,
        },
        "add-item": {
            Canonical:  "add-item",
            Variations: []string{"add item", "additem", "ai"},
            Pattern:    regexp.MustCompile(`^(add|ai)[\s-]*(item)?\s+(\S+)\s+(\S+)\s+(\d+)$`),
            Handler:    h.handleAddItem,
            MinParams:  3, // order_id, item_code, quantity
            MaxParams:  3,
        },
        // ... etc
    },
}
```

---

## Command Forms

### Canonical Form (verb-noun)
Used internally and for web CQRS interface:
```
open-order
add-item
seat-party
```

### Conversational Form (natural)
User-friendly for chat/terminal:
```
open order 5
add item coffee 2
seat party at table 3
```

### Short Form (efficiency)
For experienced users during busy service:
```
oo 5          → returns: 47
ai 47 coffee 2
sp 3 4
```

---

## Normalization Pipeline

```
User Input: "Open Order 5"
    ↓
1. Lowercase: "open order 5"
    ↓
2. Tokenize: ["open", "order", "5"]
    ↓
3. Match variations:
   - Try: "open order" → matches "open-order" variations
    ↓
4. Extract params: table_id=5
    ↓
5. Validate: 1 param required ✓
    ↓
6. Route to handler: handleOpenOrder(ctx, "5")
    ↓
7. Execute & format response
```

---

## Variation Matching Strategy

**Priority order:**
1. **Exact match** (after normalization)
2. **Short form match** (`oo` → `open-order`)
3. **Variation match** (`openorder` → `open-order`)
4. **Fuzzy match** (optional, Phase 2 with LLM)

**Examples:**

| User Input | Normalized | Matched Canonical |
|------------|------------|-------------------|
| `Open-Order 5` | `open-order 5` | `open-order` ✓ |
| `open order 5` | `open order 5` | `open-order` ✓ |
| `oo 5` | `oo 5` | `open-order` ✓ |
| `OPENORDER 5` | `openorder 5` | `open-order` ✓ |

---

## Parameter Extraction

**Flexible patterns support:**
- Position-based: `add item <order_id> <item> <qty>`
- Named (future): `add item to order=123 item=coffee qty=2`
- Context-aware (future): "add coffee 2" (infers order from context)

**Current approach:**
```go
pattern := `^(add|ai)[\s-]*(item)?\s+(\S+)\s+(\S+)\s+(\d+)$`
matches := pattern.FindStringSubmatch(normalized)
// matches[3] = order_id
// matches[4] = item_code
// matches[5] = quantity
```

---

## Error Handling

**Structured responses:**

```go
type Response struct {
    HTML    string  // Formatted output for UI
    Success bool
    Message string  // Machine-readable status
    Error   error   // Optional error details
}
```

**Error types:**
- **Unknown command**: Show help with suggestions
- **Missing params**: Show command syntax
- **Invalid params**: Show validation error + expected format
- **Service error**: Show friendly message + log details

---

## Future Extensions

### Phase 2: LLM Integration

Augment deterministic parser with LLM for:
- Natural language understanding
- Context awareness
- Intent disambiguation
- Multi-command sequences

**Interface remains the same:**
```go
type LLMProcessor struct {
    model    OllamaClient
    registry *CommandRegistry  // Still used for validation
}

func (p *LLMProcessor) Process(ctx, input) (*Response, error) {
    // LLM extracts intent + params
    // Maps to canonical command
    // Executes through same handlers
}
```

### Alternative Interfaces

**Bubble Tea CLI:**
```bash
appetite-cli
> open order 5
✓ Order 47 opened for Table 5
```

**Web CQRS Interface:**
```http
POST /commands/open-order
{ "table_id": 5 }
```

Both use the same command registry and handlers.

---

## Implementation Phases

### Phase 1: Current (Deterministic)
- [x] Basic command registry
- [x] Pattern matching
- [x] Mock data responses
- [ ] All commands from specs (30+ commands)
- [ ] Real service integration

### Phase 2: LLM Enhancement
- [ ] Ollama integration
- [ ] Context management
- [ ] Intent extraction
- [ ] Fallback to deterministic

### Phase 3: Advanced
- [ ] Multi-step commands
- [ ] Undo/redo
- [ ] Command history
- [ ] Batch operations

---

## Security Considerations

- **Authentication**: All commands require valid session
- **Authorization**: Role-based command filtering (waiter vs manager)
- **Validation**: Input sanitization before service calls
- **Rate limiting**: Prevent command spam
- **Audit log**: Track all command executions
