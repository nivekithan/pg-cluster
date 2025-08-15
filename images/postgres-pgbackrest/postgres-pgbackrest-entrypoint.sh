#!/bin/bash
set -e

# Check required pgBackRest environment variables
required_pgbackrest_vars=(
    "S3_BUCKET_NAME"
    "S3_ENDPOINT"
    "S3_ACCESS_KEY"
    "S3_ACCESS_KEY_SECRET"
    "S3_REGION"
    "REPO1_RETENTION_FULL"
    "STANZA_NAME"
    "PGDATA"
)
missing_vars=()

for var in "${required_pgbackrest_vars[@]}"; do
    if [ -z "${!var}" ]; then
        missing_vars+=("$var")
    fi
done

if [ ${#missing_vars[@]} -ne 0 ]; then
    echo "Error: Required pgBackRest environment variables are missing:"
    printf "  - %s\n" "${missing_vars[@]}"
    echo "Please set all required variables and try again."
    exit 1
fi

# Process pgBackRest configuration template
envsubst < /etc/pgbackrest/pgbackrest.conf.template > /etc/pgbackrest/pgbackrest.conf

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