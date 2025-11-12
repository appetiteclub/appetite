#!/bin/bash
set -e

echo "Deploying all services..."

for job in deployments/nomad/jobs/*.nomad; do
    if [ -f "$job" ]; then
        echo "Deploying $(basename "$job")..."
        nomad job run "$job"
    fi
done

echo "All services deployed successfully!"
