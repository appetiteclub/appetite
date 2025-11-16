#!/usr/bin/env bash
set -e

NATS_CONTAINER="appetite-nats-dev"

stop_nats_docker() {
    if docker ps --filter "name=$NATS_CONTAINER" --filter "status=running" | grep -q "$NATS_CONTAINER"; then
        echo "🛑 Stopping NATS Docker container..."
        docker stop "$NATS_CONTAINER"
        echo "✅ NATS container stopped"
        return 0
    fi
    return 1
}

stop_nats_native() {
    if [ -f "nats.pid" ]; then
        PID=$(cat nats.pid)
        if kill -0 "$PID" 2>/dev/null; then
            echo "🛑 Stopping NATS server (PID: $PID)..."
            kill "$PID"
            rm -f nats.pid
            echo "✅ NATS server stopped"
            return 0
        else
            rm -f nats.pid
        fi
    fi

    if pgrep -x "nats-server" > /dev/null; then
        echo "🛑 Stopping NATS server..."
        pkill -x "nats-server"
        echo "✅ NATS server stopped"
        return 0
    fi

    return 1
}

STOPPED=false

if stop_nats_docker; then
    STOPPED=true
fi

if stop_nats_native; then
    STOPPED=true
fi

if [ "$STOPPED" = false ]; then
    echo "ℹ️  No NATS server running"
fi
