# ============================================================
# Synapse - Root Makefile
# ============================================================
#
# Quick start:
#   make run       - Build DB (if needed), start DB, build & start API server
#   make stop      - Stop API server and DB
#
# Individual targets:
#   make db        - Ensure DB Docker image is built and container is running
#   make db-stop   - Stop the DB container
#   make db-status - Show DB container status
#   make api       - Build the API server binary
#   make api-start - Start the API server (ensures DB is up first)
#   make api-stop  - Stop the API server
#   make api-status- Show whether the API server is running
#
# Configuration:
#   The API server reads settings from config.yaml (project root).
#   Override individual values with environment variables (see config package).
#
# Swagger UI:
#   After 'make run', open http://localhost:8080/docs
# ============================================================

API_BINARY   := synapse-api
API_PID_FILE := .synapse-api.pid
API_LOG_FILE := .synapse-api.log
CONFIG_FILE  := config.yaml

STORAGE_DIR  := pkg/storage

.PHONY: run stop db db-stop db-status db-wait api api-start api-stop api-status clean

# ----------------------------------------------------------
# run: Full orchestration — DB + API
# ----------------------------------------------------------
run: api-start
	@echo ""
	@echo "============================================"
	@echo "  Synapse is running!"
	@echo "  API:         http://localhost:8080"
	@echo "  Swagger UI:  http://localhost:8080/docs"
	@echo "  OpenAPI spec: http://localhost:8080/swagger.yaml"
	@echo "  Config:      $(CONFIG_FILE)"
	@echo "============================================"

# ----------------------------------------------------------
# stop: Stop everything
# ----------------------------------------------------------
stop: api-stop db-stop

# ----------------------------------------------------------
# db: Ensure Docker image is built and container is running
# ----------------------------------------------------------
db:
	@$(MAKE) -C $(STORAGE_DIR) run

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
	@echo "Waiting for PostgreSQL to be ready..."
	@ready=0; \
	for i in $$(seq 1 20); do \
		if docker exec synapse-db pg_isready -U synapse -d synapse >/dev/null 2>&1; then \
			ready=1; \
			break; \
		fi; \
		echo "  Waiting... ($$i/20)"; \
		sleep 1; \
	done; \
	if [ "$$ready" -ne 1 ]; then \
		echo "ERROR: PostgreSQL not ready after 20s"; \
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
# api-start: Start the API server (ensures DB is up first)
#   - Skips if already running
#   - Runs in background, logs to .synapse-api.log
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
# api-stop: Stop the API server
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
# api-status: Show whether the API server is running
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
