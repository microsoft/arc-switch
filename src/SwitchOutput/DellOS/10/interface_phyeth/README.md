# Interface Physical Ethernet Service

This tool collects SFP (Small Form-factor Pluggable) data from Dell OS10 switches using the `show interface phy-eth` command and logs the information to syslog. The service runs every 60 seconds via systemd timer.

## Features

- Parses Dell OS10 interface physical ethernet output
- Converts data to structured JSON format
- Logs each SFP interface separately to syslog with tag "sfpdata"
- Runs as a systemd service with automatic timer scheduling
- Supports test mode for validation

## Manual Usage

```bash
# Run with input file (test mode)
./interface_phyeth -test -inputfile=show_interface_phy-eth.txt

# Run directly against switch (requires clish command access)
./interface_phyeth
```

## Debian Package Installation

### Prerequisites

- Debian/Ubuntu Linux system
- systemd (for service management)
- Root/sudo access for installation

### Download and Install

1. Download the latest .deb package from the releases page
2. Install the package:

```bash
# Install the package
sudo dpkg -i interface-phyeth_<version>_amd64.deb

# Verify installation
sudo systemctl status interface-phyeth.timer
sudo systemctl status interface-phyeth.service
```

### Post-Installation

After installation, the service is automatically:
- Installed to `/opt/microsoft/interface-phyeth/interface_phyeth`
- Configured as systemd service and timer
- Enabled and started automatically
- Scheduled to run every 60 seconds

### Service Management

```bash
# Check timer status
sudo systemctl status interface-phyeth.timer

# Check service status
sudo systemctl status interface-phyeth.service

# View recent logs
sudo journalctl -u interface-phyeth.service -f

# Start/stop the timer
sudo systemctl start interface-phyeth.timer
sudo systemctl stop interface-phyeth.timer

# Enable/disable automatic startup
sudo systemctl enable interface-phyeth.timer
sudo systemctl disable interface-phyeth.timer
```

### Syslog Output

The service logs SFP data to syslog with:
- Facility: `local0.info`
- Tag: `sfpdata`
- Format: JSON structure for each interface

Example syslog entry:
```
interface-phyeth: {"sfp":"1/1/1","id":3,"ext_id":0,"connector":33,...}
```

## Removal Instructions

### Uninstall Package

```bash
# Remove the package (keeps configuration)
sudo apt remove interface-phyeth

# Remove package and configuration files
sudo apt purge interface-phyeth

# Clean up any remaining dependencies
sudo apt autoremove
```

### Manual Cleanup (if needed)

If manual cleanup is required:

```bash
# Stop and disable services
sudo systemctl stop interface-phyeth.timer
sudo systemctl disable interface-phyeth.timer

# Remove systemd unit files
sudo rm -f /etc/systemd/system/interface-phyeth.service
sudo rm -f /etc/systemd/system/interface-phyeth.timer

# Remove application directory
sudo rm -rf /opt/microsoft/interface-phyeth

# Reload systemd
sudo systemctl daemon-reload
```

## Build Instructions

To build your own Debian package:

```bash
# Ensure Go is installed
go version

# Build the package
chmod +x build_interface_phyeth_deb.sh
./build_interface_phyeth_deb.sh

# Or build and install in one step
./build_interface_phyeth_deb.sh --install
```

## Troubleshooting

### Check Service Status
```bash
sudo systemctl status interface-phyeth.timer
sudo systemctl status interface-phyeth.service
```

### View Logs
```bash
# Service logs
sudo journalctl -u interface-phyeth.service -n 50

# Follow logs in real-time
sudo journalctl -u interface-phyeth.service -f

# System syslog for SFP data
sudo grep "sfpdata" /var/log/syslog
```

### Manual Test
```bash
# Test with sample data
sudo /opt/microsoft/interface-phyeth/interface_phyeth -test -inputfile=/path/to/sample/data.txt
```

## Configuration

The service is configured through systemd unit files:
- Service: `/etc/systemd/system/interface-phyeth.service`
- Timer: `/etc/systemd/system/interface-phyeth.timer`

To modify the execution interval, edit the timer file and reload systemd:

```bash
sudo systemctl edit interface-phyeth.timer
sudo systemctl daemon-reload
sudo systemctl restart interface-phyeth.timer
```