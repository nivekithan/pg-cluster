set -e

echo "Starting setup script..."

echo "Setting up PostgreSQL configuration for pgBackRest..."

# Note: Skipping environment validation as /validate-env.sh is not available

echo "PGDATA is: $PGDATA"

envsubst < /postgresql.conf.template >> "$PGDATA/postgresql.conf"
