# ============================================================
# Synapse pkg - Unified Makefile
# ============================================================
#
# Unit tests (no Docker required):
#   make ut-en              - Unit tests: event_network
#   make ut-en-cov          - Unit tests: event_network (with coverage)
#   make ut-service         - Unit tests: service + service/models
#   make ut-service-cov     - Unit tests: service + service/models (with coverage)
#   make ut-storage         - Unit tests: storage (loader, skips if DB unavailable)
#   make ut-storage-cov     - Unit tests: storage (with coverage)
#   make test               - ALL unit tests
#   make test-cov           - ALL unit tests (with merged coverage)
#
# Integration tests (auto-starts Docker DB):
#   make it-en              - Integration tests: event_network (PG network)
#   make it-en-cov          - Integration tests: event_network (with coverage)
#   make it-service         - Integration tests: service
#   make it-service-cov     - Integration tests: service (with coverage)
#   make it-storage         - Integration tests: storage (loader)
#   make it-storage-cov     - Integration tests: storage (with coverage)
#   make test-integration   - ALL integration tests
#   make test-integration-cov - ALL integration tests (with merged coverage)
#
# Docker (delegates to storage/Makefile):
#   make db-run             - Start the synapse-db container
#   make db-stop            - Stop the synapse-db container
#   make db-status          - Show container status
# ============================================================

MODULE_ROOT := ..
STORAGE_DIR := storage
SERVICE_DIR := service

COVERAGE_DIR := coverage
COVERAGE_UT  := $(COVERAGE_DIR)/ut
COVERAGE_IT  := $(COVERAGE_DIR)/it

.PHONY: ut-en ut-en-cov ut-service ut-service-cov ut-storage ut-storage-cov \
        test test-cov \
        it-en it-en-cov it-service it-service-cov it-storage it-storage-cov \
        test-integration test-integration-cov \
        db-run db-stop db-status

# ============================================================
# Docker helpers (delegate to storage/Makefile)
# ============================================================

db-run:
	$(MAKE) -C $(STORAGE_DIR) run

db-stop:
	$(MAKE) -C $(STORAGE_DIR) stop

db-status:
	$(MAKE) -C $(STORAGE_DIR) status

# ============================================================
# Unit tests
# ============================================================

# ----------------------------------------------------------
# ut-en: Unit tests for event_network
# ----------------------------------------------------------
ut-en:
	@echo "=== Unit tests: event_network ==="
	cd $(MODULE_ROOT) && go test -v -count=1 ./pkg/event_network/...
	@echo ""
	@echo "event_network unit tests passed."

# ----------------------------------------------------------
# ut-en-cov: Unit tests for event_network with coverage
# ----------------------------------------------------------
ut-en-cov:
	@echo "=== Unit tests: event_network (coverage) ==="
	@mkdir -p $(COVERAGE_UT)
	cd $(MODULE_ROOT) && go test -v -count=1 \
		-coverprofile=pkg/$(COVERAGE_UT)/en.out \
		-coverpkg=./pkg/event_network/... \
		./pkg/event_network/...
	@echo ""
	cd $(MODULE_ROOT) && go tool cover -func=pkg/$(COVERAGE_UT)/en.out
	@echo ""
	cd $(MODULE_ROOT) && go tool cover -html=pkg/$(COVERAGE_UT)/en.out -o pkg/$(COVERAGE_UT)/en.html
	@echo "Coverage report: pkg/$(COVERAGE_UT)/en.html"

# ----------------------------------------------------------
# ut-service: Unit tests for service + models
# ----------------------------------------------------------
ut-service:
	@echo "=== Unit tests: service ==="
	cd $(MODULE_ROOT) && go test -v -count=1 ./pkg/service/... ./pkg/service/models/...
	@echo ""
	@echo "service unit tests passed."

# ----------------------------------------------------------
# ut-service-cov: Unit tests for service + models with coverage
# ----------------------------------------------------------
ut-service-cov:
	@echo "=== Unit tests: service (coverage) ==="
	@mkdir -p $(COVERAGE_UT)
	cd $(MODULE_ROOT) && go test -v -count=1 \
		-coverprofile=pkg/$(COVERAGE_UT)/service.out \
		-coverpkg=./pkg/service/...,./pkg/service/models/... \
		./pkg/service/... ./pkg/service/models/...
	@echo ""
	cd $(MODULE_ROOT) && go tool cover -func=pkg/$(COVERAGE_UT)/service.out
	@echo ""
	cd $(MODULE_ROOT) && go tool cover -html=pkg/$(COVERAGE_UT)/service.out -o pkg/$(COVERAGE_UT)/service.html
	@echo "Coverage report: pkg/$(COVERAGE_UT)/service.html"

