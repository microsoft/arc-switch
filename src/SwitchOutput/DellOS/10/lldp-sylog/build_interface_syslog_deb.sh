#!/bin/bash
#
# build_interface_syslog_deb.sh
#
# This script builds a Debian package for the interface-syslog service.
# It creates a structured package with your application in /opt/microsoft,
# systemd unit files for service and timer in /etc/systemd/system,
# and includes proper post-install and post-removal scripts for service registration and cleanup.
#
# Usage:
#   ./build_interface_syslog_deb.sh          # Builds the package only.
#   ./build_interface_syslog_deb.sh --install  # Builds and installs the package.
#

# Package metadata
PACKAGE_NAME="interface-syslog"
VERSION="1.0"
ARCH="amd64"
MAINTAINER="Eric Marquez <emarq@microsoft.com>"
DESCRIPTION="Interface Syslog Service for Debian - LLDP Neighbor Service runs every 1 minute using systemd timer."

# Build directory for package content
BUILD_DIR="./${PACKAGE_NAME}_pkg"

# Clean any previous build
if [ -d "$BUILD_DIR" ]; then
    echo "Cleaning previous build directory: $BUILD_DIR"
    rm -rf "$BUILD_DIR"
fi

echo "Setting up package directory structure..."

# Create directory structure
mkdir -p "$BUILD_DIR/DEBIAN"
mkdir -p "$BUILD_DIR/opt/microsoft"
mkdir -p "$BUILD_DIR/etc/systemd/system"

# ==== Create DEBIAN/control file ====
cat > "$BUILD_DIR/DEBIAN/control" <<EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Section: base
Priority: optional
Architecture: ${ARCH}
Maintainer: ${MAINTAINER}
Description: ${DESCRIPTION}
EOF

# ==== Create DEBIAN/postinst script ====
cat > "$BUILD_DIR/DEBIAN/postinst" <<'EOF'
#!/bin/bash
# Post-install script: Reload systemd, enable and start the timer.

# Set ownership to root
chown -R root:root /opt/microsoft/interface-syslog

systemctl daemon-reload
systemctl enable interface-syslog.timer
systemctl start interface-syslog.timer
EOF
chmod 755 "$BUILD_DIR/DEBIAN/postinst"

# ==== Create DEBIAN/postrm script ====
cat > "$BUILD_DIR/DEBIAN/postrm" <<'EOF'
#!/bin/bash
# Post-removal script: Stop and disable the timer, remove unit files and reload systemd.
systemctl stop interface-syslog.timer
systemctl disable interface-syslog.timer
rm -f /etc/systemd/system/interface-syslog.service
rm -f /etc/systemd/system/interface-syslog.timer
systemctl daemon-reload
EOF
chmod 755 "$BUILD_DIR/DEBIAN/postrm"

# ==== Build the Go binary if it doesn't exist ====
if [ ! -f "./interface-syslog" ]; then
    echo "Building Go binary from lldp_syslog.go..."
    if [ ! -f "./lldp_syslog.go" ]; then
        echo "Error: lldp_syslog.go source file not found in the current directory."
        exit 1
    fi
    go build -o interface-syslog lldp_syslog.go
    if [ $? -ne 0 ]; then
        echo "Error: Failed to build Go binary."
        exit 1
    fi
fi

echo "Copying application binary to ${BUILD_DIR}/opt/microsoft/interface-syslog"
mkdir -p "$BUILD_DIR/opt/microsoft"
cp "./interface-syslog" "$BUILD_DIR/opt/microsoft/interface-syslog"
chmod 755 "$BUILD_DIR/opt/microsoft/interface-syslog"

# ==== Create systemd service file ====
cat > "$BUILD_DIR/etc/systemd/system/interface-syslog.service" <<EOF
[Unit]
Description=Interface Syslog Service - LLDP Neighbor Service
After=network.target
Wants=network.target

[Service]
Type=oneshot
ExecStart=/opt/microsoft/interface-syslog
TimeoutStartSec=45s
TimeoutStopSec=10s
User=root
Group=root
StandardOutput=journal
StandardError=journal
SyslogIdentifier=interface-syslog

[Install]
WantedBy=multi-user.target
EOF

# ==== Create systemd timer file ====
cat > "$BUILD_DIR/etc/systemd/system/interface-syslog.timer" <<EOF
[Unit]
Description=Run Interface Syslog Service every 1 minute

[Timer]
OnBootSec=1min
OnUnitActiveSec=1min
Unit=interface-syslog.service
Persistent=true
AccuracySec=1s

[Install]
WantedBy=timers.target
EOF

# ==== Build the Debian package ====
PACKAGE_FILE="${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
echo "Building Debian package ${PACKAGE_FILE}..."
# Use gzip compression for better compatibility with older dpkg versions
dpkg-deb --build --root-owner-group -Zgzip "$BUILD_DIR" "$PACKAGE_FILE"

if [ $? -ne 0 ]; then
    echo "Error building the Debian package."
    exit 1
fi

echo "Package built successfully: ${PACKAGE_FILE}"

# ==== Optionally install the package ====
if [[ "$1" == "--install" ]]; then
    echo "Installing the package..."
    sudo dpkg -i "$PACKAGE_FILE"
    if [ $? -eq 0 ]; then
        echo "Package installed successfully."
    else
        echo "Installation failed. Please check the errors above."
        exit 1
    fi
fi

echo "To uninstall the package in the future, run:"
echo "  sudo apt remove ${PACKAGE_NAME}"