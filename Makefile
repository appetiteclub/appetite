# Makefile for the Appetite monorepo

# Variables
PROJECT_NAME=appetite
SERVICES=authn authz dictionary menu order table operations admin media kitchen
UTILS=utils
BASE_PORTS=8080 8081 8082 8083 8084 8085 8086 8087 8088 8089 8090
PKG_LIBS=auth core fake telemetry
COMPOSE_FILE?=deployments/docker/compose/docker-compose.yml
COMPOSE_LOG_FILTER?=appetite-mongodb
COMPOSE_MONGO_USER?=admin
COMPOSE_MONGO_PASS?=password

NOMAD_ADDR?=http://127.0.0.1:4646
NOMAD_JOBS_DIR?=deployments/nomad/jobs
NOMAD_JOBS?=$(NOMAD_JOBS_DIR)/mongodb.nomad $(NOMAD_JOBS_DIR)/appetite-services.nomad
NOMAD_AUTHN_IMAGE?=appetite-authn:latest
NOMAD_AUTHZ_IMAGE?=appetite-authz:latest
NOMAD_DICTIONARY_IMAGE?=appetite-dictionary:latest
NOMAD_ORDER_IMAGE?=appetite-order:latest
NOMAD_TABLE_IMAGE?=appetite-table:latest
NOMAD_ADMIN_IMAGE?=appetite-admin:latest

MONGO_URL?=mongodb://admin:password@localhost:27017/admin?authSource=admin
AUTHN_DB?=appetite_authn
AUTHZ_DB?=appetite_authz
DICTIONARY_DB?=appetite_dictionary
KITCHEN_DB?=appetite_kitchen
MENU_DB?=appetite_menu
ORDER_DB?=appetite_order
TABLE_DB?=appetite_table
TAIL_LINES?=0
FRESH_LOG_LINES?=200
LOG_STREAM?=log-clean

# Go related commands
GOFUMPT=gofumpt
GCI=gci
GOLANGCI_LINT=golangci-lint
GO_TEST=go test
GO_VET=go vet
GO_VULNCHECK=govulncheck

# Phony targets
.PHONY: all build run test test-v test-short coverage coverage-html coverage-func coverage-profile coverage-check coverage-100 clean fmt lint vet check ci run-all stop-all help build-all test-all lint-all db-reset-dev reset-compose-data db-clean-dev fresh-start log-raw log-clean logs logs-clean run-compose run-compose-neat stop-compose nomad-run nomad-stop nomad-status

all: build-all

help:
	@echo "Available targets:"
	@echo "  build-all    - Build all services"
	@echo "  build-utils  - Build appetite-utils CLI tool"
	@echo "  run-all      - Kill ports, build, and start all services"
	@echo "  stop-all     - Stop all running services"
	@echo "  seed-demo    - Seed demo data (orders and kitchen tickets)"
	@echo "  clear-demo   - Clear all demo data"
	@echo "  test         - Run tests for all components"
	@echo "  test-v       - Run tests with verbose output"
	@echo "  test-short   - Run tests in short mode"
	@echo "  test-all     - Run tests for all services and pkg libs"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  coverage-html - Generate HTML coverage report"
	@echo "  coverage-func - Show function-level coverage"
	@echo "  coverage-check - Check coverage meets 80% threshold"
	@echo "  coverage-100 - Check 100% test coverage"
	@echo "  lint         - Run golangci-lint on all code"
	@echo "  lint-all     - Run lint on services and pkg libs"
	@echo "  fmt          - Format all Go code"
	@echo "  vet          - Run go vet on all code"
	@echo "  clean        - Clean all generated files and binaries"
	@echo "  run-compose  - Launch docker compose stack defined in $(COMPOSE_FILE)"
	@echo "  run-compose-neat - Launch compose stack while filtering $(COMPOSE_LOG_FILTER) logs"
	@echo "  stop-compose - Stop the compose stack defined in $(COMPOSE_FILE)"
	@echo "  reset-compose-data - Drop MongoDB databases inside the compose stack"
	@echo "  db-reset-dev - Drop all development databases"
	@echo "  check        - Run all quality checks"
	@echo "  ci           - Run CI pipeline with strict checks"
	@echo "  nomad-run    - Register MongoDB and service jobs in Nomad"
	@echo "  nomad-stop   - Stop and purge Nomad jobs"
	@echo "  nomad-status - Show current job status in Nomad"
	@echo ""
	@echo "Individual service targets (replace <service> with authn/authz/dictionary/menu/order/table/operations/admin/media/kitchen):"
	@echo "  build-<service>  - Build specific service"
	@echo "  test-<service>   - Test specific service"
	@echo "  lint-<service>   - Lint specific service"
	@echo "  run-<service>    - Run specific service"

