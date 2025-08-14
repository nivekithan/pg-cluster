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
    chown -R postgres:postgres /etc/pgbackrest /var/log/pgbackrest /var/lib/pgbackrest

# Copy pgBackRest configuration template
COPY pgbackrest.conf /etc/pgbackrest/pgbackrest.conf.template
RUN chown postgres:postgres /etc/pgbackrest/pgbackrest.conf.template

# Copy pgBackRest initialization script
COPY init-pgbackrest.sh /init-pgbackrest.sh
RUN chmod +x /init-pgbackrest.sh

# Use postgres user as default
USER postgres

# Use custom initialization script that processes environment variables
ENTRYPOINT ["/init-pgbackrest.sh"]
CMD ["tail", "-f", "/dev/null"]