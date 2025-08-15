#!/bin/bash
docker build -t mypgimg . && \
cid=$(docker run -d --rm --env-file .env -v postgres-data:/var/lib/postgresql/data mypgimg postgres) && \
echo "Container ID: $cid" && \
if [ -t 0 ]; then
    docker exec -it $cid bash
else
    echo "Non-interactive environment detected. Container is running in background."
    echo "Use: docker exec -it $cid bash"
fi
# Container auto-removes when stopped due to --rm flag