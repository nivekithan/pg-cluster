#!/bin/bash
set -e

echo "Creating pgBackRest stanza..."

# Check required environment variable
if [ -z "$STANZA_NAME" ]; then
    echo "Error: STANZA_NAME environment variable is required"
    exit 1
fi

# Create stanza
pgbackrest stanza-create --stanza="$STANZA_NAME"

echo "pgBackRest stanza '$STANZA_NAME' created successfully"