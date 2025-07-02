#!/bin/bash

# Download script for Syslog Client
# This script downloads the latest pre-built binary for your platform

set -e

# Configuration
REPO="microsoft/arc-switch"
TOOL_NAME="syslog-client"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $OS in
    linux*)
        OS="linux"
        ;;
    *)
        print_error "Unsupported operating system: $OS"
        exit 1
        ;;
    esac

    case $ARCH in
    x86_64 | amd64)
        ARCH="amd64"
        ;;
    aarch64 | arm64)
        ARCH="arm64"
        ;;
    armv7l | armv6l)
        ARCH="arm"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    print_status "Detected platform: $PLATFORM"
}

# Function to get latest release version
get_latest_version() {
    print_status "Fetching latest release information..."

    # Try to get latest release from GitHub API
    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        print_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi

    if [ -z "$VERSION" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi

    print_status "Latest version: $VERSION"
}

# Function to download binary
download_binary() {
    BINARY_NAME="${TOOL_NAME}-${PLATFORM}"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY_NAME"

    print_status "Downloading from: $DOWNLOAD_URL"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    # Download binary
    if command -v curl >/dev/null 2>&1; then
        if ! curl -L -o "$TOOL_NAME" "$DOWNLOAD_URL"; then
            print_error "Failed to download binary"
            cleanup_and_exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -O "$TOOL_NAME" "$DOWNLOAD_URL"; then
            print_error "Failed to download binary"
            cleanup_and_exit 1
        fi
    fi

    # Make it executable
    chmod +x "$TOOL_NAME"

    print_status "Download completed"
}

# Function to install binary
install_binary() {
    if [ "$EUID" -eq 0 ]; then
        # Running as root, install system-wide
        print_status "Installing to $INSTALL_DIR (system-wide)"
        cp "$TOOL_NAME" "$INSTALL_DIR/"
        print_status "Installation completed"
    else
        # Not running as root, offer options
        echo
        print_warning "Not running as root. Choose an installation option:"
        echo "1) Install to $INSTALL_DIR (requires sudo)"
        echo "2) Install to ~/.local/bin (user only)"
        echo "3) Install to current directory"
        echo "4) Don't install, just download"
        echo
        read -p "Enter your choice (1-4): " choice

        case $choice in
        1)
            print_status "Installing to $INSTALL_DIR (requires sudo)"
            sudo cp "$TOOL_NAME" "$INSTALL_DIR/"
            print_status "Installation completed"
            ;;
        2)
            LOCAL_BIN="$HOME/.local/bin"
            mkdir -p "$LOCAL_BIN"
            print_status "Installing to $LOCAL_BIN"
            cp "$TOOL_NAME" "$LOCAL_BIN/"
            print_status "Installation completed"
            print_warning "Make sure $LOCAL_BIN is in your PATH"
            ;;
        3)
            print_status "Installing to current directory"
            cp "$TOOL_NAME" "$OLDPWD/"
            print_status "Binary copied to: $OLDPWD/$TOOL_NAME"
            ;;
        4)
            print_status "Binary available at: $TMP_DIR/$TOOL_NAME"
            print_warning "Remember to copy it to your desired location"
            return
            ;;
        *)
            print_error "Invalid choice"
            cleanup_and_exit 1
            ;;
        esac
    fi
}

# Function to verify installation
verify_installation() {
    if command -v "$TOOL_NAME" >/dev/null 2>&1; then
        print_status "Verification successful!"
        echo
        "$TOOL_NAME" -version
    else
        print_warning "Tool installed but not found in PATH"
        print_warning "You may need to add the installation directory to your PATH"
    fi
}

# Function to cleanup and exit
cleanup_and_exit() {
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
    exit "${1:-0}"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Download and install the Syslog Client"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -v, --version  Download a specific version"
    echo "  --no-install   Download only, don't install"
    echo
    echo "Examples:"
    echo "  $0                    # Download and install latest version"
    echo "  $0 -v v1.0.0         # Download and install specific version"
    echo "  $0 --no-install      # Download only"
}

# Parse command line arguments
NO_INSTALL=false
SPECIFIC_VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
    -h | --help)
        show_usage
        exit 0
        ;;
    -v | --version)
        SPECIFIC_VERSION="$2"
        shift 2
        ;;
    --no-install)
        NO_INSTALL=true
        shift
        ;;
    *)
        print_error "Unknown option: $1"
        show_usage
        exit 1
        ;;
    esac
done

# Main execution
main() {
    echo "Syslog Client - Download Script"
    echo "==============================="
    echo

    # Check prerequisites
    if ! command -v uname >/dev/null 2>&1; then
        print_error "uname command not found"
        exit 1
    fi

    detect_platform

    if [ -n "$SPECIFIC_VERSION" ]; then
        VERSION="$SPECIFIC_VERSION"
        print_status "Using specified version: $VERSION"
    else
        get_latest_version
    fi

    download_binary

    if [ "$NO_INSTALL" = false ]; then
        install_binary
        verify_installation
    else
        print_status "Download completed. Binary available at: $TMP_DIR/$TOOL_NAME"
        print_warning "Remember to copy it to your desired location"
    fi

    echo
    print_status "Done! Run '$TOOL_NAME --help' for usage information"
}

# Trap to cleanup on exit
trap cleanup_and_exit EXIT

# Run main function
main "$@"
