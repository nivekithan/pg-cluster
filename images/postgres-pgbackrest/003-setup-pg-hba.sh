#!/bin/bash
set -e

echo "Setting up pg_hba.conf with trust authentication..."

# Create new pg_hba.conf with trust authentication
cat > "$PGDATA/pg_hba.conf" << 'EOF'
# TYPE  DATABASE        USER            ADDRESS                 METHOD

# "local" is for Unix domain socket connections only
local   all             all                                     trust
# IPv4 local connections:
host    all             all             127.0.0.1/32            trust
# IPv6 local connections:
host    all             all             ::1/128                 trust
# Allow replication connections from localhost, by a user with the
# replication privilege.
local   replication     all                                     trust
host    replication     all             127.0.0.1/32            trust
host    replication     all             ::1/128                 trust
host    all             all             10.0.0.0/8              trust

host all all all scram-sha-256
EOF

# Add trust authentication for Kubernetes internal network
echo "Added trust authentication for Kubernetes internal network: 10.0.0.0/8"

echo "pg_hba.conf configuration completed"