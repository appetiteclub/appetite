# ROADMAP

## Service Architecture
- **Operations Service** (port 8080): Day-to-day restaurant operations interface.
  - HTML + HTMX service with simplified navigation.
  - Embedded Chat UI as conversational gateway.
  - Coexists with Admin service (port 8081) for configuration/management.
- **Order Service** (port 8086): Dedicated service for order lifecycle management.
  - Extracted from table service for cleaner separation of concerns.
  - Interacts with table service for table state queries.
- **Table Service** (port 8087): Focused on table and reservation management.

## Conversational Mode
- Core of the system.
- Deterministic command interface (text-based) for structured operations.
- Progressive enhancement with locally hosted, fine-tuned LLM for fluency and guidance.
- Transient PIN-based authentication binding user identity to chat sessions.
- Initial integration with table and order services via basic commands.

## Graphical Mode
- Separate interaction mode.
- Built after conversational mode is solid.
- Uses the same backend actions.

## Core System
- Shared logic, data, and state management.
- Independent of the chosen interaction mode.  

