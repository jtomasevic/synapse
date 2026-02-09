# ============================================================
# Synapse Storage - Makefile
# ============================================================
# Docker targets:
#   make build   - Build the synapse-db Docker image
#   make run     - Start the synapse-db container (builds if needed)
#   make stop    - Stop and remove the synapse-db container
#   make status  - Show container status
#   make logs    - Tail container logs
#   make psql    - Open a psql shell to the synapse database
#   make clean   - Stop container and remove the Docker image
#   make purge   - Remove everything including data volume
#
# sqlc targets:
#   make sqlc-install  - Install sqlc if not already installed
#   make sqlc-generate - Generate Go repository code from SQL queries
#   make sqlc-vet      - Verify sqlc queries against the schema
#   make sqlc-diff     - Show pending sqlc changes (dry run)
#
# Integration tests:
#   make test-loader      - Run loader integration tests (auto-starts Docker)
#   make test-pg-network  - Run PG network integration tests (auto-starts Docker)
#   make test-integration - Run ALL integration tests
# ============================================================

IMAGE_NAME   := synapse-db
CONTAINER    := synapse-db
DOCKER_DIR   := docker
HOST_PORT    := 5432
VOLUME_NAME  := synapse-pgdata

SQLC_DIR     := repository
SQLC_VERSION := latest

.PHONY: build run stop status logs psql clean purge sqlc-install sqlc-generate sqlc-vet sqlc-diff test-loader test-pg-network test-integration

# ----------------------------------------------------------
# build: Build the Docker image
# ----------------------------------------------------------
build:
	@echo "Building Docker image '$(IMAGE_NAME)'..."
	docker build -t $(IMAGE_NAME) $(DOCKER_DIR)
	@echo "Image '$(IMAGE_NAME)' built successfully."

# ----------------------------------------------------------
# run: Start the container
#   - Builds image if it doesn't exist
#   - Skips if container is already running
# ----------------------------------------------------------
run:
	@if docker ps --format '{{.Names}}' | grep -q '^$(CONTAINER)$$'; then \
		echo "Synapse DB is already running."; \
	else \
		if ! docker images -q $(IMAGE_NAME) 2>/dev/null | grep -q .; then \
			echo "Image '$(IMAGE_NAME)' not found. Building..."; \
			$(MAKE) build; \
		fi; \
		if docker ps -a --format '{{.Names}}' | grep -q '^$(CONTAINER)$$'; then \
			echo "Starting existing container '$(CONTAINER)'..."; \
			docker start $(CONTAINER); \
		else \
			echo "Creating and starting container '$(CONTAINER)'..."; \
			docker run -d \
				--name $(CONTAINER) \
				-p $(HOST_PORT):5432 \
				-v $(VOLUME_NAME):/var/lib/postgresql/data \
				$(IMAGE_NAME); \
		fi; \
		echo "Synapse DB is now running on port $(HOST_PORT)."; \
		echo "  Host: localhost:$(HOST_PORT)"; \
		echo "  Database: synapse"; \
		echo "  User: synapse"; \
		echo "  Password: synapse"; \
	fi

# ----------------------------------------------------------
# stop: Stop and remove the container (data persists in volume)
# ----------------------------------------------------------
stop:
	@if docker ps --format '{{.Names}}' | grep -q '^$(CONTAINER)$$'; then \
		echo "Stopping container '$(CONTAINER)'..."; \
		docker stop $(CONTAINER); \
		echo "Container stopped."; \
	else \
		echo "Container '$(CONTAINER)' is not running."; \
	fi

# ----------------------------------------------------------
# status: Show whether the container is running
# ----------------------------------------------------------
status:
	@if docker ps --format '{{.Names}}' | grep -q '^$(CONTAINER)$$'; then \
		echo "Synapse DB is RUNNING."; \
		docker ps --filter name=^$(CONTAINER)$$ --format "table {{.ID}}\t{{.Status}}\t{{.Ports}}"; \
	else \
		echo "Synapse DB is NOT running."; \
	fi

# ----------------------------------------------------------
# logs: Tail the container logs
# ----------------------------------------------------------
logs:
	docker logs -f $(CONTAINER)

# ----------------------------------------------------------
# psql: Open a psql shell into the synapse database
# ----------------------------------------------------------
psql:
	docker exec -it $(CONTAINER) psql -U synapse -d synapse

