# Cisco Nexus LLDP Neighbor Parser

This tool parses Cisco Nexus `show lldp neighbors detail` output and converts it to structured JSON format for network topology discovery and monitoring.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive LLDP Data**: Captures all LLDP neighbor information including capabilities, VLAN mappings, and link aggregation details
- **Multi-line Parsing**: Handles complex multi-line fields like system descriptions and VLAN name lists
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each neighbor produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of null values and "not advertised" fields
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

```bash
# Build the binary
go build -o lldp_neighbor_parser lldp_neighbor_parser.go
```

## Usage

### Parse from Input File

```bash
# Parse LLDP neighbors from a text file
./lldp_neighbor_parser -input show-lldp.txt -output output.json

# Parse and output to stdout
./lldp_neighbor_parser -input show-lldp.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./lldp_neighbor_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show lldp neighbors detail` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show lldp neighbors detail` output, which includes detailed information for each LLDP neighbor with fields like:

- Chassis ID and Port ID
- System Name and Description
- Port Description
- System and Enabled Capabilities
- Management Addresses (IPv4/IPv6)
- VLAN information and mappings
- Link Aggregation status
- Frame size information

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each neighbor produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_lldp_neighbor",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    // Neighbor-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_lldp_neighbor"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-01-20T10:30:45Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all LLDP neighbor data

### Message Fields

The `message` field contains all neighbor-specific data:

```json
{
  "chassis_id": "2416.9d9f.08b0",
  "port_id": "Ethernet1/41",
  "local_port_id": "Eth1/41",
  "port_description": "MLAG Heartbeat and iBGP TOR1-TOR2",
  "system_name": "CONTOSO-DC1-TOR-01.contoso.local",
  "system_description": "Cisco Nexus Operating System (NX-OS) Software 10.3(4a)\nTAC support: http://www.cisco.com/tac\nCopyright (c) 2002-2023, Cisco Systems, Inc. All rights reserved.",
  "time_remaining": 117,
  "system_capabilities": ["B", "R"],
  "enabled_capabilities": ["B", "R"],
  "management_address": "1234.5678.90ab",
  "management_address_ipv6": "2001:db8:1234:5678::1001",
  "vlan_id": "99",
  "max_frame_size": 9216,
  "vlan_names": {
    "1": "default",
    "2": "Unused_Ports",
    "6": "HNV_PA",
    "7": "Management"
  },
  "link_aggregation": {
    "capability": "enabled",
    "status": "aggregated",
    "link_agg_id": 50
  }
}
```

#### Core Fields

- `chassis_id`: Unique chassis identifier
- `port_id`: Remote port identifier
- `local_port_id`: Local interface receiving LLDP advertisement

#### Descriptive Fields

- `port_description`: Description of the remote port (optional)
- `system_name`: Remote system hostname (optional)
- `system_description`: Detailed system description, may be multi-line (optional)
- `time_remaining`: Time in seconds until neighbor information expires

#### Capability Fields

- `system_capabilities`: Array of advertised system capabilities
- `enabled_capabilities`: Array of enabled capabilities

Capability codes:
- `R`: Router
- `B`: Bridge
- `T`: Telephone
- `C`: DOCSIS Cable Device
- `W`: WLAN Access Point
- `P`: Repeater
- `S`: Station
- `O`: Other

#### Network Fields

- `management_address`: IPv4 management address (optional)
- `management_address_ipv6`: IPv6 management address (optional)
- `vlan_id`: Native VLAN ID (optional)
- `max_frame_size`: Maximum frame size in bytes (0 if not advertised)
- `vlan_names`: Map of VLAN IDs to VLAN names (optional)

#### Link Aggregation Fields

- `link_aggregation`: Object containing:
  - `capability`: Link aggregation capability ("enabled", "not advertised", or empty)
  - `status`: Aggregation status ("aggregated", "not aggregated", or empty)
  - `link_agg_id`: Link aggregation group ID (0 if not aggregated)

## Data Handling

- **Null Values**: "null" strings are converted to empty strings
- **Not Advertised**: "not advertised" values are converted to empty strings or appropriate defaults
- **Multi-line Fields**: System descriptions and VLAN name lists are properly parsed across multiple lines
- **Empty Arrays**: Capabilities with no values are represented as empty arrays `[]`
- **Optional Fields**: Missing or not advertised fields are omitted or set to empty/zero values

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "lldp-neighbor"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "lldp-neighbor",
      "command": "show lldp neighbors detail"
    }
  ]
}
```

## Examples

### Sample Input

```text
Chassis id: 2416.9d9f.08b0
Port id: Ethernet1/41
Local Port id: Eth1/41
Port Description: MLAG Heartbeat and iBGP TOR1-TOR2
System Name: CONTOSO-DC1-TOR-01.contoso.local
System Description: Cisco Nexus Operating System (NX-OS) Software 10.3(4a)
TAC support: http://www.cisco.com/tac
Copyright (c) 2002-2023, Cisco Systems, Inc. All rights reserved.
Time remaining: 117 seconds
System Capabilities: B, R
Enabled Capabilities: B, R
Management Address: 1234.5678.90ab
Management Address IPV6: not advertised
Vlan ID: not advertised
Max Frame Size: 9216
Vlan Name TLV:
[Vlan ID: Vlan Name]  not advertised
Link Aggregation TLV: 
Capability: enabled
Status : aggregated
Link agg ID : 50
```

### Sample Output

```json
{
  "data_type": "cisco_nexus_lldp_neighbor",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "chassis_id": "2416.9d9f.08b0",
    "port_id": "Ethernet1/41",
    "local_port_id": "Eth1/41",
    "port_description": "MLAG Heartbeat and iBGP TOR1-TOR2",
    "system_name": "CONTOSO-DC1-TOR-01.contoso.local",
    "system_description": "Cisco Nexus Operating System (NX-OS) Software 10.3(4a)\nTAC support: http://www.cisco.com/tac\nCopyright (c) 2002-2023, Cisco Systems, Inc. All rights reserved.",
    "time_remaining": 117,
    "system_capabilities": ["B", "R"],
    "enabled_capabilities": ["B", "R"],
    "management_address": "1234.5678.90ab",
    "management_address_ipv6": "",
    "vlan_id": "",
    "max_frame_size": 9216,
    "vlan_names": null,
    "link_aggregation": {
      "capability": "enabled",
      "status": "aggregated",
      "link_agg_id": 50
    }
  }
}
```

## Validation

To validate that the parser output conforms to the standardized JSON structure, use the project's validation script:

```bash
# Validate parser output
./lldp_neighbor_parser -input show-lldp.txt | /workspaces/arc-switch2/validate-parser-output.sh

# Or validate a saved output file
./lldp_neighbor_parser -input show-lldp.txt -output lldp-results.json
/workspaces/arc-switch2/validate-parser-output.sh lldp-results.json
```

The validation script checks for:

- Presence of all required fields (`data_type`, `timestamp`, `date`, `message`)
- Correct timestamp format (ISO 8601)
- Correct date format (YYYY-MM-DD)
- Valid JSON structure

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed LLDP output
- Multi-line field parsing
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)
- Null and "not advertised" values

## Requirements

- Go 1.21 or later
- For direct switch access: `vsh` command-line tool available on Cisco Nexus switches