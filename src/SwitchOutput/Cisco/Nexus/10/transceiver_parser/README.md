# Cisco Nexus Transceiver Parser

This tool parses Cisco Nexus `show interface transceiver details` output and converts it to structured JSON format for optical monitoring and inventory management.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Transceiver Data**: Captures all transceiver information including type, manufacturer, serial numbers, and specifications
- **DOM Support**: Parses Digital Optical Monitoring data with thresholds and alarm status
- **Multi-vendor Support**: Handles various transceiver types (SFP, QSFP, copper, optical)
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each transceiver produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of absent transceivers and DOM data
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

```bash
# Build the binary
go build -o transceiver_parser transceiver_parser.go
```

## Usage

### Parse from Input File

```bash
# Parse transceiver data from a text file
./transceiver_parser -input show-transceiver.txt -output output.json

# Parse and output to stdout
./transceiver_parser -input show-transceiver.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./transceiver_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show interface transceiver details` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show interface transceiver details` output with information for each interface including:

- Transceiver presence status
- Type and manufacturer details
- Part numbers and serial numbers
- Bitrate and cable/fiber specifications
- Cisco product identifiers
- DOM support status
- Digital Optical Monitoring data (when available)

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each transceiver produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_transceiver",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    // Transceiver-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_transceiver"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-01-20T10:30:45Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all transceiver data

### Message Fields

The `message` field contains all transceiver-specific data:

```json
{
  "interface_name": "Ethernet1/17",
  "transceiver_present": true,
  "type": "10Gbase-SR",
  "manufacturer": "Siemon",
  "part_number": "S1S10F-V05.0M13",
  "revision": "A",
  "serial_number": "SIM24001B1A",
  "nominal_bitrate": 10300,
  "link_length": "50/125um OM3 fiber is 300 m",
  "cable_type": "",
  "cisco_id": "3",
  "cisco_extended_id": "4",
  "cisco_part_number": "10-2415-03",
  "cisco_product_id": "SFP-10G-SR",
  "cisco_version_id": "V03",
  "dom_supported": true,
  "dom_data": {
    "temperature": {
      "current_value": 34.22,
      "unit": "C",
      "alarm_high": 80.0,
      "alarm_low": -10.0,
      "warning_high": 75.0,
      "warning_low": -5.0,
      "status": "normal"
    },
    "voltage": {
      "current_value": 3.26,
      "unit": "V",
      "alarm_high": 3.6,
      "alarm_low": 3.0,
      "warning_high": 3.5,
      "warning_low": 3.1,
      "status": "normal"
    },
    "current": {
      "current_value": 6.76,
      "unit": "mA",
      "alarm_high": 15.0,
      "alarm_low": 0.0,
      "warning_high": 12.0,
      "warning_low": 0.0,
      "status": "normal"
    },
    "tx_power": {
      "current_value": -1.45,
      "unit": "dBm",
      "alarm_high": 0.99,
      "alarm_low": -8.32,
      "warning_high": 0.0,
      "warning_low": -7.79,
      "status": "normal"
    },
    "rx_power": {
      "current_value": -1.89,
      "unit": "dBm",
      "alarm_high": 0.99,
      "alarm_low": -10.91,
      "warning_high": 0.0,
      "warning_low": -9.91,
      "status": "normal"
    },
    "transmit_fault_count": 0
  }
}
```

#### Core Fields

- `interface_name`: Interface identifier (e.g., "Ethernet1/1")
- `transceiver_present`: Boolean indicating if transceiver is installed

#### Transceiver Details (when present)

- `type`: Transceiver type (e.g., "SFP-H25GB-CU3M", "10Gbase-SR", "QSFP-100G-CR4")
- `manufacturer`: Manufacturer name (e.g., "CISCO-AMPHENOL", "Siemon")
- `part_number`: Manufacturer part number
- `revision`: Hardware revision
- `serial_number`: Unique serial number
- `nominal_bitrate`: Speed in MBit/sec (e.g., 25500, 10300)

#### Physical Specifications

- `link_length`: Cable length or fiber distance specifications
- `cable_type`: Cable type code (e.g., "CA-S", "CA-N")

#### Cisco Identifiers

- `cisco_id`: Cisco ID number
- `cisco_extended_id`: Extended ID number
- `cisco_part_number`: Cisco part number
- `cisco_product_id`: Cisco product identifier
- `cisco_version_id`: Version identifier

#### DOM (Digital Optical Monitoring) Fields

- `dom_supported`: Boolean indicating DOM support
- `dom_data`: Object containing monitoring data (when supported):
  - `temperature`: Temperature monitoring with thresholds
  - `voltage`: Voltage monitoring with thresholds
  - `current`: Current monitoring with thresholds
  - `tx_power`: Transmit power monitoring with thresholds
  - `rx_power`: Receive power monitoring with thresholds
  - `transmit_fault_count`: Number of transmit faults

Each DOM parameter includes:
- `current_value`: Current measurement
- `unit`: Measurement unit (C, V, mA, dBm)
- `alarm_high`: High alarm threshold
- `alarm_low`: Low alarm threshold
- `warning_high`: High warning threshold
- `warning_low`: Low warning threshold
- `status`: Calculated status (normal, high-alarm, low-alarm, high-warning, low-warning)

## Transceiver Types

The parser handles various transceiver types:

- **Copper DAC**: SFP-H25GB-CU3M, SFP-H25GB-CU1M, QSFP-100G-CR4
- **Optical**: 10Gbase-SR, 10Gbase-LR, 40Gbase-SR4
- **Form Factors**: SFP, SFP+, SFP28, QSFP, QSFP28
- **Speeds**: 1G, 10G, 25G, 40G, 100G

## Data Handling

- **Absent Transceivers**: Interfaces without transceivers have `transceiver_present: false`
- **DOM Detection**: Automatically detects DOM support from output
- **Status Calculation**: Compares current values against thresholds to determine alarm/warning status
- **Empty Fields**: Missing or optional fields are omitted or set to empty strings
- **Numeric Parsing**: Bitrate values are parsed as integers, measurements as floats

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "transceiver"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "transceiver",
      "command": "show interface transceiver details"
    }
  ]
}
```