run-compose:
	@if [ ! -f "$(COMPOSE_FILE)" ]; then \
		echo "❌ docker compose file '$(COMPOSE_FILE)' not found. Override COMPOSE_FILE=path/to/compose.yml"; \
		exit 1; \
	fi
	@echo "Starting docker compose using $(COMPOSE_FILE)..."
	@docker compose -f $(COMPOSE_FILE) up --build

run-compose-neat:
	@if [ ! -f "$(COMPOSE_FILE)" ]; then \
		echo "❌ docker compose file '$(COMPOSE_FILE)' not found. Override COMPOSE_FILE=path/to/compose.yml"; \
		exit 1; \
	fi
	@echo "Starting docker compose using $(COMPOSE_FILE) (filter: $(COMPOSE_LOG_FILTER))..."
	@if [ -z "$(COMPOSE_LOG_FILTER)" ]; then \
		docker compose -f $(COMPOSE_FILE) up --build; \
	else \
		docker compose -f $(COMPOSE_FILE) up --build 2>&1 | grep -v '^$(COMPOSE_LOG_FILTER)'; \
	fi

stop-compose:
	@if [ ! -f "$(COMPOSE_FILE)" ]; then \
		echo "❌ docker compose file '$(COMPOSE_FILE)' not found. Override COMPOSE_FILE=path/to/compose.yml"; \
		exit 1; \
	fi
	@echo "Stopping docker compose using $(COMPOSE_FILE)..."
	@docker compose -f $(COMPOSE_FILE) down

nomad-run:
	@echo "🚀 Registering Nomad jobs using $(NOMAD_JOBS_DIR)..."
	@for job in $(NOMAD_JOBS); do \
		echo "   🗂  Running $$job"; \
		if echo $$job | grep -q "appetite-services"; then \
			env NOMAD_ADDR=$(NOMAD_ADDR) nomad job run \
				-var "authn_image=$(NOMAD_AUTHN_IMAGE)" \
				-var "authz_image=$(NOMAD_AUTHZ_IMAGE)" \
				-var "table_image=$(NOMAD_TABLE_IMAGE)" \
				-var "admin_image=$(NOMAD_ADMIN_IMAGE)" \
				$$job || exit 1; \
		else \
			env NOMAD_ADDR=$(NOMAD_ADDR) nomad job run $$job || exit 1; \
		fi; \
		done
	@echo "✅ Nomad jobs submitted"

nomad-stop:
	@echo "🛑 Stopping Nomad jobs..."
	@for name in appetite-services mongodb; do \
		echo "   🗑  Purging $$name"; \
		env NOMAD_ADDR=$(NOMAD_ADDR) nomad job stop -purge $$name >/dev/null 2>&1 || true; \
	done
	@echo "✅ Nomad jobs stopped"

nomad-status:
	@echo "📊 Nomad job status (NOMAD_ADDR=$(NOMAD_ADDR))"
	@env NOMAD_ADDR=$(NOMAD_ADDR) nomad status appetite-services || true
	@env NOMAD_ADDR=$(NOMAD_ADDR) nomad status mongodb || true

