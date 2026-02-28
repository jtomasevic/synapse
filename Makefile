# ============================================================
# Synapse - Root Makefile
# ============================================================
#
# Docker Compose (containerised DB + API):
#   make up        - Build images (if needed) and start both containers
#   make down      - Stop and remove containers
#   make up-build  - Force-rebuild images and start
#   make compose-status - Show container status
#   make compose-logs   - Tail logs from all services
#
# Local development (API runs on host, DB in Docker):
#   make run       - Build DB (if needed), start DB, build & start API locally
#   make stop      - Stop local API server and DB container
#
# Individual targets (local dev):
#   make db        - Ensure DB Docker image is built and container is running
#   make db-stop   - Stop the DB container
#   make db-status - Show DB container status
#   make api       - Build the API server binary
#   make api-start - Start the API server locally (ensures DB is up first)
#   make api-stop  - Stop the local API server
#   make api-status- Show whether the local API server is running
#
# Configuration:
#   config.yaml        - Local development settings (DB on localhost)
#   config.docker.yaml - Docker Compose settings (DB on service name "db")
#
# Swagger UI:
#   After 'make up' or 'make run', open http://localhost:8080/docs
# ============================================================

API_BINARY   := synapse-api
API_PID_FILE := .synapse-api.pid
API_LOG_FILE := .synapse-api.log
CONFIG_FILE  := config.yaml

STORAGE_DIR  := pkg/storage

COMPOSE      := docker compose

.PHONY: up down up-build compose-status compose-logs \
        run stop db db-stop db-status db-wait \
        api api-start api-stop api-status clean

# =============================================================
# Docker Compose targets
# =============================================================

# ----------------------------------------------------------
# up: Ensure images exist, start DB + API in containers
#   1. Check if each image exists; build only what's missing
#   2. Skip if containers already running
#   3. Start via docker compose
# ----------------------------------------------------------
up:
	@if $(COMPOSE) ps --status running 2>/dev/null | grep -q 'synapse-api'; then \
		echo "Synapse is already running in Docker Compose."; \
		$(COMPOSE) ps; \
	else \
		if ! docker images -q synapse-db 2>/dev/null | grep -q .; then \
			echo "Image 'synapse-db' not found. Building..."; \
			docker build -t synapse-db pkg/storage/docker; \
		else \
			echo "Image 'synapse-db' found."; \
		fi; \
		if ! docker images -q synapse-api 2>/dev/null | grep -q .; then \
			echo "Image 'synapse-api' not found. Building..."; \
			$(COMPOSE) build api; \
		else \
			echo "Image 'synapse-api' found."; \
		fi; \
		echo "Starting Synapse via Docker Compose..."; \
		$(COMPOSE) up -d; \
		echo ""; \
		echo "============================================"; \
		echo "  Synapse is running! (Docker Compose)"; \
		echo "  API:         http://localhost:8080"; \
		echo "  Swagger UI:  http://localhost:8080/docs"; \
		echo "  OpenAPI spec: http://localhost:8080/swagger.yaml"; \
		echo "============================================"; \
	fi

# ----------------------------------------------------------
# down: Stop and remove containers, networks
# ----------------------------------------------------------
down:
	$(COMPOSE) down

# ----------------------------------------------------------
# up-build: Force rebuild images then start
# ----------------------------------------------------------
up-build:
	$(COMPOSE) up -d --build
	@echo ""
	@echo "============================================"
	@echo "  Synapse is running! (Docker Compose)"
	@echo "  API:         http://localhost:8080"
	@echo "  Swagger UI:  http://localhost:8080/docs"
	@echo "============================================"

# ----------------------------------------------------------
# compose-status: Show Docker Compose service status
# ----------------------------------------------------------
compose-status:
	$(COMPOSE) ps

# ----------------------------------------------------------
# compose-logs: Tail logs from all services
# ----------------------------------------------------------
compose-logs:
	$(COMPOSE) logs -f

# =============================================================
# Local development targets (API on host, DB in standalone Docker)
# =============================================================

# ----------------------------------------------------------
# run: Full local orchestration — DB container + API on host
# ----------------------------------------------------------
run: api-start
	@echo ""
	@echo "============================================"
	@echo "  Synapse is running! (local)"
	@echo "  API:         http://localhost:8080"
	@echo "  Swagger UI:  http://localhost:8080/docs"
	@echo "  OpenAPI spec: http://localhost:8080/swagger.yaml"
	@echo "  Config:      $(CONFIG_FILE)"
	@echo "============================================"

# ----------------------------------------------------------
# stop: Stop local API and DB container
# ----------------------------------------------------------
stop: api-stop db-stop

