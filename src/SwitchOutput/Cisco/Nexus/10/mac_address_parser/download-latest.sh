#!/bin/bash

# Download and extract the latest mac_address_parser release
# Requires: wget (preferred) or curl as fallback
# Usage: ./download-latest.sh [version] [platform]
# Example: ./download-latest.sh v0.0.3-alpha.1 linux-amd64

set -e

# Default values
VERSION=${1:-"v0.0.3-alpha.1"}
PLATFORM=${2:-"linux-amd64"}
REPO="microsoft/arc-switch"

# Determine file extension based on platform
if [[ "$PLATFORM" == *"windows"* ]]; then
    EXT="zip"
    EXTRACT_CMD="unzip"
else
    EXT="tar.gz"
    EXTRACT_CMD="tar -xzf"
fi

FILENAME="mac_address_parser-${VERSION}-${PLATFORM}.${EXT}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"
CHECKSUM_URL="${DOWNLOAD_URL}.sha256"

echo "🚀 Downloading mac_address_parser ${VERSION} for ${PLATFORM}..."
echo "📦 File: ${FILENAME}"
echo "🔗 URL: ${DOWNLOAD_URL}"
echo

# Check available download tools (prioritize wget since it's more commonly available)
DOWNLOAD_TOOL=""
if command -v wget &>/dev/null; then
    DOWNLOAD_TOOL="wget"
elif command -v curl &>/dev/null; then
    DOWNLOAD_TOOL="curl"
else
    echo "❌ Error: Neither wget nor curl found."
    echo "📋 Please install wget (recommended) or curl to download files."
    echo "   On Ubuntu/Debian: sudo apt-get install wget"
    echo "   On CentOS/RHEL: sudo yum install wget"
    echo "   On Alpine: apk add wget"
    echo
    echo "🔗 Manual download URL:"
    echo "   ${DOWNLOAD_URL}"
    echo "   ${CHECKSUM_URL}"
    exit 1
fi

echo "🔧 Using ${DOWNLOAD_TOOL} for downloads"

# Download the package
echo "⬇️  Downloading package..."
if [ "$DOWNLOAD_TOOL" = "wget" ]; then
    wget "$DOWNLOAD_URL"
else
    curl -L -O "$DOWNLOAD_URL"
fi

# Download checksum
echo "🔐 Downloading checksum..."
if [ "$DOWNLOAD_TOOL" = "wget" ]; then
    wget "$CHECKSUM_URL" 2>/dev/null || echo "⚠️  Checksum file not available"
else
    curl -L -O "$CHECKSUM_URL" 2>/dev/null || echo "⚠️  Checksum file not available"
fi

# Verify checksum if available
if [ -f "${FILENAME}.sha256" ]; then
    echo "✅ Verifying checksum..."
    if command -v sha256sum &>/dev/null; then
        sha256sum -c "${FILENAME}.sha256"
    elif command -v shasum &>/dev/null; then
        shasum -a 256 -c "${FILENAME}.sha256"
    else
        echo "⚠️  Warning: No checksum utility found, skipping verification"
    fi
else
    echo "⚠️  Warning: Checksum file not found, skipping verification"
fi

# Extract the package
echo "📦 Extracting package..."
if [[ "$EXT" == "zip" ]]; then
    if command -v unzip &>/dev/null; then
        unzip "$FILENAME"
    else
        echo "❌ Error: unzip not found. Please install unzip or extract manually."
        exit 1
    fi
else
    tar -xzf "$FILENAME"
fi

# Make executable (for Unix-like systems)
if [[ "$PLATFORM" != *"windows"* ]]; then
    chmod +x mac_address_parser
    echo "🔧 Made binary executable"
fi

# Show what was extracted
echo
echo "✅ Successfully downloaded and extracted!"
echo "📁 Contents:"
ls -la mac_address_parser* README.md mac-address-table-sample.json 2>/dev/null || ls -la

echo
echo "🎉 Ready to use! Try running:"
if [[ "$PLATFORM" == *"windows"* ]]; then
    echo "   ./mac_address_parser.exe --help"
else
    echo "   ./mac_address_parser --help"
fi

# Clean up archives (optional)
read -p "🗑️  Delete downloaded archives? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f "$FILENAME" "${FILENAME}.sha256"
    echo "🧹 Cleaned up download files"
fi
