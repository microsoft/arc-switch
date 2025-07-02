# Cisco Nexus Interface Counters Parser

This tool parses Cisco Nexus `show interface counters` output and converts it to structured JSON format for monitoring and analysis.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Metrics**: Captures all interface counter metrics (ingress/egress octets, unicast/multicast/broadcast packets)
- **Interface Classification**: Automatically categorizes interfaces by type (ethernet, port-channel, vlan, management, tunnel)
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each interface produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of unavailable counters and malformed data
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

```bash
# Build the binary
go build -o interface_counters_parser interface_counters_parser.go
```

## Usage

### Parse from Input File

```bash
# Parse interface counters from a text file
./interface_counters_parser -input show-interface-counter.txt -output output.json

# Parse and output to stdout
./interface_counters_parser -input show-interface-counter.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./interface_counters_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show interface counters` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show interface counters` output, which includes multiple sections:

1. **InOctets and InUcastPkts**: Ingress octets and unicast packets
2. **InMcastPkts and InBcastPkts**: Ingress multicast and broadcast packets  
3. **OutOctets and OutUcastPkts**: Egress octets and unicast packets
4. **OutMcastPkts and OutBcastPkts**: Egress multicast and broadcast packets

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each interface produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_interface_counters",
  "timestamp": "2025-07-01T23:45:57Z",
  "date": "2025-07-01",
  "message": {
    // Interface-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_interface_counters"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-07-01T23:45:57Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all interface counter data

### Message Fields

The `message` field contains all interface-specific data:

```json
{
  "interface_name": "Eth1/29",
  "interface_type": "ethernet",
  "in_octets": 5049641112717,
  "in_ucast_pkts": 6489292847,
  "in_mcast_pkts": 963320,
  "in_bcast_pkts": 490562,
  "out_octets": 5095780391460,
  "out_ucast_pkts": 6593171948,
  "out_mcast_pkts": 10233218,
  "out_bcast_pkts": 6452477,
  "has_ingress_data": true,
  "has_egress_data": true
}
```

#### Core Fields

- `interface_name`: Interface name (e.g., Eth1/29, Po50, Vlan125, mgmt0, Tunnel1)
- `interface_type`: Categorized interface type (ethernet, port-channel, vlan, management, tunnel, unknown)

#### Counter Fields

- `in_octets`: Ingress octets
- `in_ucast_pkts`: Ingress unicast packets  
- `in_mcast_pkts`: Ingress multicast packets
- `in_bcast_pkts`: Ingress broadcast packets
- `out_octets`: Egress octets
- `out_ucast_pkts`: Egress unicast packets
- `out_mcast_pkts`: Egress multicast packets
- `out_bcast_pkts`: Egress broadcast packets

#### Status Fields

- `has_ingress_data`: True if ingress counters are available
- `has_egress_data`: True if egress counters are available

## Interface Types

The tool automatically categorizes interfaces:

- **ethernet**: Physical Ethernet interfaces (Eth1/1, Eth1/2, etc.)
- **port-channel**: Port channel/LAG interfaces (Po50, Po101, etc.)
- **vlan**: VLAN interfaces (Vlan1, Vlan125, etc.)
- **management**: Management interfaces (mgmt0)
- **tunnel**: Tunnel interfaces (Tunnel1, etc.)
- **unknown**: Unrecognized interface types

## Data Handling

- **Unavailable Counters**: Represented as `-1` in JSON output (e.g., for VLAN interfaces showing "--")
- **Zero Values**: Actual zero counters are preserved as `0`
- **Status Flags**: `has_ingress_data` and `has_egress_data` indicate data availability

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "interface-counter"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "interface-counter",
      "command": "show interface counters"
    }
  ]
}
```

## Examples

### Sample Input

```text
RR1-S46-R14-93180hl-22-1a# show interface counters 

----------------------------------------------------------------------------------
Port                                     InOctets                      InUcastPkts
----------------------------------------------------------------------------------
Eth1/1                               205027653248                        650373664
Eth1/2                               144387970112                        277741204
```

### Sample Output

```json
{
  "data_type": "cisco_nexus_interface_counters",
  "timestamp": "2025-07-01T23:45:57Z",
  "date": "2025-07-01",
  "message": {
    "interface_name": "Eth1/29",
    "interface_type": "ethernet",
    "in_octets": 5049641112717,
    "in_ucast_pkts": 6489292847,
    "in_mcast_pkts": 963320,
    "in_bcast_pkts": 490562,
    "out_octets": 5095780391460,
    "out_ucast_pkts": 6593171948,
    "out_mcast_pkts": 10233218,
    "out_bcast_pkts": 6452477,
    "has_ingress_data": true,
    "has_egress_data": true
  }
}
```

## Validation

To validate that the parser output conforms to the standardized JSON structure, use the project's validation script:

```bash
# Validate parser output
./interface_counters_parser -input show-interface-counter.txt | /workspaces/arc-switch2/validate-parser-output.sh

# Or validate a saved output file
./interface_counters_parser -input show-interface-counter.txt -output interface-results.json
/workspaces/arc-switch2/validate-parser-output.sh interface-results.json
```

The validation script checks for:

- Presence of all required fields (`data_type`, `timestamp`, `date`, `message`)
- Correct timestamp format (ISO 8601)
- Correct date format (YYYY-MM-DD)
- Valid JSON structure

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed command output
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)
- Invalid counter values

## Requirements

- Go 1.21 or later
- For direct switch access: `vsh` command-line tool available on Cisco Nexus switches
