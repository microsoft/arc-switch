# Dell OS10 Version Parser

This tool parses the output of the `show version` command from Dell OS10 switches and converts it to structured JSON format aligned with Cisco Nexus parser standards.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Standardized JSON Output**: Uses Cisco-aligned JSON keys for cross-platform compatibility
- **Comprehensive Version Data**: Captures OS name, version, build information, system type, architecture, and uptime
- **Uptime Breakdown**: Parses uptime into weeks, days, hours, minutes, and seconds
- **Error Handling**: Robust parsing with proper error handling
- **CLI Integration**: Uses `clish` for direct switch communication

## Installation

### Build from Source

```bash
# Navigate to the parser directory
cd src/SwitchOutput/DellOS/10/version_parser

# Build the binary
go build -o dell_version_parser
```

## Usage

### Parse from Input File

```bash
# Parse version data from a text file
./dell_version_parser -input show-version.txt -output output.json

# Parse and output to stdout
./dell_version_parser -input show-version.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./dell_version_parser -commands commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show version` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)

## Input Format

The tool expects Dell OS10 `show version` output:

```text
rr1-s46-r06-5248hl-6-1a# show version
Dell SmartFabric OS10 Enterprise
Copyright (c) 1999-2025 by Dell Inc. All Rights Reserved.
OS Version: 10.6.0.5
Build Version: 10.6.0.5.139
Build Time: 2025-07-02T19:13:52+0000
System Type: S5248F-ON
Architecture: x86_64
Up Time: 6 weeks 1 day 17:55:03
rr1-s46-r06-5248hl-6-1a#
```

## Output Format

The parser outputs JSON with a standardized structure using Cisco-aligned keys for cross-platform compatibility:

```json
{
  "data_type": "dell_os10_version",
  "timestamp": "2026-01-27T21:30:00Z",
  "date": "2026-01-27",
  "message": {
    "nxos_version": "Dell SmartFabric OS10 Enterprise",
    "bios_version": "10.6.0.5",
    "nxos_compile_time": "10.6.0.5.139",
    "bios_compile_time": "2025-07-02T19:13:52+0000",
    "chassis_id": "S5248F-ON",
    "cpu_name": "x86_64",
    "boot_mode": "6 weeks 1 day 17:55:03",
    "device_name": "rr1-s46-r06-5248hl-6-1a",
    "kernel_uptime": {
      "weeks": 6,
      "days": 1,
      "hours": 17,
      "minutes": 55,
      "seconds": 3
    }
  }
}
```

### Key Name Alignment with Cisco

To enable cross-platform analytics and unified monitoring, this parser uses Cisco-aligned JSON keys:

| Dell Data | JSON Key | Cisco Equivalent |
|-----------|----------|------------------|
| OS Name | `nxos_version` | NX-OS version string |
| OS Version | `bios_version` | BIOS version |
| Build Version | `nxos_compile_time` | NX-OS compile time |
| Build Time | `bios_compile_time` | BIOS compile time |
| System Type | `chassis_id` | Chassis ID |
| Architecture | `cpu_name` | CPU name |
| Up Time | `boot_mode` | Boot mode/uptime string |
| Device Name | `device_name` | Device hostname |
| Uptime Breakdown | `kernel_uptime` | Kernel uptime structure |

### Required Fields

- `data_type`: Always "dell_os10_version"
- `timestamp`: Processing timestamp in ISO 8601 format
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all version data

### Message Fields

The `message` field contains all version-specific data using Cisco-aligned keys:

- `nxos_version`: Dell SmartFabric OS name
- `bios_version`: OS version number
- `nxos_compile_time`: Build version number
- `bios_compile_time`: Build timestamp
- `chassis_id`: System/chassis type
- `cpu_name`: CPU architecture
- `boot_mode`: Uptime string
- `device_name`: Switch hostname
- `kernel_uptime`: Structured uptime breakdown
  - `weeks`: Number of weeks in uptime
  - `days`: Number of days in uptime
  - `hours`: Number of hours in uptime
  - `minutes`: Number of minutes in uptime
  - `seconds`: Number of seconds in uptime

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "version"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "version",
      "command": "show version"
    }
  ]
}
```

## Testing

Run the test suite to verify the parser functionality:

```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestOSVersion
```

The test suite covers:
- OS name parsing
- Version number extraction
- Build information parsing
- System type and architecture
- Uptime parsing (multiple formats)
- Device name extraction
- JSON serialization/deserialization
- UnifiedParser interface implementation
- Cisco key alignment validation

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed version output
- Missing or incomplete data
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)

## Requirements

- Go 1.21 or later
- For direct switch access: `clish` command-line tool available on Dell OS10 switches

## Use Cases

This parser is useful for:

1. **Version Tracking**: Monitor OS versions across Dell switch fleet
2. **Inventory Management**: Track system types and configurations
3. **Uptime Monitoring**: Analyze switch stability and availability
4. **Compliance Reporting**: Ensure switches run approved OS versions
5. **Cross-Platform Analytics**: Unified monitoring with Cisco switches using aligned JSON keys
6. **Build Version Tracking**: Track build versions for patch management

## Cross-Platform Compatibility

By using Cisco-aligned JSON keys, this parser enables:

- Unified monitoring dashboards across Dell and Cisco switches
- Common KQL queries for both platforms
- Simplified integration with existing Cisco-based analytics
- Consistent data models for multi-vendor environments
