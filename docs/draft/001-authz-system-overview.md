# Authorization System Architecture

**Status:** Draft | **Updated:** 2025-10-19 | **Version:** 0.1

## Overview

This document explains the **Authorization (AuthZ) system** architecture for Appetite. The system implements a flexible, scope-aware authorization model using **roles**, **permissions**, and **grants** to control access to resources across different contexts (global, team, organization).

The system is designed to support complex access control scenarios where users can have different capabilities in different contexts, with support for both role-based and direct permission assignments.

## 1. Core Concepts

### 1.1 Permission

A **permission** is a simple string code representing a specific action capability:

```
"users:read"       - Can view users
"users:write"      - Can create/modify users
"tables:manage"   - Can manage tables
"tables:delete"   - Can delete tables
"grants:write"     - Can assign permissions
"system:admin"     - System administration
```

**Important**: Permissions are **not stored in the database** - they are constants defined in code. They represent atomic capabilities in the system.

### 1.2 Role

A **role** is a named collection of permissions stored in the database:

```go
type Role struct {
    ID          uuid.UUID
    Name        string           // "TeamAdmin", "EstateManager", "Viewer"
    Permissions []string         // ["users:read", "tables:manage"]
    Status      UserStatus       // active, suspended, deleted
    CreatedAt   time.Time
    CreatedBy   string
    UpdatedAt   time.Time
    UpdatedBy   string
}
```

**MongoDB document:**
```json
{
  "_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "TeamAdmin",
  "permissions": [
    "users:read",
    "users:write",
    "tables:manage"
  ],
  "status": "active",
  "created_at": "2025-10-15T10:00:00Z",
  "created_by": "admin-user-id",
  "updated_at": "2025-10-15T10:00:00Z",
  "updated_by": "admin-user-id"
}
```

**Purpose**: Roles bundle related permissions together for easier management. Instead of assigning 10 individual permissions to every team admin, you create a "TeamAdmin" role once and assign it to users.

### 1.3 Grant

A **grant** is the assignment of a role or permission to a specific user in a specific context (scope):

```go
type Grant struct {
    ID        uuid.UUID
    UserID    uuid.UUID         // Which user receives the grant
    GrantType GrantType         // "role" or "permission"
    Value     string            // Role ID (if type=role) or permission string (if type=permission)
    Scope     Scope             // Where the grant applies
    ExpiresAt *time.Time        // Optional expiration
    Status    UserStatus        // active, suspended, deleted
    CreatedAt time.Time
    CreatedBy string
    UpdatedAt time.Time
    UpdatedBy string
}

type GrantType string
const (
    GrantTypeRole       GrantType = "role"        // Value is a Role ID
    GrantTypePermission GrantType = "permission"  // Value is a permission string
)

type Scope struct {
    Type string  // "global", "team", "organization"
    ID   string  // Specific context ID (e.g., "team-123"), or empty for global
}
```

**MongoDB document (role grant):**
```json
{
  "_id": "grant-789",
  "user_id": "john-doe-123",
  "grant_type": "role",
  "value": "550e8400-e29b-41d4-a716-446655440000",
  "scope": {
    "type": "team",
    "id": "appetite-team-001"
  },
  "expires_at": null,
  "status": "active",
  "created_at": "2025-10-15T10:30:00Z",
  "created_by": "admin-user-id"
}
```

**MongoDB document (direct permission grant):**
```json
{
  "_id": "grant-999",
  "user_id": "john-doe-123",
  "grant_type": "permission",
  "value": "tables:delete",
  "scope": {
    "type": "global",
    "id": ""
  },
  "expires_at": "2025-11-15T00:00:00Z",
  "status": "active"
}
```

**Purpose**: Grants are the "glue" between users and their capabilities. They answer: "What can this user do, and where?"

## 2. Complete Authorization Flow

### 2.1 Example Scenario

**Setup:**
- User: **John Doe** (ID: `john-doe-123`)
- Role: **TeamAdmin** (ID: `role-456`, permissions: `["users:read", "tables:manage"]`)
- Team: **Appetite Engineering** (ID: `appetite-team-001`)

**Grant Assignment:**
Assign the TeamAdmin role to John in the Appetite team:

```json
{
  "_id": "grant-789",
  "user_id": "john-doe-123",
  "grant_type": "role",
  "value": "role-456",
  "scope": {
    "type": "team",
    "id": "appetite-team-001"
  }
}
```

