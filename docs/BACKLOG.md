# BACKLOG

## Operations Service
- Create operations service (port 8080) with HTML + HTMX stack.
- Implement simplified navigation menu with Chat entry point.
- Embed Chat UI (Hugging Face SvelteKit) within the service.
- Configure service to run under agent identity with access to authn service.

## Order Service
- Extract order-related functionality from table service to dedicated order service (port 8086).
- Migrate OrderRepo, OrderItemRepo, and related domain logic.
- Implement RESTful endpoints for order lifecycle operations.
- Establish communication with table service for table state queries.
- Update table service to delegate order operations to order service.

## Conversational Interface
- ✅ Create `agent@system` user to act as mediator in conversational sessions.
- ✅ Add auto-generated `PIN` field to `User` for transient authentication.
- ✅ Implement PIN generation with collision detection.
- Implement deterministic command parser for structured operations.
- Create command handlers for basic table and order operations.
- Bind user identity to chat session via PIN authentication.
- Handle session lifecycle (authentication, active session, exit/cleanup).

## Future Enhancements
- Integrate locally hosted, fine-tuned LLM for natural language understanding.
- Add voice interaction capabilities to conversational mode.
- Implement comprehensive order workflow in conversational interface.  
