# Seeding Architecture Draft

This note documents how Appetite services share a deterministic seeding workflow using the new `github.com/aquamarinepk/aqm/seed` package.

## Shared Building Blocks

- `seed.Seed`: describes a versioned, idempotent mutation (`ID`, `Description`, `Run func(context.Context) error`).
- `seed.Tracker`: persistence contract (`HasRun`/`MarkRun`). Each service currently uses `seed.NewMongoTracker`, which records executions in the `_seeds` collection of its Mongo database.
- `seed.Apply`: walks the ordered slice of seeds, runs each one once, and records completion metadata (`application`, description, timestamp). If a service restarts, previously recorded IDs are skipped, so the workflow converges on the same state.
- `seed.UpsertOnce`: convenience around `$setOnInsert` when building large seeds (e.g., dictionary options), keeping each insert idempotent.

## Service Responsibilities

### Dictionary
- Seeds are defined in `services/dictionary/internal/dictionary/seeding.go` and use the same Mongo database as the app.
- During lifecycle `OnStart`, the service creates a tracker from the repo’s DB handle and calls `seed.Apply`; if the `_seeds` record exists, nothing runs.

### AuthN
- User data stays in AuthN. `ApplyUserSeeds` waits for the superadmin bootstrap to finish (polling `/system/bootstrap-status`) to guarantee the system owner already exists.
- Once the superadmin is present, the service loads `pkg/data/bootstrap/bootstrap_seed.json`, builds one `seed.Seed` per user (skipping auto-generated or reference entries as needed), and runs `seed.Apply` using the AuthN database tracker.
- Each seed ID is deterministic (`2024-11-15_authn_user_<slug>`), so the tracker behaves predictably across environments.

### AuthZ
- On startup AuthZ also checks AuthN’s bootstrap status. If AuthN still needs bootstrap, AuthZ triggers it; otherwise it reuses the reported superadmin ID.
- Role seeds combine defaults (superadmin/admin/user) with everything under the `roles` section of `bootstrap_seed.json`. Each becomes a `seed.Seed` with ID `2024-11-15_authz_role_<slug>`.
- AuthZ applies those seeds through the shared tracker tied to its Mongo DB, guaranteeing every required role exists before grants are created.
- After seeding roles, AuthZ ensures the superadmin grant by looking up the superadmin role ID and creating the grant if it isn’t recorded yet.

## Ordering Guarantees

1. AuthN bootstrap runs first (superadmin creation). AuthZ triggers it if necessary, so we never try to grant roles before the user exists.
2. AuthN user seeds wait for the bootstrap signal, ensuring they don’t race with the superadmin creation.
3. AuthZ role seeds run through `seed.Apply`, which uses the `_seeds` tracker to prevent duplicates across parallel instances or restarts.
4. Once roles exist, AuthZ enforces the superadmin grant idempotently by checking existing grants.

Because each service manages its own tracker within its database, the workflow tolerates non-deterministic startup ordering: whichever instance reaches seeding first will record the completion, and the others will observe that record and skip re-running the mutation.