**Result:**
John now has `users:read` and `tables:manage` permissions **within the Appetite team context**.

### 2.2 Multiple Grants Per User

A user can have **multiple grants** with different types and scopes:

```json
[
  {
    "_id": "grant-001",
    "user_id": "john-doe-123",
    "grant_type": "role",
    "value": "superadmin-role-id",
    "scope": { "type": "global", "id": "" }
  },
  
  {
    "_id": "grant-002",
    "user_id": "john-doe-123",
    "grant_type": "role",
    "value": "teamadmin-role-id",
    "scope": { "type": "team", "id": "appetite-team" }
  },
  
  {
    "_id": "grant-003",
    "user_id": "john-doe-123",
    "grant_type": "permission",
    "value": "tables:delete",
    "scope": { "type": "team", "id": "alpha-team" }
  },
  
  {
    "_id": "grant-004",
    "user_id": "john-doe-123",
    "grant_type": "permission",
    "value": "system:maintenance",
    "scope": { "type": "global", "id": "" },
    "expires_at": "2025-11-18T00:00:00Z"
  }
]
```

### 2.3 Permission Evaluation by Context

When the AuthZ service evaluates permissions, it **aggregates all applicable grants** for the requested context:

#### Context: Team Appetite
```
Question: "Can John manage tables in team Appetite?"

Query grants:
  - user_id = "john-doe-123"
  - status = "active"
  - scope matches (global OR team=appetite)
  - not expired

Found grants: grant-001 (global SuperAdmin), grant-002 (TeamAdmin in Appetite)

Expand roles:
  - SuperAdmin: ["users:*", "tables:*", "teams:*", ...]
  - TeamAdmin: ["users:read", "users:write", "tables:manage"]

Effective permissions: UNION of both = all SuperAdmin + TeamAdmin permissions

Check: "tables:manage" in effective permissions?
Result: ✅ YES (from both roles)
```

#### Context: Team Alpha
```
Question: "Can John delete tables in team Alpha?"

Query grants:
  - scope matches (global OR team=alpha)

Found grants: grant-001 (global SuperAdmin), grant-003 (tables:delete in Alpha)

Effective permissions:
  - SuperAdmin: ["users:*", "tables:*", ...]
  - Direct: ["tables:delete"]

Check: "tables:delete" in effective permissions?
Result: ✅ YES (from both SuperAdmin and direct grant)
```

#### Context: Global (no specific team)
```
Question: "Can John perform system maintenance?"

Query grants:
  - scope = global only

Found grants: grant-001 (SuperAdmin), grant-004 (system:maintenance, if not expired)

Check: "system:maintenance" in effective permissions?
Result: ✅ YES (if grant-004 hasn't expired)
```

## 3. Data Model Details

### 3.1 Collections

The AuthZ service uses two primary MongoDB collections:

#### `authz.roles`
Stores role definitions.

**Indexes:**
- Unique on `name`
- `status`
- `created_at`

**Example:**
```json
{
  "_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "EstateManager",
  "permissions": [
    "tables:read",
    "tables:write",
    "tables:manage",
    "tables:delete"
  ],
  "status": "active",
  "created_at": "2025-10-15T10:00:00Z",
  "created_by": "admin-user-id",
  "updated_at": "2025-10-15T10:00:00Z",
  "updated_by": "admin-user-id"
}
```

#### `authz.grants`
Stores user permission assignments.

**Indexes:**
- `user_id` (primary lookup)
- `scope.type`
- `scope.id`
- `status`
- TTL on `expires_at` (automatic cleanup of expired grants)

**Example:**
```json
{
  "_id": "grant-uuid-here",
  "user_id": "john-doe-123",
  "grant_type": "role",
  "value": "550e8400-e29b-41d4-a716-446655440000",
  "scope": {
    "type": "team",
    "id": "appetite-team-001"
  },
  "expires_at": null,
  "status": "active",
  "created_at": "2025-10-15T10:30:00Z",
  "created_by": "admin-user-id",
  "updated_at": "2025-10-15T10:30:00Z",
  "updated_by": "admin-user-id"
}
```

### 3.2 Go Domain Structs

Located in `pkg/lib/auth/types.go` and `services/authz/internal/authz/`:

```go
// Simplified core types
package auth

type Role struct {
    ID          uuid.UUID
    Name        string
    Permissions []string
}

type Grant struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    GrantType GrantType  // "role" or "permission"
    Value     string
    Scope     Scope
    ExpiresAt *time.Time
}

type GrantType string
const (
    GrantTypeRole       GrantType = "role"
    GrantTypePermission GrantType = "permission"
)

type Scope struct {
    Type string  // "global", "team", "organization"
    ID   string  // Context-specific ID or empty for global
}
```

