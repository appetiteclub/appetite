# BACKLOG

## Public Interface
- Implement the public-facing interface (non-admin).  
- Focus on the conversational mode first.  

## Conversational Agent
- Create a `User` / profile type `Agent` to act as mediator in the conversational interface.  

## Authentication
- Add an auto-generated `PIN` field to `User` for soft-login via the Agent.  
- Ensure no PIN collisions between users.  
- Extend the User CRUD to generate and update the PIN automatically.  