reset-compose-data:
	@if [ ! -f "$(COMPOSE_FILE)" ]; then \
		echo "❌ docker compose file '$(COMPOSE_FILE)' not found. Override COMPOSE_FILE=path/to/compose.yml"; \
		exit 1; \
	fi
	@if ! docker compose -f $(COMPOSE_FILE) ps --status running mongodb >/dev/null 2>&1; then \
		echo "❌ compose MongoDB service is not running. Start it first (make run-compose)."; \
		exit 1; \
	fi
	@echo "🧹 Clearing MongoDB databases inside compose (AuthN=$(AUTHN_DB), AuthZ=$(AUTHZ_DB), Dictionary=$(DICTIONARY_DB), Menu=$(MENU_DB), Order=$(ORDER_DB), Table=$(TABLE_DB), Kitchen=$(KITCHEN_DB))..."
	@docker compose -f $(COMPOSE_FILE) exec mongodb mongosh --quiet --username $(COMPOSE_MONGO_USER) --password $(COMPOSE_MONGO_PASS) --authenticationDatabase admin --eval 'const dbs = ["$(AUTHN_DB)", "$(AUTHZ_DB)", "$(DICTIONARY_DB)", "$(MENU_DB)", "$(ORDER_DB)", "$(TABLE_DB)", "$(KITCHEN_DB)"]; dbs.forEach(name => { const res = db.getSiblingDB(name).dropDatabase(); printjson({db: name, dropped: res.ok === 1}); });'
	@echo "✅ Compose MongoDB databases cleared."

# Build all services
build-all:
	@echo "🏗️  Building all services..."
	@for service in $(SERVICES); do \
		echo "   📦 Building $$service..."; \
		cd services/$$service && go build -o $$service . || exit 1; \
		cd ../..; \
	done
	@echo "✅ All services built successfully"

# Build individual services
build-authn:
	@cd services/authn && go build -o authn . 2>&1 && echo "✅ AuthN build successful"

build-authz:
	@cd services/authz && go build -o authz . 2>&1 && echo "✅ AuthZ build successful"

build-dictionary:
	@cd services/dictionary && go build -o dictionary . 2>&1 && echo "✅ Dictionary build successful"

build-order:
	@cd services/order && go build -o order . 2>&1 && echo "✅ Order build successful"

build-table:
	@cd services/table && go build -o table . 2>&1 && echo "✅ Table build successful"

build-operations:
	@cd services/operations && go build -o operations . 2>&1 && echo "✅ Operations build successful"

build-admin:
	@cd services/admin && go build -o admin . 2>&1 && echo "✅ Admin build successful"

build-media:
	@cd services/media && go build -o media . 2>&1 && echo "✅ Media build successful"

build-menu:
	@cd services/menu && go build -o menu . 2>&1 && echo "✅ Menu build successful"

build-kitchen:
	@cd services/kitchen && go build -o kitchen . 2>&1 && echo "✅ Kitchen build successful"

build-utils:
	@cd cmd/utils && go build -o ../../bin/appetite-utils . 2>&1 && echo "✅ Utils build successful"

log-stream:
	@echo "📜 Streaming raw logs from all services..."
	@tail -n $(TAIL_LINES) -F services/*/*.log 2>/dev/null | \
	awk '{ \
		if ($$0 ~ /^==> .* <==$$/) next; \
		printf "%s %s\n", strftime("[%H:%M:%S]"), $$0; \
	}' || true

log-clean:
	@echo "📜 Streaming condensed logs (time | level | message)..."
	@tail -n $(TAIL_LINES) -F services/*/*.log 2>/dev/null | scripts/log_clean.awk || true

logs: log-stream

log-clear:
	@echo "🧹 Clearing all service logs..."
	@find services -type f -name '*.log' -exec rm -f {} +
	@echo "✅ All logs removed."

fresh-start:
	@echo "♻️  Resetting development environment..."
	@$(MAKE) stop-all
	@$(MAKE) log-clear
	@$(MAKE) db-clean-dev
	@$(MAKE) db-reset-dev
	@$(MAKE) run-all
	@echo "📜 Tailing consolidated logs (last $(FRESH_LOG_LINES) lines)..."
	@TAIL_LINES=$(FRESH_LOG_LINES) $(MAKE) $(LOG_STREAM)

# Test all components
test:
	@echo "🧪 Running tests for all components..."
	@$(GO_TEST) ./...

test-v:
	@echo "🧪 Running tests with verbose output..."
	@$(GO_TEST) -v ./...

