#!/usr/bin/env bash
set -e

NATS_CONTAINER="appetite-nats-dev"
NATS_IMAGE="nats:2.10-alpine"
NATS_PORT="${NATS_PORT:-4222}"
NATS_MONITOR_PORT="${NATS_MONITOR_PORT:-8222}"

check_nats_running() {
    if command -v nats-server &> /dev/null; then
        if pgrep -x "nats-server" > /dev/null; then
            echo "✅ NATS server already running (native)"
            return 0
        fi
    fi

    if docker ps --filter "name=$NATS_CONTAINER" --filter "status=running" | grep -q "$NATS_CONTAINER"; then
        echo "✅ NATS server already running (Docker: $NATS_CONTAINER)"
        return 0
    fi

    return 1
}

start_nats_docker() {
    echo "🚀 Starting NATS server in Docker..."

    if docker ps -a --filter "name=$NATS_CONTAINER" | grep -q "$NATS_CONTAINER"; then
        echo "   Container exists, starting..."
        docker start "$NATS_CONTAINER"
    else
        echo "   Creating new container..."
        docker run -d \
            --name "$NATS_CONTAINER" \
            -p "${NATS_PORT}:4222" \
            -p "${NATS_MONITOR_PORT}:8222" \
            "$NATS_IMAGE" \
            -js -m 8222
    fi

    sleep 2

    if docker ps --filter "name=$NATS_CONTAINER" --filter "status=running" | grep -q "$NATS_CONTAINER"; then
        echo "✅ NATS server started successfully"
        echo "   Client port: $NATS_PORT"
        echo "   Monitor: http://localhost:$NATS_MONITOR_PORT"
        return 0
    else
        echo "❌ Failed to start NATS server"
        return 1
    fi
}

start_nats_native() {
    echo "🚀 Starting NATS server (native)..."
    nats-server -js -m "$NATS_MONITOR_PORT" -p "$NATS_PORT" > nats.log 2>&1 &
    echo $! > nats.pid
    sleep 2

    if pgrep -x "nats-server" > /dev/null; then
        echo "✅ NATS server started successfully"
        echo "   Client port: $NATS_PORT"
        echo "   Monitor: http://localhost:$NATS_MONITOR_PORT"
        echo "   PID: $(cat nats.pid)"
        return 0
    else
        echo "❌ Failed to start NATS server"
        return 1
    fi
}

if check_nats_running; then
    exit 0
fi

if command -v nats-server &> /dev/null; then
    start_nats_native
elif command -v docker &> /dev/null; then
    start_nats_docker
else
    echo "❌ Neither nats-server nor Docker found"
    echo "   Please install NATS or Docker. See docs/draft/nats-installation.md"
    exit 1
fi
