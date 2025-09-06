#!/bin/bash

# Download and extract the latest parser release
# Requires: wget (preferred) or curl as fallback
# Usage: ./download-latest.sh [parser] [vendor] [version]
# Examples:
#   ./download-latest.sh                                      # Downloads cisco-nexus class_map_parser latest
#   ./download-latest.sh interface_counters_parser            # Downloads cisco-nexus interface_counters_parser latest
#   ./download-latest.sh class_map_parser dell-os10           # Downloads dell-os10 class_map_parser latest
#   ./download-latest.sh class_map_parser cisco-nexus v0.0.5-alpha.7  # Specific version

set -e

# Default values
PARSER_NAME=${1:-"class_map_parser"}
VENDOR=${2:-"cisco-nexus"}
VERSION=${3:-""}
REPO="microsoft/arc-switch"
PLATFORM="linux-amd64"  # Only platform available now

# Validate vendor
if [[ "$VENDOR" != "cisco-nexus" && "$VENDOR" != "dell-os10" ]]; then
    # Check if second argument looks like a version (starts with v)
    if [[ "$2" == v* ]]; then
        VERSION="$2"
        VENDOR="cisco-nexus"  # Default vendor
    elif [ -n "$2" ]; then
        echo "Error: Invalid vendor '$VENDOR'. Must be 'cisco-nexus' or 'dell-os10'"
        exit 1
    fi
fi

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

# Help function
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Switch Parser Download Script"
    echo "=============================="
    echo
    echo "Usage: ./download-latest.sh [parser] [vendor] [version]"
    echo
    echo "Arguments:"
    echo "  parser      Parser name (default: class_map_parser)"
    echo "  vendor      Vendor type: cisco-nexus or dell-os10 (default: cisco-nexus)"
    echo "  version     Specific version tag (default: auto-detect latest)"
    echo
    echo "Available Cisco Nexus parsers:"
    echo "  - class_map_parser"
    echo "  - interface_counters_parser"
    echo "  - inventory_parser"
    echo "  - ip_arp_parser"
    echo "  - ip_route_parser"
    echo "  - lldp_neighbor_parser"
    echo "  - mac_address_parser"
    echo "  - transceiver_parser"
    echo
    echo "Available Dell OS10 parsers:"
    echo "  (Check GitHub releases for available Dell parsers)"
    echo
    echo "Examples:"
    echo "  ./download-latest.sh                                      # cisco-nexus class_map_parser, latest"
    echo "  ./download-latest.sh interface_counters_parser            # cisco-nexus parser, latest"
    echo "  ./download-latest.sh ip_arp_parser dell-os10              # dell-os10 parser, latest"
    echo "  ./download-latest.sh class_map_parser cisco-nexus v0.0.5-alpha.7  # specific version"
    echo
    echo "Repository: https://github.com/${REPO}/releases"
    exit 0
fi

# If VERSION is not set, try to auto-detect
if [ -z "$VERSION" ]; then
    echo "Fetching latest release version from GitHub..."
    LATEST_VERSION=$(get_latest_version)

    if [ -n "$LATEST_VERSION" ] && [ "$LATEST_VERSION" != "null" ]; then
        VERSION="$LATEST_VERSION"
        echo "Latest version found: $VERSION"
    else
        VERSION="v0.0.5-alpha.7" # Fallback version
        echo "No releases found in repository, using fallback version: $VERSION"
        echo "Note: The repository may not have published releases yet."
        echo "Check: https://github.com/${REPO}/releases"
    fi
fi

echo "========================================="
echo "Parser: $PARSER_NAME"
echo "Vendor: $VENDOR"
echo "Version: $VERSION"
echo "Platform: $PLATFORM (Linux x86_64)"
echo "========================================="

# Build filename based on new naming convention
FILENAME="${VENDOR}-${PARSER_NAME}-${VERSION}-${PLATFORM}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

echo
echo "Downloading ${VENDOR} ${PARSER_NAME} ${VERSION}..."
echo "File: ${FILENAME}"
echo "URL: ${DOWNLOAD_URL}"
echo

# Check available download tools
DOWNLOAD_TOOL=""
if command -v wget &>/dev/null; then
    DOWNLOAD_TOOL="wget"
elif command -v curl &>/dev/null; then
    DOWNLOAD_TOOL="curl"
else
    echo "Error: Neither wget nor curl found."
    echo "Please install wget (recommended) or curl to download files."
    if [[ "$VENDOR" == "cisco-nexus" ]]; then
        echo "   On Nexus: feature bash-shell (if available)"
    elif [[ "$VENDOR" == "dell-os10" ]]; then
        echo "   On Dell OS10: sudo apt-get install wget"
    fi
    echo
    echo "Manual download URL:"
    echo "   ${DOWNLOAD_URL}"
    exit 1
fi

echo "Using ${DOWNLOAD_TOOL} for downloads"

# Download the package
echo "Downloading package..."
if [ "$DOWNLOAD_TOOL" = "wget" ]; then
    if ! wget "$DOWNLOAD_URL"; then
        echo
        echo "Error: Failed to download ${FILENAME}"
        echo "Please check:"
        echo "  1. Parser name is correct: ${PARSER_NAME}"
        echo "  2. Vendor is correct: ${VENDOR}"
        echo "  3. Version exists: ${VERSION}"
        echo "  4. View available releases at: https://github.com/${REPO}/releases"
        exit 1
    fi
else
    if ! curl -L -O "$DOWNLOAD_URL"; then
        echo
        echo "Error: Failed to download ${FILENAME}"
        echo "Please check the parser name, vendor, and version."
        exit 1
    fi
fi

# Extract the package
echo "Extracting package..."
tar -xzf "$FILENAME"

# Make executable
chmod +x "${PARSER_NAME}"
echo "Made binary executable"

# Automatically clean up archive file
rm -f "$FILENAME"
echo "Cleaned up archive file"

# Show what was extracted
echo
echo "Successfully downloaded and extracted!"
echo "Contents:"
ls -la "${PARSER_NAME}" commands.json *.txt *.md 2>/dev/null || ls -la

echo
echo "========================================="
echo "Ready to use!"
echo "========================================="
echo

# Provide usage examples based on vendor
if [[ "$VENDOR" == "cisco-nexus" ]]; then
    echo "For Cisco Nexus Switch:"
    echo "  # Copy to switch"
    echo "  scp ${PARSER_NAME} admin@nexus-switch:/bootflash/"
    echo
    echo "  # On the switch"
    echo "  cd /bootflash"
    echo "  ./${PARSER_NAME} -input show-output.txt -output result.json"
elif [[ "$VENDOR" == "dell-os10" ]]; then
    echo "For Dell OS10 Switch:"
    echo "  # Copy to switch"
    echo "  scp ${PARSER_NAME} admin@dell-switch:/home/admin/"
    echo
    echo "  # On the switch"
    echo "  cd /home/admin"
    echo "  ./${PARSER_NAME} -input show-output.txt -output result.json"
fi

echo
echo "General usage:"
echo "  ./${PARSER_NAME} --help"
echo "  ./${PARSER_NAME} -input input.txt -output output.json"
echo "  ./${PARSER_NAME} -commands commands.json -output output.json"

echo
echo "Download complete!"