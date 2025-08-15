FROM postgres:17

RUN apt-get update && \
    apt-get install -y \
        pgbackrest \
        gettext-base \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Create pgBackRest directories with proper permissions
RUN mkdir -p /etc/pgbackrest/conf.d && \
    mkdir -p /var/log/pgbackrest && \
    mkdir -p /var/lib/pgbackrest && \
    mkdir -p /pgbackrest && \
    chown -R postgres:postgres /etc/pgbackrest /var/log/pgbackrest /var/lib/pgbackrest /pgbackrest

# Copy configuration templates
COPY pgbackrest.conf /etc/pgbackrest/pgbackrest.conf.template
COPY postgresql.conf /postgresql.conf.template

# Copy validation script
COPY validate-env.sh /validate-env.sh

# Copy entrypoint and initialization scripts
COPY postgres-pgbackrest-entrypoint.sh /postgres-pgbackrest-entrypoint.sh
COPY 001-setup-postgresql-conf.sh /docker-entrypoint-initdb.d/001-setup-postgresql-conf.sh
COPY 002-pgbackrest-create-stanza.sh /docker-entrypoint-initdb.d/002-pgbackrest-create-stanza.sh

RUN chown postgres:postgres /etc/pgbackrest/pgbackrest.conf.template /postgresql.conf.template && \
    chmod +x /validate-env.sh && \
    chmod +x /postgres-pgbackrest-entrypoint.sh && \
    chmod +x /docker-entrypoint-initdb.d/001-setup-postgresql-conf.sh && \
    chmod +x /docker-entrypoint-initdb.d/002-pgbackrest-create-stanza.sh

# Expose PostgreSQL port
EXPOSE 5432

# Use postgres user as default
USER postgres

ENTRYPOINT ["/postgres-pgbackrest-entrypoint.sh"]