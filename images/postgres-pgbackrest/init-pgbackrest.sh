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

# Custom function to handle postgresql.conf placement and pgbackrest stanza creation
setup_postgresql_config() {
    export PGDATA=${PGDATA:-/var/lib/postgresql/data}

    # Create a wrapper script that will copy postgresql.conf after PGDATA is initialized
    cat > /docker-entrypoint-initdb.d/01-copy-postgresql-conf.sh << 'EOF'
#!/bin/bash
if [ -f /postgresql.conf.template ] && [ -d "$PGDATA" ]; then
    echo "Copying PostgreSQL configuration to PGDATA..."
    cp /postgresql.conf.template "$PGDATA/postgresql.conf"
    chown postgres:postgres "$PGDATA/postgresql.conf"
    chmod 644 "$PGDATA/postgresql.conf"
    echo "PostgreSQL configuration copied successfully"
fi
EOF
    chmod +x /docker-entrypoint-initdb.d/01-copy-postgresql-conf.sh

    # Create pgBackRest stanza creation script
    cat > /docker-entrypoint-initdb.d/02-create-pgbackrest-stanza.sh << 'EOF'
#!/bin/bash
echo "Creating pgBackRest stanza..."
if pgbackrest --stanza=my-pg-database stanza-create; then
    echo "pgBackRest stanza created successfully"
else
    echo "Warning: Failed to create pgBackRest stanza"
fi
EOF
    chmod +x /docker-entrypoint-initdb.d/02-create-pgbackrest-stanza.sh
}

# Process pgBackRest configuration
process_pgbackrest_config

# Setup PostgreSQL configuration
setup_postgresql_config



# Execute the original postgres entrypoint
exec /usr/local/bin/docker-entrypoint.sh "$@"