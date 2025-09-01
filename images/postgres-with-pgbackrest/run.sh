#!/bin/bash

# Stop and remove existing container if it exists
docker stop postgres-container 2>/dev/null || true
docker rm postgres-container 2>/dev/null || true

# Build the image
docker build -t postgres-with-pgbackrest .

# Run the container
docker run -d --name postgres-container  -e POSTGRES_PASSWORD=mypassword -p 5432:5432 postgres-with-pgbackrest
