#!/bin/bash
set -e

# Source centralized environment validation
source /validate-env.sh

# Validate all required environment variables
if ! validate_required_env; then
    exit 1
fi

# Process pgBackRest configuration template
envsubst < /etc/pgbackrest/pgbackrest.conf.template > /etc/pgbackrest/pgbackrest.conf

ls /var/lib/postgresql/data

case "$1" in
    "postgres")
        # Start PostgreSQL with original entrypoint
        exec docker-entrypoint.sh postgres
        ;;
    "backup")
        # Execute backup command with remaining arguments
        exec pgbackrest backup "${@:2}"
        ;;
    *)
        # For any other command, execute as-is
        exec "$@"
        ;;
esac