#!/bin/bash
set -e

echo "Creating pgBackRest stanza..."

# Source centralized environment validation
source /validate-env.sh

# Validate all required environment variables
if ! validate_required_env; then
    exit 1
fi

# Create stanza
pgbackrest stanza-create --stanza="$STANZA_NAME" --log-level-console=info

echo "pgBackRest stanza '$STANZA_NAME' created successfully"