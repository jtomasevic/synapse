# ============================================================
# Synapse Service - Makefile
# ============================================================
# Targets:
#   make test-service          - Run service integration tests (auto-starts Docker DB)
#   make test-service-coverage - Same as above, with HTML coverage report
#   make db-run                - Start the synapse-db container (delegates to storage)
#   make db-stop               - Stop the synapse-db container  (delegates to storage)
#   make db-status             - Show container status           (delegates to storage)
# ============================================================

STORAGE_DIR  := ../storage
HOST_PORT    := 5432
CONTAINER    := synapse-db
DB_DSN       := postgres://synapse:synapse@localhost:$(HOST_PORT)/synapse?sslmode=disable
DB_READY_TIMEOUT := 15

COVERAGE_DIR := coverage
COVERAGE_OUT := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

.PHONY: test-service test-service-coverage db-run db-stop db-status

# ----------------------------------------------------------
# db-run / db-stop / db-status: delegate to storage/Makefile
# ----------------------------------------------------------
db-run:
	$(MAKE) -C $(STORAGE_DIR) run

db-stop:
	$(MAKE) -C $(STORAGE_DIR) stop

db-status:
	$(MAKE) -C $(STORAGE_DIR) status

# ----------------------------------------------------------
# test-service: Run service integration tests against real DB
#   1. Ensures the Docker image is built & container is running
#      (delegated to storage/Makefile's "run" target)
#   2. Waits for PostgreSQL to accept connections
#   3. Runs the integration test suite
# ----------------------------------------------------------
test-service: db-run
	@echo "Waiting for PostgreSQL to be ready..."
	@ready=0; \
	for i in $$(seq 1 $(DB_READY_TIMEOUT)); do \
		if docker exec $(CONTAINER) pg_isready -U synapse -d synapse >/dev/null 2>&1; then \
			ready=1; \
			break; \
		fi; \
		echo "  Waiting... ($$i/$(DB_READY_TIMEOUT))"; \
		sleep 1; \
	done; \
	if [ "$$ready" -ne 1 ]; then \
		echo "ERROR: PostgreSQL not ready after $(DB_READY_TIMEOUT)s"; \
		exit 1; \
	fi
	@echo "PostgreSQL is ready. Running service integration tests..."
	cd ../.. && SYNAPSE_TEST_DSN="$(DB_DSN)" go test -v -count=1 ./pkg/service/integration_test/...
	@echo ""
	@echo "Service integration tests passed."

# ----------------------------------------------------------
# test-service-coverage: Same as test-service but with
#   coverage profile + HTML report
# ----------------------------------------------------------
test-service-coverage: db-run
	@echo "Waiting for PostgreSQL to be ready..."
	@ready=0; \
	for i in $$(seq 1 $(DB_READY_TIMEOUT)); do \
		if docker exec $(CONTAINER) pg_isready -U synapse -d synapse >/dev/null 2>&1; then \
			ready=1; \
			break; \
		fi; \
		echo "  Waiting... ($$i/$(DB_READY_TIMEOUT))"; \
		sleep 1; \
	done; \
	if [ "$$ready" -ne 1 ]; then \
		echo "ERROR: PostgreSQL not ready after $(DB_READY_TIMEOUT)s"; \
		exit 1; \
	fi
	@mkdir -p $(COVERAGE_DIR)
	@echo "PostgreSQL is ready. Running service tests with coverage..."
	cd ../.. && SYNAPSE_TEST_DSN="$(DB_DSN)" go test -v -count=1 \
		-coverprofile=pkg/service/$(COVERAGE_OUT) \
		-coverpkg=./pkg/service/...,./pkg/service/models/... \
		./pkg/service/... ./pkg/service/integration_test/...
	@echo ""
	cd ../.. && go tool cover -func=pkg/service/$(COVERAGE_OUT)
	@echo ""
	cd ../.. && go tool cover -html=pkg/service/$(COVERAGE_OUT) -o pkg/service/$(COVERAGE_HTML)
	@echo "Coverage report: pkg/service/$(COVERAGE_HTML)"
	@echo "Open with: open pkg/service/$(COVERAGE_HTML)"