test-short:
	@echo "🧪 Running tests in short mode..."
	@$(GO_TEST) -short ./...

# Test all services and pkg libs individually
test-all:
	@echo "🧪 Running tests for all services and pkg libs..."
	@for lib in $(PKG_LIBS); do \
		echo "   🧪 Testing pkg/lib/$$lib..."; \
		cd pkg/lib/$$lib && go test ./... || exit 1; \
		cd ../../..; \
	done
	@for service in $(SERVICES); do \
		echo "   🧪 Testing $$service service..."; \
		cd services/$$service && go test ./... || exit 1; \
		cd ../..; \
	done

# Test individual services
test-authn:
	@cd services/authn && go test ./...

test-authz:
	@cd services/authz && go test ./...

test-dictionary:
	@cd services/dictionary && go test ./...

test-order:
	@cd services/order && go test ./...

test-table:
	@cd services/table && go test ./...

test-operations:
	@cd services/operations && go test ./...

test-admin:
	@cd services/admin && go test ./...

test-menu:
	@cd services/menu && go test ./...

test-kitchen:
	@cd services/kitchen && go test ./...

# Coverage targets
coverage:
	@echo "📊 Running tests with coverage..."
	@$(GO_TEST) -cover ./...

coverage-profile:
	@echo "📊 Generating coverage profile..."
	@$(GO_TEST) -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

coverage-html: coverage-profile
	@echo "📊 Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

coverage-func: coverage-profile
	@echo "📊 Function-level coverage:"
	@go tool cover -func=coverage.out

coverage-check: coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "📊 Coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below 80% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets the 80% threshold"; \
	fi

coverage-100: coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "📊 Coverage: $$COVERAGE%"; \
	if [ "$$COVERAGE" != "100.0" ]; then \
		echo "❌ Coverage $$COVERAGE% is not 100%"; \
		go tool cover -func=coverage.out | grep -v "100.0%"; \
		exit 1; \
	else \
		echo "🎉 Perfect! 100% test coverage!"; \
	fi

# Code quality targets
fmt:
	@echo "🎨 Formatting Go code..."
	@$(GOFUMPT) -l -w .
	@$(GCI) -w .
	@echo "✅ Go code formatted"

vet:
	@echo "🔍 Running go vet..."
	@$(GO_VET) ./...

lint:
	@echo "🔍 Running golangci-lint..."
	@$(GOLANGCI_LINT) run
	@echo "✅ golangci-lint finished"

# Lint all services and pkg libs
lint-all:
	@echo "🔍 Running golangci-lint on all components..."
	@for lib in $(PKG_LIBS); do \
		echo "   🔍 Linting pkg/lib/$$lib..."; \
		cd pkg/lib/$$lib && golangci-lint run || exit 1; \
		cd ../../..; \
	done
	@for service in $(SERVICES); do \
		echo "   🔍 Linting $$service service..."; \
		cd services/$$service && golangci-lint run || exit 1; \
		cd ../..; \
	done

# Individual service lint
lint-authn:
	@cd services/authn && golangci-lint run

lint-authz:
	@cd services/authz && golangci-lint run

lint-dictionary:
	@cd services/dictionary && golangci-lint run

lint-order:
	@cd services/order && golangci-lint run

lint-table:
	@cd services/table && golangci-lint run

lint-operations:
	@cd services/operations && golangci-lint run

lint-admin:
	@cd services/admin && golangci-lint run

lint-menu:
	@cd services/menu && golangci-lint run

lint-kitchen:
	@cd services/kitchen && golangci-lint run

# Quality checks
check: fmt vet test coverage-check lint
	@echo "✅ All quality checks passed!"

ci: fmt vet test coverage-100 lint
	@echo "🚀 CI pipeline passed!"

