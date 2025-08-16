#!/bin/bash
set -e

echo "Setting up PostgreSQL configuration for pgBackRest..."

# Source centralized environment validation
source /validate-env.sh

# Validate all required environment variables
if ! validate_required_env; then
    exit 1
fi

# Process and append PostgreSQL configuration template
if [ -f /postgresql.conf.template ]; then
    envsubst < /postgresql.conf.template >> "$PGDATA/postgresql.conf"
    echo "PostgreSQL configuration updated with pgBackRest settings:"
    echo "  - STANZA_NAME: $STANZA_NAME"
    echo "  - ARCHIVE_TIMEOUT: $ARCHIVE_TIMEOUT"
    echo "  - MAX_WAL_SENDERS: $MAX_WAL_SENDERS"
    echo "  - WAL_KEEP_SIZE: $WAL_KEEP_SIZE"
else
    echo "Error: postgresql.conf.template not found"
    exit 1
fi

# Add trust authentication for service connections
echo "# Trust authentication for service connections" >> "$PGDATA/pg_hba.conf"
echo "host all all postgres-sample-service trust" >> "$PGDATA/pg_hba.conf"
echo "Added trust authentication for postgres-sample-service"