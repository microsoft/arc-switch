#!/bin/bash
#
# build_lldpsyslog_deb.sh
#
# This script builds a Debian package for the lldpsyslog service.
# It creates a structured package with your application in /opt/microsoft/lldpsyslog,
# systemd unit files for service and timer in /etc/systemd/system,
# and includes proper post-install and post-removal scripts for service registration and cleanup.
#
# Usage:
#   ./build_lldpsyslog_deb.sh          # Builds the package only.
#   ./build_lldpsyslog_deb.sh --install  # Builds and installs the package.
#

# Package metadata
PACKAGE_NAME="lldpsyslog"
VERSION="1.0"
ARCH="amd64"
MAINTAINER="Eric Marquez <emarq@microsoft.com>"
DESCRIPTION="LLDP Neighbor Service for Debian - Runs every 1 minute using systemd timer."

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
mkdir -p "$BUILD_DIR/opt/microsoft/${PACKAGE_NAME}"
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
# Post-install script: Create user, reload systemd, enable and start the timer.

# Create system user for the service
if ! id "lldpsyslog" &>/dev/null; then
    useradd --system --no-create-home --shell /bin/false --home-dir /opt/microsoft/lldpsyslog lldpsyslog
fi

# Set ownership
chown -R lldpsyslog:lldpsyslog /opt/microsoft/lldpsyslog

systemctl daemon-reload
systemctl enable lldpsyslog.timer
systemctl start lldpsyslog.timer
EOF
chmod 755 "$BUILD_DIR/DEBIAN/postinst"

# ==== Create DEBIAN/postrm script ====
cat > "$BUILD_DIR/DEBIAN/postrm" <<'EOF'
#!/bin/bash
# Post-removal script: Stop and disable the timer, remove unit files and reload systemd.
systemctl stop lldpsyslog.timer
systemctl disable lldpsyslog.timer
rm -f /etc/systemd/system/lldpsyslog.service
rm -f /etc/systemd/system/lldpsyslog.timer
systemctl daemon-reload

# Remove user on purge
if [ "$1" = "purge" ]; then
    if id "lldpsyslog" &>/dev/null; then
        userdel lldpsyslog
    fi
fi
EOF
chmod 755 "$BUILD_DIR/DEBIAN/postrm"

# ==== Copy application binary ====
# Assumes the lldpsyslog binary is in the same directory as this script.
if [ ! -f "./lldpsyslog" ]; then
    echo "Error: lldpsyslog binary not found in the current directory."
    exit 1
fi

echo "Copying application binary to ${BUILD_DIR}/opt/microsoft/${PACKAGE_NAME}/lldpsyslog"
cp "./lldpsyslog" "$BUILD_DIR/opt/microsoft/${PACKAGE_NAME}/lldpsyslog"
chmod 755 "$BUILD_DIR/opt/microsoft/${PACKAGE_NAME}/lldpsyslog"

# ==== Create systemd service file ====
cat > "$BUILD_DIR/etc/systemd/system/lldpsyslog.service" <<EOF
[Unit]
Description=LLDP Neighbor Service
After=network.target
Wants=network.target

[Service]
Type=oneshot
ExecStart=/opt/microsoft/${PACKAGE_NAME}/lldpsyslog
TimeoutStartSec=45s
TimeoutStopSec=10s
User=lldpsyslog
Group=lldpsyslog
StandardOutput=journal
StandardError=journal
SyslogIdentifier=lldpsyslog

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log
PrivateTmp=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true

[Install]
WantedBy=multi-user.target
EOF

# ==== Create systemd timer file ====
cat > "$BUILD_DIR/etc/systemd/system/lldpsyslog.timer" <<EOF
[Unit]
Description=Run LLDP Neighbor every 1 minute

[Timer]
OnBootSec=1min
OnUnitActiveSec=1min
Unit=lldpsyslog.service
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