# Service management
run-all:
	@echo "🚀 Starting full Appetite environment..."
	@$(MAKE) stop-all
	@$(MAKE) build-all
	@echo "🚀 Starting NATS..."
	@$(MAKE) run-nats
	@echo "🚀 Starting services..."
	@echo "   📦 Starting Admin on :8081..."
	@cd services/admin && nohup ./admin > admin.log 2>&1 & echo $$! > admin.pid; sleep 2
	@echo "   📦 Starting AuthN on :8082..."
	@cd services/authn && nohup ./authn > authn.log 2>&1 & echo $$! > authn.pid; sleep 2
	@echo "   📦 Starting AuthZ on :8083..."
	@cd services/authz && nohup ./authz > authz.log 2>&1 & echo $$! > authz.pid; sleep 2
	@echo "   📦 Starting Dictionary on :8085..."
	@cd services/dictionary && nohup ./dictionary > dictionary.log 2>&1 & echo $$! > dictionary.pid; sleep 2
	@echo "   📦 Starting Order on :8086..."
	@cd services/order && nohup ./order > order.log 2>&1 & echo $$! > order.pid; sleep 2
	@echo "   📦 Starting Menu on :8088..."
	@cd services/menu && nohup ./menu > menu.log 2>&1 & echo $$! > menu.pid; sleep 2
	@echo "   📦 Starting Table on :8087..."
	@cd services/table && nohup ./table > table.log 2>&1 & echo $$! > table.pid; sleep 2
	@echo "   📦 Starting Operations on :8080..."
	@cd services/operations && nohup ./operations > operations.log 2>&1 & echo $$! > operations.pid; sleep 2
	@echo "   📦 Starting Media on :8090..."
	@cd services/media && nohup ./media > media.log 2>&1 & echo $$! > media.pid; sleep 2
	@echo "   📦 Starting Kitchen on :8089..."
	@cd services/kitchen && KITCHEN_NATS_STREAM_ENABLED=true nohup ./kitchen > kitchen.log 2>&1 & echo $$! > kitchen.pid; sleep 2
	@echo ""
	@echo "🎉 All Appetite services started!"
	@echo "📡 Services running:"
	@echo "   • Operations: http://localhost:8080 (restaurant operations + chat)"
	@echo "   • Admin:      http://localhost:8081 (business admin)"
	@echo "   • AuthN:      http://localhost:8082 (authentication)"
	@echo "   • AuthZ:      http://localhost:8083 (authorization)"
	@echo "   • Dictionary: http://localhost:8085 (dictionary)"
	@echo "   • Order:      http://localhost:8086 (order management)"
	@echo "   • Table:      http://localhost:8087 (restaurant tables)"
	@echo "   • Menu:       http://localhost:8088 (menu management)"
	@echo "   • Kitchen:    http://localhost:8089 (kitchen tickets)"
	@echo "   • Media:      http://localhost:8090 (media assets)"
	@echo ""
	@echo "🛑 To stop all services: make stop-all"

