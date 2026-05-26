#!/usr/bin/env bash
#
# update-fider.sh — deploy an HCM Fider release on VM 201.
#
# Usage:
#   sudo ./update-fider.sh                # deploys the latest hcm-* tag
#   sudo ./update-fider.sh hcm-v0.1.0     # deploys a specific tag
#
# The repo at $REPO_URL is expected to carry HCM release tags (hcm-vX.Y.Z)
# on top of the upstream Fider history, so every deployable release is
# preserved in git.

set -euo pipefail

# --- Configuration ----------------------------------------------------------
REPO_URL="https://git.hcm.adev/hody/fider"
SRC_DIR="/opt/fider-src"
COMPOSE_DIR="/opt/fider"
COMPOSE_FILE="${COMPOSE_DIR}/docker-compose.yml"
IMAGE_TAG="fider-hcm:stable"
TAG_PATTERN="hcm-*"
REQUESTED_TAG="${1:-}"

# --- Helpers ----------------------------------------------------------------
log()  { printf '\n\033[1;32m[update-fider]\033[0m %s\n' "$*"; }
warn() { printf '\n\033[1;33m[update-fider]\033[0m %s\n' "$*"; }
die()  { printf '\n\033[1;31m[update-fider ERROR]\033[0m %s\n' "$*" >&2; exit 1; }

command -v git    >/dev/null || die "git is not installed"
command -v docker >/dev/null || die "docker is not installed"
docker compose version >/dev/null 2>&1 || die "docker compose plugin is not available"

# --- 1. Clone or pull -------------------------------------------------------
if [[ ! -d "$SRC_DIR/.git" ]]; then
  log "Cloning $REPO_URL into $SRC_DIR"
  mkdir -p "$(dirname "$SRC_DIR")"
  git clone "$REPO_URL" "$SRC_DIR" || die "git clone failed"
else
  log "Updating existing checkout at $SRC_DIR"
fi

cd "$SRC_DIR"
git fetch --all --tags --prune --force || die "git fetch failed"

# --- Resolve which release tag to deploy ------------------------------------
if [[ -n "$REQUESTED_TAG" ]]; then
  RELEASE_TAG="$REQUESTED_TAG"
  git rev-parse -q --verify "refs/tags/${RELEASE_TAG}" >/dev/null \
    || die "Requested tag '${RELEASE_TAG}' does not exist in $REPO_URL"
else
  RELEASE_TAG="$(git tag -l "$TAG_PATTERN" --sort=-v:refname | head -n1 || true)"
  [[ -n "$RELEASE_TAG" ]] || die "No tag matching '$TAG_PATTERN' found. Tag a release on hcm-theme (e.g. 'git tag hcm-v0.1.0 && git push origin hcm-v0.1.0') and re-run."
fi

log "Deploying release: $RELEASE_TAG"
git checkout --force "$RELEASE_TAG" || die "git checkout $RELEASE_TAG failed"

[[ -f "Dockerfile" ]] || die "No Dockerfile at $SRC_DIR/Dockerfile"

# --- 2. Build Docker image --------------------------------------------------
log "Building Docker image $IMAGE_TAG (this can take several minutes)"
docker build -t "$IMAGE_TAG" . || die "Docker build failed"

# Also tag the image with the release name so older builds are retained locally.
docker tag "$IMAGE_TAG" "fider-hcm:${RELEASE_TAG}" \
  || warn "Could not apply release tag to image (continuing)"

# --- 3. Update docker-compose.yml -------------------------------------------
[[ -f "$COMPOSE_FILE" ]] || die "$COMPOSE_FILE not found"

BACKUP="${COMPOSE_FILE}.bak.$(date +%Y%m%d-%H%M%S)"
cp "$COMPOSE_FILE" "$BACKUP"
log "Backed up compose file to $BACKUP"

# Match either the upstream image or a previous HCM image and rewrite it.
# The (^[[:space:]]*image:[[:space:]]*) capture preserves indentation.
sed -i -E "s|^([[:space:]]*image:[[:space:]]*)(getfider/fider|fider-hcm):.*|\1${IMAGE_TAG}|" "$COMPOSE_FILE"

if ! grep -qE "^[[:space:]]*image:[[:space:]]*${IMAGE_TAG//\//\\/}\s*$" "$COMPOSE_FILE"; then
  die "Failed to rewrite image line in $COMPOSE_FILE — restore from $BACKUP and inspect manually"
fi
log "Pointed $COMPOSE_FILE at $IMAGE_TAG"

# --- 4. Bring up the stack --------------------------------------------------
cd "$COMPOSE_DIR"
log "Running docker compose up -d"
docker compose up -d || die "docker compose up failed"

# --- 5. Wait and check ------------------------------------------------------
log "Waiting 10s for services to settle..."
sleep 10

log "Current container state:"
docker compose ps

# --- 6. Final status --------------------------------------------------------
# Find any service whose name contains 'fider' (the compose service is usually
# named 'fider' or 'app') and confirm it is in the 'running' state.
if docker compose ps --status running --format '{{.Service}} {{.Image}}' \
     | grep -q "$IMAGE_TAG"; then
  log "SUCCESS: Fider release $RELEASE_TAG is live (image $IMAGE_TAG)"
  exit 0
else
  warn "Fider container does not appear to be running on $IMAGE_TAG."
  warn "Inspect with: docker compose -f $COMPOSE_FILE logs --tail=200"
  die  "Deployment did not reach a healthy running state"
fi
