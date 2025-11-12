#!/bin/bash

echo "Checking service health..."

for job in deployments/nomad/jobs/*.nomad; do
    service_name=$(basename "$job" .nomad)
    echo "Checking $service_name..."
    
    if nomad job status "$service_name" | grep -q "Status.*running"; then
        echo "✓ $service_name is running"
    else
        echo "✗ $service_name is not running"
    fi
done
