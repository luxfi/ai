#!/bin/bash

# Download external binaries from GitHub releases.
#
# Each desktop app ships its OWN node binary (no symlinks):
#   - Zoo  -> zoo-node   (zooai/node)
#   - Zoo  -> zoo-node   (zooai/node)
#   - Hanzo-> hanzo-node (hanzoai/node)
#
# We first try this app's own node release; if that release/asset isn't
# published yet for this target we fall back to the Hanzo Node build and
# install it under this app's own name as a REAL file (a copy, never a
# symlink), so the app always has its own independently-named binary.

set -e

# ---- Per-app configuration -------------------------------------------------
NODE_NAME="zoo-node"            # binary + external-binaries/<NODE_NAME> dir
BRAND="zoo"                     # <BRAND>-tools-runner-resources dir prefix
NODE_REPO="zooai/node"          # this app's own node repo
NODE_TAG="v1.1.14"              # own node release tag to try first
FALLBACK_REPO="hanzoai/node"    # shared AI-backend node used as fallback
FALLBACK_TAG="v1.1.14"
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARIES_DIR="$SCRIPT_DIR/external-binaries"
NODE_DIR="$BINARIES_DIR/$NODE_NAME"
TOOLS_DIR="$NODE_DIR/${BRAND}-tools-runner-resources"

# Detect platform
OS="$(uname -s)"
ARCH="$(uname -m)"

if [ -n "$TAURI_BUILD_TARGET" ]; then
    TARGET="$TAURI_BUILD_TARGET"
    ARCH_NAME="${TARGET%%-*}"
    PLATFORM="${TARGET#*-}"
    echo "Using CI target: $TARGET"
else
    case "$OS" in
        Darwin*) PLATFORM="apple-darwin" ;;
        Linux*)  PLATFORM="unknown-linux-gnu" ;;
        MINGW*|MSYS*|CYGWIN*) PLATFORM="pc-windows-msvc" ;;
        *) echo "Unsupported OS: $OS"; exit 1 ;;
    esac
    case "$ARCH" in
        x86_64|amd64) ARCH_NAME="x86_64" ;;
        arm64|aarch64) ARCH_NAME="aarch64" ;;
        i686|i386) ARCH_NAME="i686" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    TARGET="${ARCH_NAME}-${PLATFORM}"
    echo "Detected platform: $TARGET"
fi

IS_WINDOWS="false"
[[ "$PLATFORM" == *"windows"* ]] && IS_WINDOWS="true"
EXE=""; [[ "$IS_WINDOWS" == "true" ]] && EXE=".exe"

mkdir -p "$TOOLS_DIR"
mkdir -p "$BINARIES_DIR/ollama"

# IMPORTANT: no symlinks. If an old `zoo-node`/symlinked layout exists, drop it.
if [ -L "$BINARIES_DIR/zoo-node" ]; then
    rm -f "$BINARIES_DIR/zoo-node"
    echo "Removed legacy symlink: zoo-node"
fi

download_file() {
    # repo tag asset out  -> 0 on success
    local repo=$1 tag=$2 asset=$3 out=$4
    local url="https://github.com/$repo/releases/download/$tag/$asset"
    echo "Trying $url"
    curl -L -f -sS -o "$out" "$url"
}

create_stub_binary() {
    local target_path=$1
    if [[ "$IS_WINDOWS" == "true" ]]; then
        cat > "${target_path}" <<'STUBEOF'
@echo off
echo Node binary not available for this platform/target.
exit /b 1
STUBEOF
    else
        cat > "${target_path}" <<'STUBEOF'
#!/bin/bash
echo "Node binary not available for this platform/target."
exit 1
STUBEOF
        chmod +x "${target_path}"
    fi
    echo "Created stub binary: ${target_path}"
}

# Install the node binary + bundled tools (deno/uv) under this app's own name.
# $1 = directory the unzip landed in; we look for either <BRAND>/own or hanzo
# named payloads and normalise everything to ${NODE_NAME}-${TARGET}.
install_payload() {
    local src_dir=$1
    # node binary: accept own-named, hanzo-named, or bare
    local node_bin=""
    for cand in "$src_dir/${NODE_NAME}${EXE}" "$src_dir/hanzo-node${EXE}" "$src_dir/node${EXE}"; do
        [ -f "$cand" ] && node_bin="$cand" && break
    done
    if [ -z "$node_bin" ]; then
        echo "  ! no node binary found in payload"
        return 1
    fi
    cp "$node_bin" "$NODE_DIR/${NODE_NAME}${EXE}"
    cp "$node_bin" "$NODE_DIR/${NODE_NAME}-${TARGET}${EXE}"
    chmod +x "$NODE_DIR/${NODE_NAME}"* 2>/dev/null || true

    # bundled tool runners (deno/uv) ship inside the hanzo/own zip under a
    # *-tools-runner-resources dir; normalise into our brand tools dir.
    local found_tools
    for tool in deno uv; do
        found_tools="$(find "$src_dir" -type f -name "$tool" 2>/dev/null | head -1)"
        if [ -n "$found_tools" ]; then
            cp "$found_tools" "$TOOLS_DIR/${tool}${EXE}"
            cp "$found_tools" "$TOOLS_DIR/${tool}-${TARGET}${EXE}"
            chmod +x "$TOOLS_DIR/${tool}"* 2>/dev/null || true
        fi
    done
}