## Examples

### Sample Input

```text
Ethernet1/17
    transceiver is present
    type is 10Gbase-SR
    name is Siemon
    part number is S1S10F-V05.0M13
    revision is A
    serial number is SIM24001B1A
    nominal bitrate is 10300 MBit/sec
    cisco id is 3
    cisco extended id number is 4

           SFP Detail Diagnostics Information (internal calibration)
  ----------------------------------------------------------------------------
                Current              Alarms                  Warnings
                Measurement     High        Low         High          Low
  ----------------------------------------------------------------------------
  Temperature   34.22 C        80.00 C    -10.00 C     75.00 C       -5.00 C
  Voltage        3.26 V         3.60 V      3.00 V      3.50 V        3.10 V
  Current        6.76 mA       15.00 mA     0.00 mA    12.00 mA       0.00 mA
  Tx Power      -1.45 dBm       0.99 dBm   -8.32 dBm    0.00 dBm     -7.79 dBm
  Rx Power      -1.89 dBm       0.99 dBm  -10.91 dBm    0.00 dBm     -9.91 dBm
  Transmit Fault Count = 0
```

### Sample Output

```json
{
  "data_type": "cisco_nexus_transceiver",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "interface_name": "Ethernet1/17",
    "transceiver_present": true,
    "type": "10Gbase-SR",
    "manufacturer": "Siemon",
    "part_number": "S1S10F-V05.0M13",
    "revision": "A",
    "serial_number": "SIM24001B1A",
    "nominal_bitrate": 10300,
    "cisco_id": "3",
    "cisco_extended_id": "4",
    "dom_supported": true,
    "dom_data": {
      "temperature": {
        "current_value": 34.22,
        "unit": "C",
        "alarm_high": 80.0,
        "alarm_low": -10.0,
        "warning_high": 75.0,
        "warning_low": -5.0,
        "status": "normal"
      },
      "voltage": {
        "current_value": 3.26,
        "unit": "V",
        "alarm_high": 3.6,
        "alarm_low": 3.0,
        "warning_high": 3.5,
        "warning_low": 3.1,
        "status": "normal"
      },
      "current": {
        "current_value": 6.76,
        "unit": "mA",
        "alarm_high": 15.0,
        "alarm_low": 0.0,
        "warning_high": 12.0,
        "warning_low": 0.0,
        "status": "normal"
      },
      "tx_power": {
        "current_value": -1.45,
        "unit": "dBm",
        "alarm_high": 0.99,
        "alarm_low": -8.32,
        "warning_high": 0.0,
        "warning_low": -7.79,
        "status": "normal"
      },
      "rx_power": {
        "current_value": -1.89,
        "unit": "dBm",
        "alarm_high": 0.99,
        "alarm_low": -10.91,
        "warning_high": 0.0,
        "warning_low": -9.91,
        "status": "normal"
      },
      "transmit_fault_count": 0
    }
  }
}
```

## Validation

To validate that the parser output conforms to the standardized JSON structure, use the project's validation script:

```bash
# Validate parser output
./transceiver_parser -input show-transceiver.txt | /workspaces/arc-switch2/validate-parser-output.sh

# Or validate a saved output file
./transceiver_parser -input show-transceiver.txt -output transceiver-results.json
/workspaces/arc-switch2/validate-parser-output.sh transceiver-results.json
```

The validation script checks for:

- Presence of all required fields (`data_type`, `timestamp`, `date`, `message`)
- Correct timestamp format (ISO 8601)
- Correct date format (YYYY-MM-DD)
- Valid JSON structure

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed transceiver output
- DOM data parsing with varying formats
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)
- Absent transceivers and missing DOM support

## Requirements

- Go 1.21 or later
- For direct switch access: `vsh` command-line tool available on Cisco Nexus switches