# ----------------------------------------------------------
# db: Ensure Docker image is built and container is running
#   1. If already running → skip
#   2. If stopped container exists → start it
#   3. Otherwise → delegate to storage Makefile (build + create)
# ----------------------------------------------------------
db:
	@if docker ps --format '{{.Names}}' | grep -q '^synapse-db$$'; then \
		echo "Synapse DB is already running."; \
	elif docker ps -a --format '{{.Names}}' | grep -q '^synapse-db$$'; then \
		echo "Synapse DB container exists but is stopped. Starting..."; \
		docker start synapse-db; \
	else \
		echo "No synapse-db container found. Creating via storage Makefile..."; \
		$(MAKE) -C $(STORAGE_DIR) run; \
	fi

# ----------------------------------------------------------
# db-stop: Stop the DB container
# ----------------------------------------------------------
db-stop:
	@$(MAKE) -C $(STORAGE_DIR) stop

# ----------------------------------------------------------
# db-status: Show DB container status
# ----------------------------------------------------------
db-status:
	@$(MAKE) -C $(STORAGE_DIR) status

# ----------------------------------------------------------
# db-wait: Wait for PostgreSQL to accept connections
# ----------------------------------------------------------
db-wait: db
	@echo "Waiting for PostgreSQL to be ready (up to 60s)..."
	@ready=0; \
	for i in $$(seq 1 60); do \
		if docker exec synapse-db pg_isready -U synapse -d synapse >/dev/null 2>&1; then \
			ready=1; \
			break; \
		fi; \
		echo "  Waiting... ($$i/60)"; \
		sleep 1; \
	done; \
	if [ "$$ready" -ne 1 ]; then \
		echo "ERROR: PostgreSQL not ready after 60s"; \
		echo "Container state:"; \
		docker ps -a --filter name=^synapse-db$$ --format "  {{.Status}}"; \
		echo "Check container logs: docker logs synapse-db"; \
		exit 1; \
	fi
	@echo "PostgreSQL is ready."

# ----------------------------------------------------------
# api: Build the API server binary
# ----------------------------------------------------------
api:
	@echo "Building $(API_BINARY)..."
	@go build -o $(API_BINARY) ./cmd/synapse-api
	@echo "Binary built: ./$(API_BINARY)"

# ----------------------------------------------------------
# api-start: Start the API server locally (ensures DB is up first)
# ----------------------------------------------------------
api-start: db-wait api
	@if [ -f $(API_PID_FILE) ] && kill -0 $$(cat $(API_PID_FILE)) 2>/dev/null; then \
		echo "API server is already running (PID $$(cat $(API_PID_FILE)))."; \
	else \
		echo "Starting API server..."; \
		./$(API_BINARY) -config $(CONFIG_FILE) > $(API_LOG_FILE) 2>&1 & \
		echo $$! > $(API_PID_FILE); \
		sleep 1; \
		if kill -0 $$(cat $(API_PID_FILE)) 2>/dev/null; then \
			echo "API server started (PID $$(cat $(API_PID_FILE)))."; \
		else \
			echo "ERROR: API server failed to start. Check $(API_LOG_FILE)"; \
			cat $(API_LOG_FILE); \
			rm -f $(API_PID_FILE); \
			exit 1; \
		fi; \
	fi

# ----------------------------------------------------------
# api-stop: Stop the local API server
# ----------------------------------------------------------
api-stop:
	@if [ -f $(API_PID_FILE) ] && kill -0 $$(cat $(API_PID_FILE)) 2>/dev/null; then \
		echo "Stopping API server (PID $$(cat $(API_PID_FILE)))..."; \
		kill $$(cat $(API_PID_FILE)); \
		rm -f $(API_PID_FILE); \
		echo "API server stopped."; \
	else \
		echo "API server is not running."; \
		rm -f $(API_PID_FILE) 2>/dev/null; \
	fi

# ----------------------------------------------------------
# api-status: Show whether the local API server is running
# ----------------------------------------------------------
api-status:
	@if [ -f $(API_PID_FILE) ] && kill -0 $$(cat $(API_PID_FILE)) 2>/dev/null; then \
		echo "API server is RUNNING (PID $$(cat $(API_PID_FILE)))."; \
		echo "  Swagger UI: http://localhost:8080/docs"; \
		echo "  Config:     $(CONFIG_FILE)"; \
	else \
		echo "API server is NOT running."; \
	fi

# ----------------------------------------------------------
# clean: Stop everything, remove binary and logs
# ----------------------------------------------------------
clean: stop
	@rm -f $(API_BINARY) $(API_PID_FILE) $(API_LOG_FILE)
	@echo "Cleaned up."
