# Cisco Nexus Interface Counters Parser

This tool parses Cisco Nexus `show interface counters` output and converts it to structured JSON format for monitoring and analysis.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Metrics**: Captures all interface counter metrics (ingress/egress octets, unicast/multicast/broadcast packets)
- **Interface Classification**: Automatically categorizes interfaces by type (ethernet, port-channel, vlan, management, tunnel)
- **JSON Output**: Structured JSON format with proper data types and metadata
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

Each interface is represented as a JSON object with the following structure:

```json
{
  "data_type": "interface_counters",
  "timestamp": "2024-01-15T10:30:00Z",
  "date": "2024-01-15",
  "interface_name": "Eth1/1",
  "interface_type": "ethernet",
  "in_octets": 205027653248,
  "in_ucast_pkts": 650373664,
  "in_mcast_pkts": 2262324,
  "in_bcast_pkts": 68097,
  "out_octets": 3195383643785,
  "out_ucast_pkts": 2314463086,
  "out_mcast_pkts": 365931965,
  "out_bcast_pkts": 53571839,
  "has_ingress_data": true,
  "has_egress_data": true
}
```

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
[
  {
    "data_type": "interface_counters",
    "timestamp": "2024-01-15T10:30:00Z",
    "date": "2024-01-15",
    "interface_name": "Eth1/1",
    "interface_type": "ethernet",
    "in_octets": 205027653248,
    "in_ucast_pkts": 650373664,
    "in_mcast_pkts": 2262324,
    "in_bcast_pkts": 68097,
    "out_octets": 3195383643785,
    "out_ucast_pkts": 2314463086,
    "out_mcast_pkts": 365931965,
    "out_bcast_pkts": 53571839,
    "has_ingress_data": true,
    "has_egress_data": true
  }
]
```

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
