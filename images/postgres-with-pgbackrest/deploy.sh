#!/bin/bash

# Docker image deploy script for postgres-pgbackrest
set -e  # Exit on any error

# Image configuration
IMAGE_NAME="postgres-pgbackrest"
DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:-$(docker system info | grep -i username | awk '{print $2}')}"
FULL_IMAGE_NAME="${DOCKERHUB_USERNAME}/${IMAGE_NAME}"

echo "🏗️  Building Docker image: ${FULL_IMAGE_NAME}:latest"
docker build -t "${FULL_IMAGE_NAME}:latest" .

echo "📤 Pushing image to Docker Hub"
docker push "${FULL_IMAGE_NAME}:latest"

echo "✅ Successfully deployed ${FULL_IMAGE_NAME}:latest to Docker Hub"
echo "📋 To use this image:"
echo "   docker pull ${FULL_IMAGE_NAME}:latest"
