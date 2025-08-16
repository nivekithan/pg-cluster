#!/bin/bash

set -e

POD_NAME="debug-postgres-$(date +%s)"

echo "Creating postgres debug pod: $POD_NAME"
kubectl run $POD_NAME --image=postgres:17 --restart=Never --rm -i --tty --env="POSTGRES_PASSWORD=debug" -- bash

echo "Pod $POD_NAME will be automatically deleted when you exit."