## 4. Authorization Query Pattern

### 4.1 Typical Query Flow

When a service needs to check permissions:

```
1. Service receives request with user_id and context (e.g., team_id)
2. Service calls AuthZ: "Can user_id perform action X in context Y?"
3. AuthZ queries MongoDB:
```

```javascript
db.grants.find({
  user_id: "john-doe-123",
  status: "active",
  $or: [
    { "scope.type": "global" },
    { "scope.type": "team", "scope.id": "appetite-team-001" }
  ],
  $or: [
    { expires_at: null },
    { expires_at: { $gt: new Date() } }
  ]
})
```

```
4. AuthZ expands roles:
   - For each grant with grant_type="role", fetch the role and get its permissions
   - For each grant with grant_type="permission", use the value directly

5. AuthZ builds effective permission set (UNION of all permissions)

6. AuthZ checks if requested permission is in the effective set

7. Returns: { "allowed": true/false }
```

### 4.2 Go Implementation Sketch

```go
func (s *AuthZService) CheckPermission(
    ctx context.Context,
    userID uuid.UUID,
    permission string,
    scope Scope,
) (bool, error) {
    // 1. Query grants for user in requested scope
    grants, err := s.grantRepo.FindByUserAndScope(ctx, userID, scope)
    if err != nil {
        return false, err
    }
    
    // 2. Filter expired grants
    activeGrants := filterActive(grants)
    
    // 3. Expand roles and collect permissions
    effectivePerms := make(map[string]bool)
    
    for _, grant := range activeGrants {
        if grant.GrantType == GrantTypePermission {
            effectivePerms[grant.Value] = true
        } else if grant.GrantType == GrantTypeRole {
            role, err := s.roleRepo.FindByID(ctx, grant.Value)
            if err != nil {
                continue
            }
            for _, perm := range role.Permissions {
                effectivePerms[perm] = true
            }
        }
    }
    
    // 4. Check if requested permission exists
    return effectivePerms[permission], nil
}
```

## 5. Use Cases and Examples

### 5.1 Global Administrator

**Scenario:** Jane needs full system access everywhere.

**Setup:**
```json
{
  "role": {
    "name": "SystemAdmin",
    "permissions": ["*"]
  },
  "grant": {
    "user_id": "jane-doe-456",
    "grant_type": "role",
    "value": "systemadmin-role-id",
    "scope": { "type": "global", "id": "" }
  }
}
```

**Result:** Jane can do anything, anywhere.

### 5.2 Team-Scoped Administrator

**Scenario:** Bob manages only the "Sales" team.

**Setup:**
```json
{
  "role": {
    "name": "TeamAdmin",
    "permissions": ["users:read", "users:write", "tables:manage"]
  },
  "grant": {
    "user_id": "bob-smith-789",
    "grant_type": "role",
    "value": "teamadmin-role-id",
    "scope": { "type": "team", "id": "sales-team" }
  }
}
```

**Result:** Bob has admin capabilities only within the Sales team context.

### 5.3 Temporary Permission

**Scenario:** Alice needs emergency access to delete tables for 7 days.

**Setup:**
```json
{
  "grant": {
    "user_id": "alice-jones-321",
    "grant_type": "permission",
    "value": "tables:delete",
    "scope": { "type": "global", "id": "" },
    "expires_at": "2025-10-26T00:00:00Z"
  }
}
```

**Result:** Alice can delete tables globally, but only until October 26. After that, the grant expires automatically (via MongoDB TTL index).

### 5.4 Multiple Teams, Different Roles

**Scenario:** John is a TeamAdmin in "Engineering" but only a Viewer in "Finance".

**Setup:**
```json
[
  {
    "user_id": "john-doe-123",
    "grant_type": "role",
    "value": "teamadmin-role-id",
    "scope": { "type": "team", "id": "engineering-team" }
  },
  {
    "user_id": "john-doe-123",
    "grant_type": "role",
    "value": "viewer-role-id",
    "scope": { "type": "team", "id": "finance-team" }
  }
]
```

**Result:**
- In Engineering: Full admin capabilities
- In Finance: Read-only access

### 5.5 Hybrid Role + Direct Permission

**Scenario:** Sarah has a Manager role in Marketing, plus a special direct permission to export data globally.