# ----------------------------------------------------------
# ut-storage: Unit tests for storage (loader)
#   NOTE: loader tests need a real DB; they skip gracefully
#   when SYNAPSE_TEST_DSN is not set.
# ----------------------------------------------------------
ut-storage:
	@echo "=== Unit tests: storage ==="
	cd $(MODULE_ROOT) && go test -v -count=1 ./pkg/storage/...
	@echo ""
	@echo "storage unit tests passed."

# ----------------------------------------------------------
# ut-storage-cov: Unit tests for storage with coverage
# ----------------------------------------------------------
ut-storage-cov:
	@echo "=== Unit tests: storage (coverage) ==="
	@mkdir -p $(COVERAGE_UT)
	cd $(MODULE_ROOT) && go test -v -count=1 \
		-coverprofile=pkg/$(COVERAGE_UT)/storage.out \
		-coverpkg=./pkg/storage/... \
		./pkg/storage/...
	@echo ""
	cd $(MODULE_ROOT) && go tool cover -func=pkg/$(COVERAGE_UT)/storage.out
	@echo ""
	cd $(MODULE_ROOT) && go tool cover -html=pkg/$(COVERAGE_UT)/storage.out -o pkg/$(COVERAGE_UT)/storage.html
	@echo "Coverage report: pkg/$(COVERAGE_UT)/storage.html"

# ----------------------------------------------------------
# test: Run ALL unit tests
# ----------------------------------------------------------
test: ut-en ut-service ut-storage
	@echo ""
	@echo "========================================="
	@echo "  All unit tests passed."
	@echo "========================================="

# ----------------------------------------------------------
# test-cov: Run ALL unit tests with merged coverage
# ----------------------------------------------------------
test-cov:
	@echo "=== All unit tests (coverage) ==="
	@mkdir -p $(COVERAGE_UT)
	cd $(MODULE_ROOT) && go test -v -count=1 \
		-coverprofile=pkg/$(COVERAGE_UT)/all.out \
		-coverpkg=./pkg/event_network/...,./pkg/service/...,./pkg/service/models/...,./pkg/storage/... \
		./pkg/event_network/... ./pkg/service/... ./pkg/service/models/... ./pkg/storage/...
	@echo ""
	cd $(MODULE_ROOT) && go tool cover -func=pkg/$(COVERAGE_UT)/all.out
	@echo ""
	cd $(MODULE_ROOT) && go tool cover -html=pkg/$(COVERAGE_UT)/all.out -o pkg/$(COVERAGE_UT)/all.html
	@echo ""
	@echo "========================================="
	@echo "  All unit tests passed (with coverage)."
	@echo "  Report: pkg/$(COVERAGE_UT)/all.html"
	@echo "========================================="

# ============================================================
# Integration tests (require Docker DB)
# ============================================================

# ----------------------------------------------------------
# it-en: Integration tests for event_network (PG network)
# ----------------------------------------------------------
it-en:
	$(MAKE) -C $(STORAGE_DIR) test-pg-network

# ----------------------------------------------------------
# it-en-cov: Integration tests for event_network with coverage
# ----------------------------------------------------------
it-en-cov:
	$(MAKE) -C $(STORAGE_DIR) test-pg-network-coverage

# ----------------------------------------------------------
# it-service: Integration tests for service
# ----------------------------------------------------------
it-service:
	$(MAKE) -C $(SERVICE_DIR) test-service

# ----------------------------------------------------------
# it-service-cov: Integration tests for service with coverage
# ----------------------------------------------------------
it-service-cov:
	$(MAKE) -C $(SERVICE_DIR) test-service-coverage

# ----------------------------------------------------------
# it-storage: Integration tests for storage (loader)
# ----------------------------------------------------------
it-storage:
	$(MAKE) -C $(STORAGE_DIR) test-loader

# ----------------------------------------------------------
# it-storage-cov: Integration tests for storage with coverage
# ----------------------------------------------------------
it-storage-cov:
	$(MAKE) -C $(STORAGE_DIR) test-loader-coverage

# ----------------------------------------------------------
# test-integration: Run ALL integration tests
# ----------------------------------------------------------
test-integration: it-storage it-en it-service
	@echo ""
	@echo "========================================="
	@echo "  All integration tests passed."
	@echo "========================================="

# ----------------------------------------------------------
# test-integration-cov: Run ALL integration tests with coverage
#   Runs each suite individually (each needs its own DB session
#   and coverage profile), then merges into a single report.
# ----------------------------------------------------------
test-integration-cov: it-storage-cov it-en-cov it-service-cov
	@echo ""
	@echo "========================================="
	@echo "  All integration tests passed (with coverage)."
	@echo "  Individual reports:"
	@echo "    storage:       pkg/storage/$(COVERAGE_DIR)/coverage.html"
	@echo "    event_network: pkg/storage/$(COVERAGE_DIR)/en/coverage.html"
	@echo "    service:       pkg/service/$(COVERAGE_DIR)/coverage.html"
	@echo "========================================="