run-demo:
	@echo "🎭 Starting Appetite environment with DEMO seeding..."
	@$(MAKE) stop-all
	@$(MAKE) db-reset-dev
	@$(MAKE) build-all
	@echo "🚀 Starting NATS..."
	@$(MAKE) run-nats
	@echo "🚀 Starting services with seeding.demo=true..."
	@echo "   📦 Starting Admin on :8081..."
	@cd services/admin && nohup ./admin > admin.log 2>&1 & echo $$! > admin.pid; sleep 2
	@echo "   📦 Starting AuthN on :8082..."
	@cd services/authn && nohup ./authn > authn.log 2>&1 & echo $$! > authn.pid; sleep 2
	@echo "   📦 Starting AuthZ on :8083..."
	@cd services/authz && nohup ./authz > authz.log 2>&1 & echo $$! > authz.pid; sleep 2
	@echo "   📦 Starting Dictionary on :8085..."
	@cd services/dictionary && nohup ./dictionary > dictionary.log 2>&1 & echo $$! > dictionary.pid; sleep 2
	@echo "   📦 Starting Menu on :8088..."
	@cd services/menu && nohup ./menu > menu.log 2>&1 & echo $$! > menu.pid; sleep 2
	@echo "   📦 Starting Table on :8087 (with demo seeding)..."
	@cd services/table && TABLE_SEEDING_DEMO=true nohup ./table > table.log 2>&1 & echo $$! > table.pid; sleep 3
	@echo "   📦 Starting Order on :8086 (with demo seeding)..."
	@cd services/order && ORDER_SEEDING_DEMO=true nohup ./order > order.log 2>&1 & echo $$! > order.pid; sleep 3
	@echo "   📦 Starting Kitchen on :8089 (with demo seeding)..."
	@cd services/kitchen && KITCHEN_SEEDING_DEMO=true KITCHEN_NATS_STREAM_ENABLED=true nohup ./kitchen > kitchen.log 2>&1 & echo $$! > kitchen.pid; sleep 3
	@echo "   📦 Starting Operations on :8080..."
	@cd services/operations && nohup ./operations > operations.log 2>&1 & echo $$! > operations.pid; sleep 2
	@echo "   📦 Starting Media on :8090..."
	@cd services/media && nohup ./media > media.log 2>&1 & echo $$! > media.pid; sleep 2
	@echo ""
	@echo "🎉 All Appetite services started with DEMO data!"
	@echo "📡 Services running:"
	@echo "   • Operations: http://localhost:8080 (restaurant operations + chat)"
	@echo "   • Admin:      http://localhost:8081 (business admin)"
	@echo "   • AuthN:      http://localhost:8082 (authentication)"
	@echo "   • AuthZ:      http://localhost:8083 (authorization)"
	@echo "   • Dictionary: http://localhost:8085 (dictionary)"
	@echo "   • Order:      http://localhost:8086 (order management)"
	@echo "   • Table:      http://localhost:8087 (restaurant tables)"
	@echo "   • Menu:       http://localhost:8088 (menu management)"
	@echo "   • Kitchen:    http://localhost:8089 (kitchen tickets)"
	@echo "   • Media:      http://localhost:8090 (media assets)"
	@echo ""
	@echo "🛑 To stop all services: make stop-all"

# Individual service runners
run-authn: build-authn
	@cd services/authn && ./authn

run-authz: build-authz
	@cd services/authz && ./authz

run-dictionary: build-dictionary
	@cd services/dictionary && ./dictionary

run-order: build-order
	@cd services/order && ./order

run-table: build-table
	@cd services/table && ./table

run-operations: build-operations
	@cd services/operations && ./operations

run-admin: build-admin
	@cd services/admin && ./admin

run-media: build-media
	@cd services/media && ./media

run-menu: build-menu
	@cd services/menu && ./menu

run-kitchen: build-kitchen
	@cd services/kitchen && ./kitchen

stop-all:
	@echo "🛑 Stopping all Appetite services..."
	@for port in $(BASE_PORTS); do \
		if lsof -ti:$$port >/dev/null 2>&1; then \
			echo "🛑 Stopping process on port $$port"; \
			lsof -ti:$$port | xargs -r kill -9 || true; \
		fi; \
	done
	@for service in $(SERVICES); do \
		pid_file="services/$$service/$$service.pid"; \
		if [ -f "$$pid_file" ]; then \
			pid=$$(cat "$$pid_file"); \
			if ps -p "$$pid" >/dev/null 2>&1; then \
				echo "🛑 Stopping $$service (PID: $$pid)"; \
				kill -9 "$$pid" 2>/dev/null || true; \
			fi; \
			rm -f "$$pid_file"; \
		fi; \
	done
	@$(MAKE) stop-nats
	@echo "✅ All Appetite services stopped"

# NATS targets
run-nats:
	@./scripts/util/nats-start.sh

stop-nats:
	@./scripts/util/nats-stop.sh

install-nats-docker:
	@echo "📦 Pulling NATS Docker image..."
	@docker pull nats:2.10-alpine
	@echo "✅ NATS Docker image ready"

# Clean targets
clean:
	@echo "🧹 Cleaning up..."
	@for service in $(SERVICES); do \
		cd services/$$service && rm -f $$service *.log *.pid; \
		cd ../..; \
	done
	@go clean -testcache
	@rm -f coverage.out coverage.html
	@echo "✅ Cleanup complete"

# Security check (if govulncheck is available)
security:
	@echo "🔒 Running security checks..."
	@$(GO_VULNCHECK) ./... || echo "⚠️  govulncheck not available, install with: go install golang.org/x/vuln/cmd/govulncheck@latest"