**Setup:**
```json
[
  {
    "user_id": "sarah-wilson-654",
    "grant_type": "role",
    "value": "manager-role-id",
    "scope": { "type": "team", "id": "marketing-team" }
  },
  {
    "user_id": "sarah-wilson-654",
    "grant_type": "permission",
    "value": "data:export",
    "scope": { "type": "global", "id": "" }
  }
]
```

**Result:**
- In Marketing: All manager capabilities (from role)
- Everywhere: Can export data (from direct permission)

## 6. Scope Hierarchy and Matching

### 6.1 Scope Types

```
global          - Applies everywhere (highest precedence)
organization    - Applies to an entire organization
team            - Applies to a specific team
```

### 6.2 Scope Matching Rules

When checking permissions in a context:

1. **Global grants always apply** - If a user has a global grant, it applies in every context
2. **Specific grants require exact match** - A team-scoped grant only applies in that specific team
3. **No inheritance** - Team grants don't automatically apply to sub-resources (future extension)

**Example:**
```
User has grant with scope { type: "global" }
  → Applies in context: team-A, team-B, org-X, anywhere

User has grant with scope { type: "team", id: "team-A" }
  → Applies in context: team-A only
  → Does NOT apply in: team-B, global, org-X
```

### 6.3 Implementation

```go
func (g *Grant) MatchesScope(requestedScope Scope) bool {
    // Global scope matches everything
    if g.Scope.Type == "global" {
        return true
    }
    
    // Exact match required for specific scopes
    return g.Scope.Type == requestedScope.Type && 
           g.Scope.ID == requestedScope.ID
}
```

## 7. Common Permission Patterns

### 7.1 CRUD-Based Permissions

```
resource:read     - View/list resources
resource:write    - Create/update resources
resource:delete   - Remove resources
resource:manage   - Full control (implies read, write, delete)
```

**Example:**
```
tables:read
tables:write
tables:delete
tables:manage   (includes all above)
```

### 7.2 Action-Based Permissions

```
resource:action   - Specific operation
```

**Example:**
```
users:invite      - Can invite new users
users:suspend     - Can suspend user accounts
reports:export    - Can export reports
```

### 7.3 Wildcard Permissions

```
*                 - System administrator (all permissions)
resource:*        - All actions on a resource
```

**Example:**
```
users:*           - All user operations
tables:*         - All table operations
```

## 8. Authorization Repository Interface

### 8.1 Role Repository

