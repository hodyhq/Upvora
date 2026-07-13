#!/bin/sh
# Upvora self-update sidecar.
#
# Watches a shared volume for a trigger file written by the app's admin System
# box and, when it appears, pulls the app image and recreates the container.
# The app container never touches the Docker socket — only this sidecar does.
# The trigger file carries no content; it is a doorbell, not a message.
#
# Compose service (add next to app/postgres; see etc/updater/compose.example.yml):
#   updater:
#     image: docker:cli
#     restart: unless-stopped
#     volumes:
#       - /var/run/docker.sock:/var/run/docker.sock
#       - .:/compose
#       - updater-shared:/shared
#     entrypoint: ["sh", "/compose/updater.sh"]
# and on the app service:
#   volumes:
#     - updater-shared:/var/upvora-updater
#   environment:
#     - UPDATER_SHARED_DIR=/var/upvora-updater

SHARED=${SHARED_DIR:-/shared}
COMPOSE_DIR=${COMPOSE_DIR:-/compose}

cd "$COMPOSE_DIR" || exit 1
mkdir -p "$SHARED"

echo "[updater] watching $SHARED for update requests"
while true; do
  if [ -f "$SHARED/update-requested" ]; then
    rm -f "$SHARED/update-requested"
    echo "running" > "$SHARED/status"
    {
      echo "[updater] $(date -u '+%Y-%m-%d %H:%M:%S') update requested — pulling app image"
      docker compose pull app &&
        echo "[updater] recreating app container" &&
        docker compose up -d app &&
        echo "[updater] done"
    } > "$SHARED/log" 2>&1 && echo "done" > "$SHARED/status" || echo "error" > "$SHARED/status"
  fi
  sleep 5
done
