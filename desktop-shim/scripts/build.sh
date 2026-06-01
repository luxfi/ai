#!/usr/bin/env bash
# Build Lux Desktop from upstream hanzoai/desktop with BRAND=lux overlay.
set -euo pipefail

# shellcheck source=./_resolve-upstream.sh
source "$(dirname "$0")/_resolve-upstream.sh"

sync_brand

cd "$UPSTREAM"
[ -d node_modules ] || npm install --legacy-peer-deps

cd "$APP_DIR"
[ -d node_modules ] || npm install --legacy-peer-deps

VITE_BRAND=lux BRAND=lux npm run build:lux

echo
echo "built: $APP_DIR/src-tauri/target/release/bundle/"
ls "$APP_DIR/src-tauri/target/release/bundle/" 2>/dev/null || true