if [ ! -f "$NODE_DIR/${NODE_NAME}-${TARGET}${EXE}" ]; then
    TMP="$(mktemp -d)"
    OK="false"

    # 1) try this app's own node release
    if download_file "$NODE_REPO" "$NODE_TAG" "${NODE_NAME}-${TARGET}.zip" "$TMP/own.zip"; then
        unzip -o "$TMP/own.zip" -d "$TMP/own" >/dev/null 2>&1 || true
        if install_payload "$TMP/own"; then OK="true"; echo "Installed $NODE_NAME from $NODE_REPO@$NODE_TAG"; fi
    fi

    # 2) fall back to the shared Hanzo Node build, installed under our own name
    if [[ "$OK" != "true" ]]; then
        echo "Own release unavailable for $TARGET; falling back to $FALLBACK_REPO (installed as $NODE_NAME, real copy)"
        if download_file "$FALLBACK_REPO" "$FALLBACK_TAG" "hanzo-node-${TARGET}.zip" "$TMP/fb.zip"; then
            unzip -o "$TMP/fb.zip" -d "$TMP/fb" >/dev/null 2>&1 || true
            if install_payload "$TMP/fb"; then OK="true"; echo "Installed $NODE_NAME (from Hanzo Node fallback)"; fi
        fi
    fi

    # 3) last resort: stub so the build still bundles a sidecar
    if [[ "$OK" != "true" ]]; then
        echo "Warning: no node binary available for $TARGET; creating stub"
        create_stub_binary "$NODE_DIR/${NODE_NAME}-${TARGET}${EXE}"
        cp "$NODE_DIR/${NODE_NAME}-${TARGET}${EXE}" "$NODE_DIR/${NODE_NAME}${EXE}" 2>/dev/null || true
        create_stub_binary "$TOOLS_DIR/deno-${TARGET}${EXE}"
        create_stub_binary "$TOOLS_DIR/uv-${TARGET}${EXE}"
    fi

    rm -rf "$TMP"
fi

# ---- Ollama ----------------------------------------------------------------
OLLAMA_TAG="v0.12.9"
OLLAMA_AVAILABLE=true
OLLAMA_ARCHIVE=""
if [[ "$PLATFORM" == "apple-darwin" ]]; then
    OLLAMA_ARCHIVE="ollama-darwin.tgz"
elif [[ "$PLATFORM" == "unknown-linux-gnu" ]]; then
    if [[ "$ARCH_NAME" == "x86_64" ]]; then OLLAMA_ARCHIVE="ollama-linux-amd64.tgz"
    elif [[ "$ARCH_NAME" == "aarch64" ]]; then OLLAMA_ARCHIVE="ollama-linux-arm64.tgz"
    else OLLAMA_AVAILABLE=false; fi
elif [[ "$PLATFORM" == *"windows"* ]]; then
    if [[ "$ARCH_NAME" == "x86_64" ]]; then OLLAMA_ARCHIVE="ollama-windows-amd64.zip"
    elif [[ "$ARCH_NAME" == "aarch64" ]]; then OLLAMA_ARCHIVE="ollama-windows-arm64.zip"
    else OLLAMA_AVAILABLE=false; fi
else OLLAMA_AVAILABLE=false; fi

create_ollama_stub() {
    local target_path=$1
    if [[ "$IS_WINDOWS" == "true" ]]; then
        printf '@echo off\necho Ollama not available for this platform.\nexit /b 1\n' > "${target_path}"
    else
        printf '#!/bin/bash\necho "Ollama not available for this platform."\nexit 1\n' > "${target_path}"
        chmod +x "${target_path}"
    fi
    echo "Created stub binary: ${target_path}"
}

OLLAMA_BIN_NAME="ollama-${TARGET}${EXE}"
if [ ! -f "$BINARIES_DIR/ollama/$OLLAMA_BIN_NAME" ]; then
    if [[ "$OLLAMA_AVAILABLE" == "true" ]]; then
        echo "Downloading Ollama ${OLLAMA_TAG}..."
        url="https://github.com/ollama/ollama/releases/download/${OLLAMA_TAG}/${OLLAMA_ARCHIVE}"
        DOWNLOAD_SUCCESS=false
        if curl -L -f -sS -o "$BINARIES_DIR/ollama/${OLLAMA_ARCHIVE}" "$url"; then
            cd "$BINARIES_DIR/ollama"
            if [[ "$OLLAMA_ARCHIVE" == *.tgz ]]; then
                tar -xzf "$OLLAMA_ARCHIVE"
                if [ -f "ollama" ]; then cp ollama "$OLLAMA_BIN_NAME"; chmod +x ollama "$OLLAMA_BIN_NAME"; DOWNLOAD_SUCCESS=true
                elif [ -f "bin/ollama" ]; then cp bin/ollama "$OLLAMA_BIN_NAME"; chmod +x "$OLLAMA_BIN_NAME"; DOWNLOAD_SUCCESS=true; fi
            elif [[ "$OLLAMA_ARCHIVE" == *.zip ]]; then
                unzip -o "$OLLAMA_ARCHIVE"
                if [ -f "ollama.exe" ]; then cp ollama.exe "$OLLAMA_BIN_NAME"; DOWNLOAD_SUCCESS=true; fi
            fi
            rm -f "$OLLAMA_ARCHIVE"
            [[ "$DOWNLOAD_SUCCESS" == "true" ]] && echo "Downloaded Ollama ${OLLAMA_TAG}" || { echo "Ollama binary not found, stubbing"; create_ollama_stub "$BINARIES_DIR/ollama/$OLLAMA_BIN_NAME"; }
        else
            echo "Warning: failed to download Ollama, creating stub"; create_ollama_stub "$BINARIES_DIR/ollama/$OLLAMA_BIN_NAME"
        fi
    else
        echo "Note: Ollama not available for $TARGET, creating stub"; create_ollama_stub "$BINARIES_DIR/ollama/$OLLAMA_BIN_NAME"
    fi
fi

echo "Binary download complete!"
echo "Node binary: $NODE_DIR/${NODE_NAME}-${TARGET}${EXE}"