# Development helpers
dev-deps:
	@echo "📥 Installing development dependencies..."
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/daixiang0/gci@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "✅ Development dependencies installed"

# Reset MongoDB collections for local/Nomad development (host network)
db-reset-dev:
	@command -v mongosh >/dev/null 2>&1 || { echo "❌ mongosh not found. Install MongoDB Shell."; exit 1; }
	@echo "🧹 FULL RESET - Dropping ALL development databases..."
	@echo "🧹 Dropping AuthN database ($(AUTHN_DB))..."
	@mongosh "$(MONGO_URL)" --quiet --eval 'db = db.getSiblingDB("$(AUTHN_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, dropped: "$(AUTHN_DB)" });'
	@if [ "$(AUTHN_DB)" != "auth" ]; then \
		echo "🧹 Also dropping legacy AuthN database (auth)..."; \
		mongosh "$(MONGO_URL)" --quiet --eval 'db = db.getSiblingDB("auth"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, dropped: "auth" });'; \
	fi
	@echo "🧹 Dropping AuthZ database ($(AUTHZ_DB))..."
	@mongosh "$(MONGO_URL)" --quiet --eval 'db = db.getSiblingDB("$(AUTHZ_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, dropped: "$(AUTHZ_DB)" });'
	@echo "🧹 Dropping Dictionary database ($(DICTIONARY_DB))..."
	@mongosh "$(MONGO_URL)" --quiet --eval 'db = db.getSiblingDB("$(DICTIONARY_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, dropped: "$(DICTIONARY_DB)" });'
	@echo "🧹 Dropping Menu database ($(MENU_DB))..."
	@mongosh "$(MONGO_URL)" --quiet --eval 'db = db.getSiblingDB("$(MENU_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, dropped: "$(MENU_DB)" });'
	@echo "🧹 Dropping Order database ($(ORDER_DB))..."
	@mongosh "$(MONGO_URL)" --quiet --eval 'db = db.getSiblingDB("$(ORDER_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, dropped: "$(ORDER_DB)" });'
	@echo "🧹 Dropping Table database ($(TABLE_DB))..."
	@mongosh "$(MONGO_URL)" --quiet --eval 'db = db.getSiblingDB("$(TABLE_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, dropped: "$(TABLE_DB)" });'
	@echo "🧹 Dropping Kitchen database ($(KITCHEN_DB))..."
	@mongosh "$(MONGO_URL)" --quiet --eval 'db = db.getSiblingDB("$(KITCHEN_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, dropped: "$(KITCHEN_DB)" });'
	@echo "✅ All development databases dropped!"

# Reset MongoDB collections for Docker Compose (container network)
db-reset-compose:
	@echo "🧹 Clearing Docker Compose MongoDB collections..."
	@docker exec appetite-mongodb mongosh "mongodb://$(COMPOSE_MONGO_USER):$(COMPOSE_MONGO_PASS)@localhost:27017/admin?authSource=admin" --quiet --eval 'db = db.getSiblingDB("$(AUTHN_DB)"); result = db.users.deleteMany({}); printjson(result);' || echo "⚠️  AuthN collection clear failed"
	@docker exec appetite-mongodb mongosh "mongodb://$(COMPOSE_MONGO_USER):$(COMPOSE_MONGO_PASS)@localhost:27017/admin?authSource=admin" --quiet --eval 'db = db.getSiblingDB("$(AUTHZ_DB)"); result = db.roles.deleteMany({}); printjson(result);' || echo "⚠️  AuthZ roles clear failed"
	@docker exec appetite-mongodb mongosh "mongodb://$(COMPOSE_MONGO_USER):$(COMPOSE_MONGO_PASS)@localhost:27017/admin?authSource=admin" --quiet --eval 'db = db.getSiblingDB("$(AUTHZ_DB)"); result = db.grants.deleteMany({}); printjson(result);' || echo "⚠️  AuthZ grants clear failed"
	@docker exec appetite-mongodb mongosh "mongodb://$(COMPOSE_MONGO_USER):$(COMPOSE_MONGO_PASS)@localhost:27017/admin?authSource=admin" --quiet --eval 'db = db.getSiblingDB("$(AUTHZ_DB)"); result = db._seeds.deleteMany({}); printjson(result);' || echo "⚠️  AuthZ seed tracker clear failed"
	@docker exec appetite-mongodb mongosh "mongodb://$(COMPOSE_MONGO_USER):$(COMPOSE_MONGO_PASS)@localhost:27017/admin?authSource=admin" --quiet --eval 'db = db.getSiblingDB("$(DICTIONARY_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, droppedDatabase: result.dropped });' || echo "⚠️  Dictionary database drop failed"
	@docker exec appetite-mongodb mongosh "mongodb://$(COMPOSE_MONGO_USER):$(COMPOSE_MONGO_PASS)@localhost:27017/admin?authSource=admin" --quiet --eval 'db = db.getSiblingDB("$(ORDER_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, droppedDatabase: result.dropped });' || echo "⚠️  Order database drop failed"
	@docker exec appetite-mongodb mongosh "mongodb://$(COMPOSE_MONGO_USER):$(COMPOSE_MONGO_PASS)@localhost:27017/admin?authSource=admin" --quiet --eval 'db = db.getSiblingDB("$(TABLE_DB)"); result = db.dropDatabase(); printjson({ acknowledged: result.ok === 1, droppedDatabase: result.dropped });' || echo "⚠️  Table database drop failed"
	@echo "✅ Docker Compose MongoDB collections cleared."

