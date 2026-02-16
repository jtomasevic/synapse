#!/bin/bash
set -e

# ============================================================
# Synapse DB initialization script
# Runs inside the PostgreSQL container on first startup.
#
# This script is executed by the postgres entrypoint when placed
# in /docker-entrypoint-initdb.d/. The entrypoint only runs
# these scripts when the data directory is empty (i.e. first
# initialization), so the "already initialized" case is handled
# automatically by Docker's postgres image.
#
# However, we add an explicit check so the script is also safe
# to run manually against an existing cluster.
# ============================================================

SYNAPSE_DB="synapse"
SYNAPSE_USER="synapse"
SYNAPSE_PASS="synapse"

echo "=== Synapse DB initialization ==="

# Check if the database already exists
DB_EXISTS=$(psql -U "$POSTGRES_USER" -tAc "SELECT 1 FROM pg_database WHERE datname='${SYNAPSE_DB}'" 2>/dev/null || true)

if [ "$DB_EXISTS" = "1" ]; then
    echo "Database '${SYNAPSE_DB}' already exists. Skipping creation."
else
    echo "Creating user '${SYNAPSE_USER}'..."
    psql -U "$POSTGRES_USER" -tAc "SELECT 1 FROM pg_roles WHERE rolname='${SYNAPSE_USER}'" | grep -q 1 || \
        psql -U "$POSTGRES_USER" -c "CREATE USER ${SYNAPSE_USER} WITH PASSWORD '${SYNAPSE_PASS}';"

    echo "Creating database '${SYNAPSE_DB}'..."
    psql -U "$POSTGRES_USER" -c "CREATE DATABASE ${SYNAPSE_DB} OWNER ${SYNAPSE_USER};"

    echo "Granting privileges..."
    psql -U "$POSTGRES_USER" -c "GRANT ALL PRIVILEGES ON DATABASE ${SYNAPSE_DB} TO ${SYNAPSE_USER};"

    echo "Database '${SYNAPSE_DB}' created successfully."
fi

# Apply schema (idempotent - all CREATE IF NOT EXISTS)
echo "Applying schema to '${SYNAPSE_DB}'..."
psql -U "${SYNAPSE_USER}" -d "${SYNAPSE_DB}" -f /docker-entrypoint-initdb.d/init.sql

# Grant schema-level permissions (needed for PostgreSQL 15+)
psql -U "$POSTGRES_USER" -d "${SYNAPSE_DB}" -c "GRANT ALL ON SCHEMA public TO ${SYNAPSE_USER};"
psql -U "$POSTGRES_USER" -d "${SYNAPSE_DB}" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ${SYNAPSE_USER};"
psql -U "$POSTGRES_USER" -d "${SYNAPSE_DB}" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ${SYNAPSE_USER};"

echo "=== Synapse DB initialization complete ==="
