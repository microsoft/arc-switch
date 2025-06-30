# Syslog Client

A robust Go-based tool for sending JSON entries to Linux syslog systems. This tool is designed to work across Debian and RPM-based distributions and handles JSON data with standardized common fields.

## Features

- **Cross-platform Linux support**: Works on Debian, Ubuntu, CentOS, RHEL, and other Linux distributions
- **Flexible input methods**: Process single entries or files with multiple entries
- **JSON-native logging**: Complete JSON objects are written to syslog as-is, preserving all data
- **Robust validation**: Ensures all entries contain required common fields
- **Configurable syslog settings**: Support for various facilities, priorities, and tags
- **Entry size management**: Automatic truncation of oversized entries
- **Test mode**: Validate entries without writing to syslog
- **Detailed statistics**: Track processing success/failure rates
- **Systemd detection**: Automatically detects and works with systemd-journald

## Installation

### Prerequisites

- Go 1.21 or later
- Linux operating system (Debian/Ubuntu/CentOS/RHEL/etc.)
- Syslog daemon (rsyslog, syslog-ng, or systemd-journald)

### Build from Source

```bash
cd /workspaces/arc-switch/src/SyslogTools/syslog-client
go mod tidy
go build -o syslog-client main.go
```

### Install System-wide

```bash
# Build the binary
go build -o syslog-client main.go

# Copy to system path (requires sudo)
sudo cp syslog-client /usr/local/bin/

# Make it executable
sudo chmod +x /usr/local/bin/syslog-client
```

## Usage

### Command Line Options

```text
-file string
      Path to file containing JSON entries (one per line)
-entry string
      Single JSON entry to log
-tag string
      Syslog tag to use (default "syslog-client")
-facility string
      Syslog facility (default "local0")
-priority string
      Syslog priority level (default "info")
-max-size int
      Maximum entry size in bytes (default 4096)
-verbose
      Enable verbose output
-test
      Test mode - validate entries but don't write to syslog
-version
      Show version information
-help
      Show help information
```

### Required JSON Fields

All JSON entries **must** contain these three required fields:

- `data_type`: String identifying the type of data (can also serve as tag identifier for the entry)
- `timestamp`: Full timestamp with date and time
- `date`: Date in MM/DD/YYYY format

Additional fields are preserved and logged as-is.

### Examples

#### Process a Single JSON Entry

```bash
./syslog-client -entry '{"data_type":"test_entry","timestamp":"06/24/2025 10:00:01 AM","date":"06/24/2025","message":"Test message 1","priority":"info"}'
```

#### Process a File with Multiple Entries

```bash
./syslog-client -file /path/to/data.json -tag "network-data" -verbose
```

#### Test Mode (Validation Only)

```bash
./syslog-client -file /path/to/data.json -test -verbose
```

#### Custom Syslog Configuration

```bash
./syslog-client -file data.json -facility local1 -priority warning -tag "my-app" -max-size 8192
```

#### Process MAC Address Table Data

```bash
./syslog-client -file /workspaces/arc-switch/src/SwitchOutput/Cisco/Nexus/10/mac_address_parser/mac-address-table-sample.json -tag "cisco-mac-table" -verbose
```

## Input File Format

The tool expects JSON entries in **JSON Lines** format (one JSON object per line). The complete JSON objects are written to syslog as-is:

```json
{"data_type":"cisco_nexus_mac_table","timestamp":"06/24/2025 10:00:01 AM","date":"06/24/2025","vlan":"7","mac_address":"02ec.a004.0000","port":"Eth1/1"}
{"data_type":"cisco_nexus_mac_table","timestamp":"06/24/2025 10:00:02 AM","date":"06/24/2025","vlan":"125","mac_address":"0015.5d53.7e5b","port":"Po102"}
{"data_type":"interface_status","timestamp":"06/24/2025 10:00:03 AM","date":"06/24/2025","interface":"eth0","status":"up","speed":"1000Mbps"}
```

## Syslog Configuration

### Supported Facilities

- `local0` through `local7` (default: `local0`)
- `user`, `mail`, `daemon`, `auth`, `syslog`, `lpr`, `news`, `uucp`, `cron`, `authpriv`, `ftp`

### Supported Priority Levels

