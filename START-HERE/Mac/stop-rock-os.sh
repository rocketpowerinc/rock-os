#!/usr/bin/env sh
set -eu

PORT="${1:-8000}"

echo "Looking for Rock-OS on port $PORT..."

PIDS=""

if command -v lsof >/dev/null 2>&1; then
    PIDS="$(lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true)"
elif command -v ss >/dev/null 2>&1; then
    PIDS="$(ss -ltnp "sport = :$PORT" 2>/dev/null | awk -F'pid=' 'NF > 1 { split($2, parts, ","); print parts[1] }' | sort -u)"
elif command -v netstat >/dev/null 2>&1; then
    PIDS="$(netstat -ltnp 2>/dev/null | awk -v port=":$PORT" '$4 ~ port "$" { split($7, parts, "/"); print parts[1] }' | sort -u)"
fi

if [ -z "$PIDS" ]; then
    echo "No process is listening on port $PORT."
    exit 0
fi

for PID in $PIDS; do
    case "$PID" in
        ''|*[!0-9]*)
            continue
            ;;
    esac

    echo "Stopping process $PID on port $PORT..."
    kill "$PID" 2>/dev/null || {
        echo "Could not stop process $PID. You may need sudo:"
        echo "  sudo kill $PID"
        exit 1
    }
done

echo "Rock-OS stop request complete."
