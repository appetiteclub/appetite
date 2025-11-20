# Envoy Proxy for gRPC-Web

This directory contains the Envoy proxy configuration to enable gRPC-Web communication between the browser and Kitchen service.

## Architecture

```
Browser (gRPC-Web) → Envoy Proxy (:8090) → Kitchen gRPC Server (:50051)
```

## Running Envoy

### Using Docker Compose (Recommended)

```bash
cd deployments/envoy
docker-compose up
```

### Using Docker directly

```bash
docker run --rm -it --network host \
  -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml:ro \
  envoyproxy/envoy:v1.28-latest \
  -c /etc/envoy/envoy.yaml
```

## Ports

- **8090**: gRPC-Web endpoint (browser connects here)
- **9901**: Envoy admin interface
- **50051**: Kitchen gRPC server (internal)

## Testing

### Check Envoy is running
```bash
curl http://localhost:9901/ready
```

### Test with grpcurl (requires Kitchen service running)
```bash
# List services
grpcurl -plaintext localhost:50051 list

# Subscribe to events
grpcurl -plaintext -d '{"station_id": ""}' \
  localhost:50051 appetite.kitchen.v1.EventStream/StreamKitchenEvents
```

## Browser Integration

The browser client will connect to `http://localhost:8090` using gRPC-Web protocol.
Envoy translates the requests to native gRPC and forwards to Kitchen service on port 50051.
