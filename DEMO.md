# Appetite Demo Mode

This document describes the demo mode functionality for Appetite services.

## Overview

Demo mode provides realistic, pre-populated data for demonstration and testing purposes. When enabled, the system creates a complete restaurant scenario with tables, orders, and kitchen tickets in various states.

## Quick Start

### Run with Demo Data

```bash
make run-demo
```

This will:
1. Stop all running services
2. **Drop all databases** (complete reset via `db-reset-dev`)
3. Build all services
4. Start NATS
5. Start all services with demo seeding enabled

### Run with Standard Data

```bash
make run-all
```

This runs the normal seeding (users, grants, base tables, menu) without demo data.

## Configuration

Demo mode is controlled by the `seeding.demo` configuration flag. Each service reads this from its own configuration using the standard aqm config system.

**Enable via environment variable:**
```bash
# Each service uses its own namespace prefix
TABLE_SEEDING_DEMO=true ./table
ORDER_SEEDING_DEMO=true ./order
KITCHEN_SEEDING_DEMO=true ./kitchen
```

**Enable via config file:**
```yaml
seeding:
  demo: "true"
```

**How it works:**
- The aqm config system automatically maps environment variables to config keys
- Format: `{NAMESPACE}_{KEY}_{SUBKEY}` → `{key}.{subkey}`
- Example: `TABLE_SEEDING_DEMO=true` → `seeding.demo=true` in TABLE service config

## Demo Data

### Table Service

**Standard seeding runs first** (users, grants, 8 base tables, menu), then:

**5 tables modified to "open" status:**
- **Window-1**: 2 guests
- **Center-2**: 4 guests
- **Patio-3**: 1 guest
- **Booth-7**: 3 guests
- **Terrace-8**: 6 guests

### Order Service

**4 realistic scenarios created:**

**Scenario 1: Window-1 - Couple having drinks and desserts**
- Aperol Spritz (x1) - bar - ready
- Espresso Martini (x1) - bar - ready
- Chocolate Lava Cake (x1) - dessert - preparing - "Extra vanilla ice cream"
- Tiramisu (x1) - dessert - preparing

**Scenario 2: Center-2 - Group of 4 having dinner**
- Bistro Steak (x2) - kitchen - preparing - "Medium rare, no sauce on one"
- Seared Salmon (x1) - kitchen - preparing
- Harvest Bowl (x1) - kitchen - pending - "Gluten free"
- West Coast IPA (x2) - bar - ready
- Sparkling Water (x1) - no production - ready
- Cappuccino (x2) - coffee - pending

**Scenario 3: Patio-3 - Solo diner**
- Smash Burger (x1) - kitchen - ready - "No pickles"
- House Iced Tea (x1) - no production - ready - "No ice"

**Scenario 4: Booth-7 - Small group with cocktails**
- Classic Martini (x2) - bar - ready - "Shaken, not stirred"
- Old Fashioned (x1) - bar - preparing - "Extra orange peel"
- Negroni (x1) - bar - pending

### Kitchen Service

Creates kitchen tickets for **all order items that require production**, mapping statuses:
- `pending` → created
- `preparing` → started (with `started_at` timestamp)
- `ready` → ready (with `started_at` and `finished_at`)
- `delivered` → delivered (with full timestamp chain)

**Production stations:**
- Kitchen (main line)
- Bar (cocktails, beer)
- Dessert (pastry)
- Coffee (espresso bar)

## Implementation Details

### Architecture

**File structure (flat, Go-idiomatic):**
```
services/table/internal/tables/seeding_demo.go
services/order/internal/order/seeding_demo.go
services/kitchen/internal/kitchen/seeding_demo.go
```

**Execution order:**
1. Table service: Standard seeding → Demo modifications
2. Order service: Creates orders + items (depends on tables)
3. Kitchen service: Creates tickets (depends on order items)

### Seeding Mechanics

- **Idempotent**: Uses `seed.Apply()` with tracker (MongoDB `_seeds` collection)
- **Background execution**: Runs in goroutines with context cancellation
- **Error handling**: Logs failures without crashing services
- **Dependencies**: Order service waits for tables, kitchen waits for order items

### Data Consistency

All demo data uses:
- `created_by: "seed:demo"` for tracking
- Realistic timestamps (items created 5-45 minutes ago)
- Status progression (older items tend to be ready/delivered)
- Coherent relationships (tickets match order items)

## Database Reset

`make run-demo` performs a **full database reset**:

```bash
# Drops ALL databases:
- appetite_authn
- appetite_authz
- appetite_dictionary
- appetite_menu
- appetite_order
- appetite_table
- appetite_kitchen
```

This ensures a clean slate before demo seeding.

## Service Startup Order

```
run-demo sequence:
1. Stop all services
2. db-reset-dev (drops all DBs)
3. Build all services
4. Start NATS
5. Start services:
   - Admin, AuthN, AuthZ (standard seeding)
   - Dictionary, Menu (standard seeding)
   - Table (sleep 3s) - standard + demo
   - Order (sleep 3s) - demo only
   - Kitchen (sleep 3s) - demo only
   - Operations, Media
```

Extra sleep time (3s vs 2s) for table/order/kitchen allows seeding to complete.

## Development Notes

### Adding New Demo Scenarios

To add more demo scenarios, edit:
- `services/order/internal/order/seeding_demo.go` - add new `createScenarioN()` function
- Update scenario in `seedDemoOrders()` to call it
- Kitchen tickets will automatically be created

### Station IDs

Fixed UUIDs used for production stations:
```go
barStation     = "00000000-0000-0000-0000-000000000001"
kitchenStation = "00000000-0000-0000-0000-000000000002"
dessertStation = "00000000-0000-0000-0000-000000000003"
coffeeStation  = "00000000-0000-0000-0000-000000000004"
```

### Status IDs (Kitchen)

Kitchen ticket status mapping:
```go
statusCreatedID   = "10000000-0000-0000-0000-000000000001"
statusStartedID   = "10000000-0000-0000-0000-000000000002"
statusReadyID     = "10000000-0000-0000-0000-000000000003"
statusDeliveredID = "10000000-0000-0000-0000-000000000004"
```

## Troubleshooting

**Tables not appearing?**
- Check table service logs: `tail -f services/table/table.log`
- Verify demo seeding ran: look for "Demo seeding enabled for table service"

**Order items missing?**
- Ensure tables were created first (order seeding depends on tables)
- Check order service logs: `tail -f services/order/order.log`

**Kitchen tickets not created?**
- Verify order items exist first
- Check kitchen service logs: `tail -f services/kitchen/kitchen.log`
- Look for "Demo kitchen tickets seeded successfully"

**Full reset needed?**
```bash
make stop-all
make db-reset-dev
make run-demo
```

## API Endpoints

After `make run-demo`, demo data is available via:

**Tables:**
```bash
curl http://localhost:8087/tables
```

**Orders:**
```bash
curl http://localhost:8086/orders
```

**Kitchen Tickets:**
```bash
curl http://localhost:8089/tickets
```

**Operations Dashboard:**
```
http://localhost:8080
```