# Tidy all modules
tidy:
	@echo "🧹 Running go mod tidy on all modules..."
	@for lib in $(PKG_LIBS); do \
		echo "   📦 Tidying pkg/lib/$$lib..."; \
		cd pkg/lib/$$lib && go mod tidy; \
		cd ../../..; \
	done
	@for service in $(SERVICES); do \
		echo "   📦 Tidying $$service service..."; \
		cd services/$$service && go mod tidy; \
		cd ../..; \
	done
	@echo "✅ All modules tidied"

# Demo data seeding (new approach using utils CLI)
seed-demo: build-utils
	@echo "🎬 Seeding demo orders and kitchen tickets..."
	@./bin/appetite-utils seed-demo
	@echo "🔄 Reloading kitchen cache..."
	@curl -s -X POST http://localhost:8089/internal/reload-cache > /dev/null 2>&1 || echo "⚠️  Kitchen service not running or cache reload failed"
	@echo ""
	@echo "✅ Demo data seeded! Sample test data now available."
	@echo "📋 Check /orders and /kitchen for demo content"
	@echo "💡 Tip: Run 'make clear-demo' to remove demo data or 'make db-reset-dev' to reset everything"

clear-demo: build-utils
	@echo "🧹 Clearing demo data..."
	@./bin/appetite-utils clear-demo

# Legacy demo seeding (kept for reference, will be removed in future)
seed-demo-legacy:
	@echo "⚠️  This is the legacy seeding approach. Use 'make seed-demo' instead."
	@echo "🎬 Seeding demo orders and kitchen tickets..."
	@command -v mongosh >/dev/null 2>&1 || { echo "❌ mongosh not found. Install MongoDB Shell."; exit 1; }
	@echo "   🗑️  Clearing previous demo seed tracking..."
	@mongosh "$(MONGO_URL)" --quiet --eval 'db.getSiblingDB("appetite_order")._seeds.deleteOne({_id: "demo_orders_v1"});' >/dev/null 2>&1 || true
	@mongosh "$(MONGO_URL)" --quiet --eval 'db.getSiblingDB("appetite_kitchen")._seeds.deleteOne({_id: "demo_tickets_v1"});' >/dev/null 2>&1 || true
	@echo "   🚀 Restarting services with seed.demo.enabled=true..."
	@$(MAKE) stop-all >/dev/null 2>&1 || true
	@SEED_DEMO_ENABLED=true $(MAKE) run-all
	@echo ""
	@echo "✅ Demo data seeded! Sample test data now available."
	@echo "📋 Check /orders and /kitchen for demo content"
	@echo "💡 Tip: Run 'make db-reset-dev' to clear all data and start fresh"
