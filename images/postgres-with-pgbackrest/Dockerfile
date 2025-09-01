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
COPY postgresql.conf /postgresql.conf.template


COPY 001-setup-postgresql.sh /docker-entrypoint-initdb.d/001-setup-postgresql.sh

RUN chmod +x /docker-entrypoint-initdb.d/001-setup-postgresql.sh

# Expose PostgreSQL port
EXPOSE 5432
ENTRYPOINT [ "docker-entrypoint.sh" ]
CMD ["postgres"]
