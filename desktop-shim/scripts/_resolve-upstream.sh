#!/usr/bin/env bash
# Resolve the upstream hanzo/desktop checkout. Sourced by build.sh / dev.sh.
# Forward-only: one resolution path. Override with HANZO_DESKTOP=/abs/path.
set -euo pipefail

SHIM_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UPSTREAM="${HANZO_DESKTOP:-$SHIM_ROOT/../../../hanzo/desktop}"

if [ ! -d "$UPSTREAM/apps/hanzo-desktop" ]; then
  echo "error: upstream hanzo/desktop not found at $UPSTREAM" >&2
  echo "  clone github.com/hanzoai/desktop next to this repo, or set HANZO_DESKTOP." >&2
  exit 1
fi

UPSTREAM="$(cd "$UPSTREAM" && pwd)"
APP_DIR="$UPSTREAM/apps/hanzo-desktop"
ICONS_DEST="$APP_DIR/src-tauri/icons/lux"

export SHIM_ROOT UPSTREAM APP_DIR ICONS_DEST

sync_brand() {
  rm -rf "$ICONS_DEST"
  mkdir -p "$(dirname "$ICONS_DEST")"
  ln -s "$SHIM_ROOT/brand/icons" "$ICONS_DEST"
}
