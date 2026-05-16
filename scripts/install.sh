#!/usr/bin/env sh
# shipcheck installer — detects OS/arch, downloads the right binary from GitHub releases
# Usage: curl -fsSL https://get.shipcheck.dev | sh
# Or:    curl -fsSL https://raw.githubusercontent.com/tejgokani/shipcheck/main/scripts/install.sh | sh

set -e

REPO="tejgokani/shipcheck"
BINARY_NAME="shipcheck"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"

# Color output helpers
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info()  { printf "${GREEN}[shipcheck]${NC} %s\n" "$*"; }
warn()  { printf "${YELLOW}[shipcheck]${NC} %s\n" "$*"; }
error() { printf "${RED}[shipcheck]${NC} ERROR: %s\n" "$*" >&2; exit 1; }

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux*)  echo "Linux" ;;
        Darwin*) echo "Darwin" ;;
        CYGWIN*|MINGW*|MSYS*) echo "Windows" ;;
        *) error "Unsupported OS: $OS" ;;
    esac
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)   echo "x86_64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
}

# Get the latest version tag from GitHub
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -fsSL "$GITHUB_API" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "$GITHUB_API" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    else
        error "Neither curl nor wget found. Please install one and retry."
    fi

    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Check your internet connection."
    fi

    echo "$VERSION"
}

# Download a URL to a file
download() {
    URL="$1"
    DEST="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$URL" -o "$DEST"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$DEST" "$URL"
    else
        error "Neither curl nor wget found."
    fi
}

# Verify checksum
verify_checksum() {
    ARCHIVE="$1"
    CHECKSUM_FILE="$2"
    EXPECTED=$(grep "$(basename "$ARCHIVE")" "$CHECKSUM_FILE" | awk '{print $1}')

    if [ -z "$EXPECTED" ]; then
        warn "Could not find checksum for $(basename "$ARCHIVE"). Skipping verification."
        return 0
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL=$(sha256sum "$ARCHIVE" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL=$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')
    else
        warn "sha256sum/shasum not found. Skipping checksum verification."
        return 0
    fi

    if [ "$ACTUAL" != "$EXPECTED" ]; then
        error "Checksum mismatch! Expected: $EXPECTED  Got: $ACTUAL"
    fi

    info "Checksum verified."
}

main() {
    info "Installing shipcheck..."
    echo ""

    OS=$(detect_os)
    ARCH=$(detect_arch)

    # Use provided version or fetch latest
    VERSION="${VERSION:-$(get_latest_version)}"
    info "Version: $VERSION"
    info "OS: $OS / Arch: $ARCH"
    echo ""

    # Construct archive name (must match goreleaser naming)
    if [ "$OS" = "Windows" ]; then
        ARCHIVE_NAME="${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}.zip"
        EXT=".zip"
    else
        ARCHIVE_NAME="${BINARY_NAME}_${VERSION#v}_${OS}_${ARCH}.tar.gz"
        EXT=".tar.gz"
    fi

    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
    ARCHIVE_URL="${BASE_URL}/${ARCHIVE_NAME}"
    CHECKSUM_URL="${BASE_URL}/checksums.txt"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    ARCHIVE_PATH="${TMP_DIR}/${ARCHIVE_NAME}"
    CHECKSUM_PATH="${TMP_DIR}/checksums.txt"

    # Download archive and checksums
    info "Downloading ${ARCHIVE_URL}..."
    download "$ARCHIVE_URL" "$ARCHIVE_PATH" || error "Failed to download archive. Is v${VERSION} released?"

    info "Downloading checksums..."
    download "$CHECKSUM_URL" "$CHECKSUM_PATH" || warn "Could not download checksums file."

    # Verify checksum if we got it
    if [ -f "$CHECKSUM_PATH" ]; then
        verify_checksum "$ARCHIVE_PATH" "$CHECKSUM_PATH"
    fi

    # Extract archive
    info "Extracting..."
    if [ "$EXT" = ".zip" ]; then
        unzip -q "$ARCHIVE_PATH" -d "$TMP_DIR"
    else
        tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"
    fi

    # Find the binary
    EXTRACTED_BINARY=$(find "$TMP_DIR" -name "$BINARY_NAME" -not -path "*/\.*" | head -1)
    if [ -z "$EXTRACTED_BINARY" ]; then
        EXTRACTED_BINARY=$(find "$TMP_DIR" -name "${BINARY_NAME}.exe" | head -1)
    fi

    if [ -z "$EXTRACTED_BINARY" ]; then
        error "Could not find ${BINARY_NAME} binary in archive."
    fi

    # Install the binary
    INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}"

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        cp "$EXTRACTED_BINARY" "$INSTALL_PATH"
        chmod +x "$INSTALL_PATH"
    else
        info "Installing to ${INSTALL_DIR} (may prompt for password)..."
        sudo cp "$EXTRACTED_BINARY" "$INSTALL_PATH"
        sudo chmod +x "$INSTALL_PATH"
    fi

    echo ""
    info "shipcheck ${VERSION} installed to ${INSTALL_PATH}"
    echo ""
    info "Get started:"
    printf "  shipcheck --help\n"
    printf "  shipcheck scan\n"
    printf "  shipcheck init   # set up post-session hooks\n"
    echo ""
}

main "$@"
