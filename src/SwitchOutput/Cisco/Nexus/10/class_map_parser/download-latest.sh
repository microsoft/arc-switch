#!/bin/bash

# Download and extract the latest class_map_parser release
# Requires: wget (preferred) or curl as fallback
# Usage: ./download-latest.sh [platform] [version]
#        ./download-latest.sh [version] (if version starts with 'v')
# Examples:
#   ./download-latest.sh                    # Auto-detect latest version, linux-amd64
#   ./download-latest.sh windows-amd64      # Auto-detect latest version, windows-amd64
#   ./download-latest.sh v0.0.6-alpha.1     # Specific version, linux-amd64
#   ./download-latest.sh linux-arm64 v0.0.6-alpha.1 # Specific platform and version

set -e

# Default values
PLATFORM=${1:-"linux-amd64"}
REPO="microsoft/arc-switch"

# Function to get the latest release version from GitHub API
get_latest_version() {
    local latest_api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local all_releases_api_url="https://api.github.com/repos/${REPO}/releases"

    # First try to get the latest published release
    if command -v curl &>/dev/null; then
        LATEST=$(curl -s "$latest_api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -n "$LATEST" ]; then
            echo "$LATEST"
            return
        fi
        # If no published release, get the first from all releases (including pre-releases)
        curl -s "$all_releases_api_url" | grep '"tag_name":' | head -1 | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget &>/dev/null; then
        LATEST=$(wget -qO- "$latest_api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -n "$LATEST" ]; then
            echo "$LATEST"
            return
        fi
        # If no published release, get the first from all releases (including pre-releases)
        wget -qO- "$all_releases_api_url" | grep '"tag_name":' | head -1 | sed -E 's/.*"([^"]+)".*/\1/'
    else
        echo ""
    fi
}

# Determine version to use
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Class Map Parser Download Script"
    echo "================================"
    echo
    echo "Usage: ./download-latest.sh [platform] [version]"
    echo "       ./download-latest.sh [version] (if version starts with 'v')"
    echo
    echo "Arguments:"
    echo "  platform    Target platform (default: linux-amd64)"
    echo "  version     Specific version tag (default: auto-detect latest)"
    echo
    echo "Examples:"
    echo "  ./download-latest.sh                    # Auto-detect latest version, linux-amd64"
    echo "  ./download-latest.sh windows-amd64      # Auto-detect latest version, windows-amd64"
    echo "  ./download-latest.sh v0.0.6-alpha.1     # Specific version, linux-amd64"
    echo "  ./download-latest.sh linux-arm64 v0.0.6-alpha.1 # Specific platform and version"
    echo
    echo "Supported platforms:"
    echo "  linux-amd64, linux-arm64, windows-amd64, darwin-amd64, darwin-arm64"
    echo
    echo "Repository: ${REPO}"
    exit 0
elif [ -n "$2" ]; then
    # Two arguments provided: platform and version
    VERSION="$2"
    # PLATFORM already set from $1
elif [ -n "$1" ] && [[ "$1" == v* ]]; then
    # First argument looks like a version tag, use default platform
    VERSION="$1"
    PLATFORM="linux-amd64"
elif [ -n "$1" ]; then
    # First argument provided but doesn't look like version, treat as platform
    # PLATFORM already set from $1, will auto-detect version
    true # No-op, just for clarity
fi

# If VERSION is not set by now, try to auto-detect
if [ -z "$VERSION" ]; then
    # Try to get latest version from GitHub API
    echo "Fetching latest release version from GitHub..."
    LATEST_VERSION=$(get_latest_version)

    if [ -n "$LATEST_VERSION" ] && [ "$LATEST_VERSION" != "null" ]; then
        VERSION="$LATEST_VERSION"
        echo "Latest version found: $VERSION"
    else
        VERSION="v0.0.6-alpha.1" # Fallback version
        echo "No releases found in repository, using fallback version: $VERSION"
        echo "Note: The repository may not have published releases yet."
        echo "Check: https://github.com/${REPO}/releases"
    fi
fi

echo "Using version: $VERSION for platform: $PLATFORM"

# Determine file extension based on platform
if [[ "$PLATFORM" == *"windows"* ]]; then
    EXT="zip"
    EXTRACT_CMD="unzip"
else
    EXT="tar.gz"
    EXTRACT_CMD="tar -xzf"
fi

FILENAME="class_map_parser-${VERSION}-${PLATFORM}.${EXT}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"
CHECKSUM_URL="${DOWNLOAD_URL}.sha256"

echo "Downloading class_map_parser ${VERSION} for ${PLATFORM}..."
echo "File: ${FILENAME}"
echo "URL: ${DOWNLOAD_URL}"
echo

# Check available download tools (prioritize wget since it's more commonly available)
DOWNLOAD_TOOL=""
if command -v wget &>/dev/null; then
    DOWNLOAD_TOOL="wget"
elif command -v curl &>/dev/null; then
    DOWNLOAD_TOOL="curl"
else
    echo "Error: Neither wget nor curl found."
    echo "Please install wget (recommended) or curl to download files."
    echo "   On Ubuntu/Debian: sudo apt-get install wget"
    echo "   On CentOS/RHEL: sudo yum install wget"
    echo "   On Alpine: apk add wget"
    echo
    echo "Manual download URL:"
    echo "   ${DOWNLOAD_URL}"
    echo "   ${CHECKSUM_URL}"
    exit 1
fi

echo "Using ${DOWNLOAD_TOOL} for downloads"

# Download the package
echo "Downloading package..."
if [ "$DOWNLOAD_TOOL" = "wget" ]; then
    wget "$DOWNLOAD_URL"
else
    curl -L -O "$DOWNLOAD_URL"
fi

# Download checksum
echo "Downloading checksum..."
if [ "$DOWNLOAD_TOOL" = "wget" ]; then
    wget "$CHECKSUM_URL" 2>/dev/null || echo "Checksum file not available"
else
    curl -L -O "$CHECKSUM_URL" 2>/dev/null || echo "Checksum file not available"
fi

# Verify checksum if available
if [ -f "${FILENAME}.sha256" ]; then
    echo "Verifying checksum..."
    if command -v sha256sum &>/dev/null; then
        sha256sum -c "${FILENAME}.sha256"
    elif command -v shasum &>/dev/null; then
        shasum -a 256 -c "${FILENAME}.sha256"
    else
        echo "Warning: No checksum utility found, skipping verification"
    fi
else
    echo "Warning: Checksum file not found, skipping verification"
fi

# Extract the package
echo "Extracting package..."
if [[ "$EXT" == "zip" ]]; then
    if command -v unzip &>/dev/null; then
        unzip "$FILENAME"
    else
        echo "Error: unzip not found. Please install unzip or extract manually."
        exit 1
    fi
else
    tar -xzf "$FILENAME"
fi

# Make executable (for Unix-like systems)
if [[ "$PLATFORM" != *"windows"* ]]; then
    chmod +x class_map_parser
    echo "Made binary executable"
fi

# Show what was extracted
echo
echo "Successfully downloaded and extracted!"
echo "Contents:"
ls -la class_map_parser* README.md class-map-sample.json 2>/dev/null || ls -la

echo
echo "Ready to use! Try running:"
if [[ "$PLATFORM" == *"windows"* ]]; then
    echo "   ./class_map_parser.exe --help"
    echo "   ./class_map_parser.exe -input show-class-map.txt"
else
    echo "   ./class_map_parser --help"
    echo "   ./class_map_parser -input show-class-map.txt"
fi

echo
echo "Usage examples:"
echo "   # Parse from input file to JSON output"
echo "   ./class_map_parser -input show-class-map.txt -output output.json"
echo
echo "   # Get data directly from switch using commands.json"
echo "   ./class_map_parser -commands commands.json -output output.json"
echo
echo "   # Parse from input file and output to stdout"
echo "   ./class_map_parser -input show-class-map.txt"

# Clean up archives (optional)
read -p "Delete downloaded archives? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f "$FILENAME" "${FILENAME}.sha256"
    echo "Cleaned up download files"
fi