# Demo & Test Data Seeding

Quick guide for seeding test/demo data in Appetite during development.

## Quick Start

```bash
# Start application (runs natural seeding automatically)
make run-all

# Add demo data
make seed-demo

# Clear demo data
make clear-demo
```

## Basic Workflow

### 1. First Run / After Reset

```bash
make run-all
```

**What happens:**
- Kills processes on required ports
- Builds all services
- Starts NATS
- Starts all services in background
- **Natural seeding runs automatically** on service startup:
  - AuthN: Super Admin and Agent users
  - AuthZ: Roles and permissions
  - Dictionary: Dictionary sets
  - Table: 8 tables (from `services/table/seed.json`)
  - Menu: Menu items with stations

### 2. Add Demo Data

```bash
make seed-demo
```

**Creates:**
- 3 orders on 3 different tables
- 12 order items with states: `pending`, `preparing`, `ready`
- 10 kitchen tickets (items requiring production)

**Features:**
- ✅ Doesn't restart services
- ✅ Fast (~3 seconds)
- ✅ Idempotent (can run multiple times)
- ✅ Automatic seed tracking

### 3. Clear Demo Data

```bash
make clear-demo
```

**Removes:**
- Orders with `created_by: "demo-seed"`
- Order items with `created_by: "demo-seed"`
- Kitchen tickets with `created_by: "demo-seed"`

**Preserves:**
- Users
- Tables
- Menu
- Roles/permissions
- Dictionaries

## Common Workflows

### Daily Development

```bash
# Day 1
make run-all
make seed-demo

# Day 2+
make run-all  # Data persists
```

### Testing with Clean Data

```bash
make clear-demo
make seed-demo
# Test your feature
```

### Client Demo

```bash
make fresh-start  # Complete reset
make seed-demo
# Open http://localhost:8080
```

### Full Reset

```bash
make fresh-start
```

**Does:**
- Stops services
- Clears logs
- Drops all databases
- Restarts services (natural seeding runs)
- Shows logs in real-time

## Utility Commands

### View Status

```bash
# Real-time logs
make log-clean

# Condensed logs
make logs

# View processes
ps aux | grep appetite

# View used ports
lsof -ti:8080,8081,8082,8083,8084,8085,8086,8087,8088,8089,8090
```

### Stop Everything

```bash
make stop-all
```

### Verify Data

```bash
# Count demo items
mongosh "mongodb://admin:password@localhost:27017/admin?authSource=admin" \
  --quiet --eval 'db.getSiblingDB("appetite_order").order_items.find({created_by: "demo-seed"}).count()'

# Count demo tickets
mongosh "mongodb://admin:password@localhost:27017/admin?authSource=admin" \
  --quiet --eval 'db.getSiblingDB("appetite_kitchen").tickets.find({created_by: "demo-seed"}).count()'
```

## Utility CLI - appetite-utils

The utility CLI is at `bin/appetite-utils`:

```bash
# Build
make build-utils

# Available commands
./bin/appetite-utils help
./bin/appetite-utils seed-demo
./bin/appetite-utils clear-demo
./bin/appetite-utils reset-db  # ⚠️ DANGEROUS - drops ALL databases
```

### Environment Variables

```bash
# Change MongoDB URL
UTILS_MONGO_URL=mongodb://localhost:27017 ./bin/appetite-utils seed-demo

# Change log level
UTILS_LOG_LEVEL=debug ./bin/appetite-utils seed-demo
```

## Seeding Architecture

### Natural Seeding (Automatic on service startup)

Each service has its own seeding that runs on startup:

**AuthN** (`services/authn/seed.json`):
- Super Admin user
- Agent user

**Table** (`services/table/seed.json`):
- 8 tables with different states and capacities

**Menu** (`services/menu/internal/menu/seeding.go`):
- Menu items with stations (kitchen, bar, coffee, dessert)
- Menu dictionary (allergens, dietary, categories)

**Dictionary** (`services/dictionary/internal/dictionary/seeding.go`):
- Base dictionary sets

**AuthZ** (`services/authz/internal/authz/bootstrap.go`):
- Base roles and permissions

### Demo Seeding (Manual via CLI)

**Independent from services:**
- Doesn't require services to be running
- No coupling between CLI and service internals
- Logic duplicated in `cmd/utils/internal/seeding/`

**Order Demo** (`cmd/utils/internal/seeding/order.go`):
- 3 realistic order scenarios
- Items distributed across different stations
- Varied states for testing

**Kitchen Demo** (`cmd/utils/internal/seeding/kitchen.go`):
- Tickets matching order items
- Status mapping (pending → created, preparing → started, etc)
- Only items requiring production

## Seed Tracking

Seeds are tracked in the `_seeds` collection:

```javascript
{
  _id: "demo_orders_v1",
  description: "Create demo orders...",
  applied_at: ISODate("2024-11-22T...")
}
```

**Benefits:**
- Prevents duplication
- Allows versioning (v1, v2, etc)
- Easy to clean up

## Troubleshooting

### "No demo order items found"
```bash
# Order seeding didn't run
make seed-demo
```

### "need at least 3 tables for demo data"
```bash
# Natural seeding tables are missing
make fresh-start
make seed-demo
```

### Demo data doesn't appear in UI
```bash
# 1. Verify data in DB
mongosh "mongodb://admin:password@localhost:27017/admin?authSource=admin" \
  --quiet --eval 'db.getSiblingDB("appetite_order").order_items.count()'

# 2. Verify services running
ps aux | grep -E "(order|kitchen)"

# 3. Check logs
tail -f services/order/order.log
tail -f services/kitchen/kitchen.log
```

### Services won't start
```bash
make stop-all
killall -9 authn authz order kitchen table menu operations admin media dictionary
make run-all
```

## Old vs New Approach

### ❌ Old (DISABLED)
```bash
make run-all
SEED_DEMO_ENABLED=true make run-all  # Restarted everything
```

**Problems:**
- Restarted ALL services
- Slow (30+ seconds)
- Depended on env vars
- Fragile

### ✅ New (CURRENT)
```bash
make run-all    # Once
make seed-demo  # As many times as needed
```

**Benefits:**
- Services keep running
- Fast (~3 seconds)
- Independent
- Reliable
