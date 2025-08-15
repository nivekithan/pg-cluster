#!/bin/bash
set -e

echo "Setting up PostgreSQL configuration for pgBackRest..."

# Check required environment variables
required_vars=("STANZA_NAME" "ARCHIVE_TIMEOUT" "MAX_WAL_SENDERS" "WAL_KEEP_SIZE")
missing_vars=()

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        missing_vars+=("$var")
    fi
done

if [ ${#missing_vars[@]} -ne 0 ]; then
    echo "Error: Required environment variables are missing:"
    printf "  - %s\n" "${missing_vars[@]}"
    echo "Please set all required variables and try again."
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