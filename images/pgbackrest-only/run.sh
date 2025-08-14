#!/bin/bash
docker build -t pgbackrest-only . && \
cid=$(docker run -d --rm --env-file .env -v postgres-data:/var/lib/postgresql/data pgbackrest-only) && \
echo "Container ID: $cid" && \
docker exec -it $cid bash
# Container auto-removes when stopped due to --rm flag