```go
type RoleRepo interface {
    Create(ctx context.Context, role *Role) error
    FindByID(ctx context.Context, id uuid.UUID) (*Role, error)
    FindByName(ctx context.Context, name string) (*Role, error)
    List(ctx context.Context) ([]*Role, error)
    Update(ctx context.Context, role *Role) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### 8.2 Grant Repository

```go
type GrantRepo interface {
    Create(ctx context.Context, grant *Grant) error
    FindByID(ctx context.Context, id uuid.UUID) (*Grant, error)
    FindByUser(ctx context.Context, userID uuid.UUID) ([]*Grant, error)
    FindByUserAndScope(ctx context.Context, userID uuid.UUID, scope Scope) ([]*Grant, error)
    List(ctx context.Context) ([]*Grant, error)
    Update(ctx context.Context, grant *Grant) error
    Delete(ctx context.Context, id uuid.UUID) error
}
```

## 9. Integration with Services

### 9.1 Preflight Permission Check

Services should check permissions **before** performing operations:

```go
func (h *EstateHandler) DeleteEstate(w http.ResponseWriter, r *http.Request) {
    userID := extractUserFromContext(r)
    teamID := chi.URLParam(r, "teamId")
    tableID := chi.URLParam(r, "tableId")
    
    // Check permission before proceeding
    scope := Scope{Type: "team", ID: teamID}
    allowed, err := h.authzClient.CheckPermission(
        r.Context(),
        userID,
        "tables:delete",
        scope,
    )
    
    if err != nil || !allowed {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // Proceed with deletion
    if err := h.tableRepo.Delete(r.Context(), tableID); err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}
```

### 9.2 UI Conditional Rendering

Frontend templates can check permissions to show/hide actions:

```html
{{if .CanDeleteEstates}}
<button hx-delete="/teams/{{.TeamID}}/tables/{{.EstateID}}"
        class="btn-delete">
    Delete Estate
</button>
{{end}}
```

The backend populates `CanDeleteEstates` by checking permissions during page load.

## 10. Performance and Caching

### 10.1 Caching Strategy

**Grant Lookups:**
- Cache grants by `user_id + scope` for 30-120 seconds
- Invalidate on grant creation/deletion/update

**Role Definitions:**
- Cache roles by ID for 5-10 minutes (roles rarely change)
- Invalidate on role update

**Permission Checks:**
- Cache final permission check results for 30-60 seconds
- Key: `{userID}:{permission}:{scope.type}:{scope.id}`

### 10.2 Query Optimization

**Indexes on `authz.grants`:**
```javascript
db.grants.createIndex({ user_id: 1, status: 1 })
db.grants.createIndex({ user_id: 1, "scope.type": 1, "scope.id": 1 })
db.grants.createIndex({ expires_at: 1 }, { expireAfterSeconds: 0 })  // TTL
```

## 11. Administration Operations

### 11.1 Creating a Role

```bash
POST /authz/roles
Content-Type: application/json

{
  "name": "ContentEditor",
  "permissions": [
    "content:read",
    "content:write",
    "media:upload"
  ]
}
```

### 11.2 Assigning a Role to a User

```bash
POST /authz/grants
Content-Type: application/json

{
  "user_id": "john-doe-123",
  "grant_type": "role",
  "value": "contenteditor-role-id",
  "scope": {
    "type": "team",
    "id": "marketing-team"
  }
}
```

### 11.3 Granting a Direct Permission

```bash
POST /authz/grants
Content-Type: application/json

{
  "user_id": "jane-smith-456",
  "grant_type": "permission",
  "value": "system:maintenance",
  "scope": {
    "type": "global",
    "id": ""
  },
  "expires_at": "2025-10-26T00:00:00Z"
}
```

### 11.4 Listing User Permissions

```bash
GET /authz/users/john-doe-123/permissions?scope_type=team&scope_id=marketing-team

Response:
{
  "user_id": "john-doe-123",
  "scope": { "type": "team", "id": "marketing-team" },
  "effective_permissions": [
    "content:read",
    "content:write",
    "media:upload",
    "users:read"
  ],
  "grants": [
    {
      "id": "grant-001",
      "grant_type": "role",
      "role_name": "ContentEditor",
      "scope": { "type": "team", "id": "marketing-team" }
    },
    {
      "id": "grant-002",
      "grant_type": "permission",
      "value": "users:read",
      "scope": { "type": "global", "id": "" }
    }
  ]
}
```

## 12. Security Considerations

### 12.1 Grant Creation Authorization

Only users with `grants:write` permission should be able to create/modify grants.

### 12.2 Principle of Least Privilege

Always grant the **minimum necessary permissions** in the **narrowest possible scope**.

**Bad:**
```json
{ "grant_type": "role", "value": "admin-role", "scope": { "type": "global" } }
```

**Good:**
```json
{ "grant_type": "role", "value": "team-admin-role", "scope": { "type": "team", "id": "specific-team" } }
```

### 12.3 Temporal Grants

Use `expires_at` for temporary elevated permissions:
- Emergency access
- Time-limited promotions
- Temporary contractors

### 12.4 Audit Trail

All grants include `created_by`, `created_at`, `updated_by`, `updated_at` for audit purposes.

## 13. Future Extensions

### 13.1 Planned Features

- **Attribute-Based Access Control (ABAC)**: Evaluate permissions based on user attributes, resource properties, and environmental context
- **Hierarchical Scopes**: Team grants automatically apply to sub-teams
- **Conditional Permissions**: Time-of-day, IP-based, or other conditional rules
- **Permission Bundles**: Pre-defined sets of permissions for common use cases
- **Role Templates**: Quick-start roles for new teams/organizations

### 13.2 Integration Points

- **Event Bus**: Publish `GrantCreated`, `GrantRevoked` events for other services
- **Admin UI**: Visual grant management interface
- **Audit Service**: Comprehensive logging of all authorization changes
- **Metrics**: Track permission check latency, cache hit rates, common denials

## Conclusion

The Appetite authorization system provides a flexible way to control access to resources using:

1. **Permissions** - Atomic capabilities (strings)
2. **Roles** - Named bundles of permissions (database entities)
3. **Grants** - Assignments of roles/permissions to users in specific scopes (database entities)

The system supports:
- Multiple grants per user
- Mixed role-based and direct permission grants
- Scope-aware access control (global, team, organization)
- Temporal grants with automatic expiration
- Efficient caching and query optimization