- `emerg`/`emergency` - System is unusable
- `alert` - Action must be taken immediately
- `crit`/`critical` - Critical conditions
- `err`/`error` - Error conditions
- `warning`/`warn` - Warning conditions
- `notice` - Normal but significant condition
- `info` - Informational messages (default)
- `debug` - Debug-level messages

### Example Syslog Output

When using facility `local0` and tag `syslog-client`, the complete JSON entries appear in syslog as:

```text
Jun 24 10:00:01 hostname syslog-client: {"data_type":"test_entry","timestamp":"06/24/2025 10:00:01 AM","date":"06/24/2025","message":"Test message 1","priority":"info"}
Jun 24 10:00:02 hostname syslog-client: {"data_type":"cisco_nexus_mac_table","timestamp":"06/24/2025 10:00:02 AM","date":"06/24/2025","vlan":"7","mac_address":"02ec.a004.0000","port":"Eth1/1"}
```

## Configuration for Different Distributions

### Debian/Ubuntu (rsyslog)

Add to `/etc/rsyslog.d/50-syslog-client.conf`:

```text
# Log syslog-client entries to separate file
local0.*    /var/log/syslog-client.log
& stop
```

Restart rsyslog:

```bash
sudo systemctl restart rsyslog
```

### CentOS/RHEL (rsyslog)

Add to `/etc/rsyslog.conf`:

```text
# Syslog client entries
local0.*    /var/log/syslog-client.log
```

Restart rsyslog:

```bash
sudo systemctl restart rsyslog
```

### systemd-journald

View logs using journalctl:

```bash
# View all entries with specific tag
journalctl -t syslog-client

# Follow live entries
journalctl -t syslog-client -f

# Filter by time
journalctl -t syslog-client --since "1 hour ago"
```

## Error Handling

The tool provides comprehensive error handling:

- **JSON validation**: Ensures all entries are valid JSON
- **Required field validation**: Checks for `data_type`, `timestamp`, and `date` fields
- **Data type identification**: The `data_type` field serves dual purpose as data identifier and potential tag identifier
- **Size management**: Automatically truncates entries that exceed the maximum size
- **Statistics tracking**: Reports success/failure rates
- **Graceful failure**: Continues processing even if individual entries fail

## Performance Considerations

- **Maximum entry size**: Default 4096 bytes (configurable)
- **Memory efficient**: Processes files line-by-line without loading entire file into memory
- **Concurrent safe**: Can be used safely in concurrent environments
- **Non-blocking**: Doesn't block on syslog operations

## Integration Examples

### Integration with MAC Address Parser

```bash
# Process MAC address table data and log to syslog
/path/to/mac_address_parser -input show-mac-address-table.txt | \
./syslog-client -file - -tag "cisco-mac-table"
```

### Integration with Monitoring Scripts

```bash
#!/bin/bash
# Monitor script that generates JSON and logs to syslog

# Generate JSON data with data_type serving as tag identifier
JSON_ENTRY='{"data_type":"system_health_monitor","timestamp":"'$(date +'%m/%d/%Y %I:%M:%S %p')'","date":"'$(date +'%m/%d/%Y')'","cpu_usage":"25%","memory_usage":"60%"}'

# Log to syslog
./syslog-client -entry "$JSON_ENTRY" -tag "system-monitor" -facility local1
```

### Systemd Service Example

Create `/etc/systemd/system/json-logger.service`:

```ini
[Unit]
Description=Syslog Client JSON Logger Service
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/syslog-client -file /var/log/json-input.log -tag "json-service"
User=syslog
Group=syslog

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

### Common Issues

1. **Permission denied**: Ensure the user has permission to write to syslog
2. **Invalid JSON**: Check that all entries are properly formatted JSON
3. **Missing fields**: Verify all entries contain required fields (`data_type`, `timestamp`, `date`)
4. **Syslog not receiving entries**: Check syslog daemon configuration and ensure it's running

### Debug Mode

Use verbose mode to get detailed information:

```bash
./syslog-client -file data.json -verbose
```

### Test Mode

Validate entries without writing to syslog:

```bash
./syslog-client -file data.json -test -verbose
```

## Contributing

This tool follows the repository's Go best practices. See the main project's `CONTRIBUTING.md` and `copilot-instructions.md` for guidelines.

## License

This project is licensed under the same license as the parent arc-switch repository.

## Version History

- **v1.0.0**: Initial release with support for JSON logging to Linux syslog
