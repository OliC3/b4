#!/bin/sh
# Watch for changes and rebuild

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
"$SCRIPT_DIR/_build.sh"

# Try inotifywait first (Linux)
if command -v inotifywait >/dev/null 2>&1; then
    echo "Watching installer/ with inotifywait..."
    while true; do
        inotifywait -q -e modify,create,delete "$SCRIPT_DIR"/*.sh
        echo "Change detected, rebuilding..."
        "$SCRIPT_DIR/_build.sh"
    done
# Try fswatch (macOS)
elif command -v fswatch >/dev/null 2>&1; then
    echo "Watching installer/ with fswatch..."
    fswatch -o "$SCRIPT_DIR"/*.sh | while read; do
        echo "Change detected, rebuilding..."
        "$SCRIPT_DIR/_build.sh"
    done
# Fallback: polling
else
    echo "No inotifywait/fswatch, using polling (2s)..."
    last_hash=""
    while true; do
        current_hash=$(cat "$SCRIPT_DIR"/*.sh 2>/dev/null | md5sum | cut -d' ' -f1)
        if [ "$current_hash" != "$last_hash" ]; then
            [ -n "$last_hash" ] && echo "Change detected, rebuilding..." && "$SCRIPT_DIR/_build.sh"
            last_hash="$current_hash"
        fi
        sleep 2
    done
fi