# ----------------------------------------------------------
# clean: Stop container, remove container and image
#        NOTE: Volume (data) is preserved. Use 'make purge' to remove it.
# ----------------------------------------------------------
clean:
	@if docker ps --format '{{.Names}}' | grep -q '^$(CONTAINER)$$'; then \
		docker stop $(CONTAINER); \
	fi
	@if docker ps -a --format '{{.Names}}' | grep -q '^$(CONTAINER)$$'; then \
		docker rm $(CONTAINER); \
	fi
	@if docker images -q $(IMAGE_NAME) 2>/dev/null | grep -q .; then \
		docker rmi $(IMAGE_NAME); \
	fi
	@echo "Cleaned up container and image."

# ----------------------------------------------------------
# purge: Remove everything including persistent data volume
# ----------------------------------------------------------
purge: clean
	@if docker volume ls -q | grep -q '^$(VOLUME_NAME)$$'; then \
		docker volume rm $(VOLUME_NAME); \
		echo "Volume '$(VOLUME_NAME)' removed."; \
	fi
	@echo "Full purge complete."

# ============================================================
# Integration tests
# ============================================================

DB_DSN := postgres://synapse:synapse@localhost:$(HOST_PORT)/synapse?sslmode=disable
DB_READY_TIMEOUT := 15

# ----------------------------------------------------------
# test-loader: Run loader integration tests against real DB
#   - Builds the Docker image if it doesn't exist
#   - Starts the container if it's not running
#   - Waits for PostgreSQL to accept connections
#   - Runs the test suite
# ----------------------------------------------------------
test-loader: run
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
	@echo "PostgreSQL is ready. Running loader integration tests..."
	cd .. && SYNAPSE_TEST_DSN="$(DB_DSN)" go test -v -count=1 ./storage/repository/loader/...
	@echo ""
	@echo "Loader integration tests passed."

# ----------------------------------------------------------
# test-pg-network: Run EventNetworkOnPG integration tests
#   - Same Docker lifecycle as test-loader
#   - Exercises PG-backed network, graph traversals, hydration,
#     MemoizedNetwork, CachedRelationProvider, StructuralMemory
# ----------------------------------------------------------
test-pg-network: run
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
	@echo "PostgreSQL is ready. Running PG network integration tests..."
	cd .. && SYNAPSE_TEST_DSN="$(DB_DSN)" go test -v -count=1 ./pkg/event_network/integration_tests/...
	@echo ""
	@echo "PG network integration tests passed."

# ----------------------------------------------------------
# test-integration: Run ALL integration tests
# ----------------------------------------------------------
test-integration: test-loader test-pg-network
	@echo ""
	@echo "All integration tests passed."git 

# ============================================================
# sqlc - Code generation from SQL queries
# ============================================================

# ----------------------------------------------------------
# ensure-sqlc: Install sqlc if not present (used by other targets)
# ----------------------------------------------------------
define ensure_sqlc
	@if ! command -v sqlc >/dev/null 2>&1; then \
		echo "sqlc not found. Installing via go install..."; \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION); \
		if ! command -v sqlc >/dev/null 2>&1; then \
			echo ""; \
			echo "sqlc was installed but is not on your PATH."; \
			echo "Add Go bin to PATH:  export PATH=\"\$$(go env GOPATH)/bin:\$$PATH\""; \
			echo "Or install via Homebrew:  brew install sqlc"; \
			exit 1; \
		fi; \
		echo "sqlc installed successfully: $$(sqlc version)"; \
	fi
endef

# ----------------------------------------------------------
# sqlc-install: Explicitly install sqlc
# ----------------------------------------------------------
sqlc-install:
	@if command -v sqlc >/dev/null 2>&1; then \
		echo "sqlc is already installed: $$(sqlc version)"; \
	else \
		echo "Installing sqlc via go install..."; \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION); \
		echo "Done. Make sure \$$(go env GOPATH)/bin is on your PATH."; \
	fi

# ----------------------------------------------------------
# sqlc-generate: Generate Go code from SQL queries
# ----------------------------------------------------------
sqlc-generate:
	$(ensure_sqlc)
	cd $(SQLC_DIR) && sqlc generate
	@echo "sqlc code generated in $(SQLC_DIR)/."

# ----------------------------------------------------------
# sqlc-vet: Validate queries against the schema
# ----------------------------------------------------------
sqlc-vet:
	$(ensure_sqlc)
	cd $(SQLC_DIR) && sqlc vet
	@echo "sqlc vet passed."

# ----------------------------------------------------------
# sqlc-diff: Show what sqlc generate would change (dry run)
# ----------------------------------------------------------
sqlc-diff:
	$(ensure_sqlc)
	cd $(SQLC_DIR) && sqlc diff
	@echo "sqlc diff complete."
