#!/usr/bin/env bash
set -e

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# New logging functions
info() { echo "[INFO] $1"; }
error() { echo "[ERROR] $1"; exit 1; }

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi

# Updated to deploy on GitHub repository fdorantesm/go-http-cannon
BIN_NAME="cannon"  # Updated binary name to match repository
BASE_URL="https://github.com/fdorantesm/go-http-cannon/releases/latest/download"
DOWNLOAD_URL="$BASE_URL/${BIN_NAME}-${OS}-${ARCH}"

info "Detected OS: $OS, ARCH: $ARCH"
info "Starting installer..."

# Check that either curl or wget is available
if command_exists curl; then
    DOWNLOAD_TOOL="curl -L"
elif command_exists wget; then
    DOWNLOAD_TOOL="wget -qO-"
else
    error "curl or wget is required to continue."
fi

# Installation directory (modify as needed)
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    error "No write permissions in $INSTALL_DIR. Exiting."
fi

TARGET="$INSTALL_DIR/$BIN_NAME"
if [ -f "$TARGET" ]; then
    info "$BIN_NAME is already installed, updating..."
fi

info "Downloading ${BIN_NAME} from $DOWNLOAD_URL..."
$DOWNLOAD_TOOL "$DOWNLOAD_URL" -o "$TARGET"

# Make the downloaded binary executable
chmod +x "$TARGET"

info "Installation complete: $TARGET"