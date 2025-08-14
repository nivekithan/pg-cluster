#!/bin/bash
set -e

# Function to process environment variables in pgbackrest.conf
process_pgbackrest_config() {
    if [ -f /etc/pgbackrest/pgbackrest.conf.template ]; then
        echo "Processing pgBackRest configuration template..."

        # Use envsubst to replace environment variables
        envsubst < /etc/pgbackrest/pgbackrest.conf.template > /etc/pgbackrest/pgbackrest.conf

        echo "pgBackRest configuration processed successfully"

        # Set proper permissions
        chown postgres:postgres /etc/pgbackrest/pgbackrest.conf
        chmod 644 /etc/pgbackrest/pgbackrest.conf
    else
        echo "Warning: pgBackRest configuration template not found"
    fi
}

# Process pgBackRest configuration
process_pgbackrest_config

# Execute the command passed to the container
exec "$@"