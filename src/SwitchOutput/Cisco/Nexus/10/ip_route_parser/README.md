# Cisco Nexus IP Route Parser

This tool parses Cisco Nexus `show ip route` output and converts it to structured JSON format for network routing analysis and monitoring.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Route Data**: Captures all routing table information including next-hops, metrics, and attributes
- **Multi-path Support**: Handles routes with multiple next-hop entries (ECMP/load balancing)
- **Protocol Recognition**: Automatically identifies route protocols (BGP, direct, local, HSRP)
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each route produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of complex route attributes and tags
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

```bash
# Build the binary
go build -o ip_route_parser ip_route_parser.go
```

## Usage

### Parse from Input File

```bash
# Parse IP routes from a text file
./ip_route_parser -input show-ip-route.txt -output output.json

# Parse and output to stdout
./ip_route_parser -input show-ip-route.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./ip_route_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show ip route` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show ip route` output with the following structure:

- VRF information header
- Network entries with ubest/mbest counters
- Next-hop entries with via addresses, interfaces, and attributes
- Route attributes including preference/metric, age, protocol, and tags

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each route produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_ip_route",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    // Route-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_ip_route"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-01-20T10:30:45Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all route data

### Message Fields

The `message` field contains all route-specific data:

```json
{
  "vrf": "default",
  "network": "0.0.0.0/0",
  "prefix": "0.0.0.0",
  "prefix_length": 0,
  "ubest": 2,
  "mbest": 0,
  "route_type": "bgp",
  "next_hops": [
    {
      "via": "192.168.100.1",
      "interface": "",
      "preference": 20,
      "metric": 0,
      "age": "14w1d",
      "protocol": "bgp-65238",
      "attributes": ["external", "tag 64846"]
    },
    {
      "via": "192.168.100.9",
      "interface": "",
      "preference": 20,
      "metric": 0,
      "age": "14w1d",
      "protocol": "bgp-65238",
      "attributes": ["external", "tag 64846"]
    }
  ]
}
```

#### Core Fields

- `vrf`: VRF name (typically "default" for global routing table)
- `network`: Network address with CIDR notation (e.g., "192.168.1.0/24")
- `prefix`: IP address portion of the network
- `prefix_length`: Subnet mask length (0-32)

#### Path Selection Fields

- `ubest`: Number of best unicast paths
- `mbest`: Number of best multicast paths
- `route_type`: Type of route (attached, bgp, direct, local, hsrp, unknown)

#### Next-Hop Array

Each next-hop entry contains:
- `via`: Next-hop IP address
- `interface`: Egress interface (e.g., "Eth1/47", "Vlan7", "Po50", "Lo0", "Tunnel1")
- `preference`: Administrative distance/preference value
- `metric`: Route metric
- `age`: Route age (e.g., "14w1d", "37w6d")
- `protocol`: Routing protocol (e.g., "bgp-65238", "direct", "local", "hsrp")
- `attributes`: Array of additional attributes (e.g., "internal", "external", "tag 65238")

## Route Types

The parser automatically categorizes routes:

- **attached**: Directly connected networks
- **bgp**: BGP learned routes
- **direct**: Direct routes
- **local**: Local interface addresses
- **hsrp**: HSRP virtual IP routes
- **unknown**: Unrecognized route types

## Data Handling

- **Multiple Next-Hops**: Routes with multiple paths are captured with all next-hop details
- **Empty Fields**: Missing interface or attributes are represented as empty strings or arrays
- **Preference/Metric**: Extracted from `[x/y]` format where x=preference, y=metric
- **Route Attributes**: BGP attributes (internal/external) and tags are parsed into arrays
- **Host Routes**: Routes without explicit prefix length default to /32

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "ip-route"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "ip-route",
      "command": "show ip route"
    }
  ]
}
```

## Examples

### Sample Input

```text
CONTOSO-DC1-TOR-01# show ip route |no
IP Route Table for VRF "default"
'*' denotes best ucast next-hop
'**' denotes best mcast next-hop
'[x/y]' denotes [preference/metric]
'%<string>' in via output denotes VRF <string>

0.0.0.0/0, ubest/mbest: 2/0
    *via 192.168.100.1, [20/0], 14w1d, bgp-65238, external, tag 64846
    *via 192.168.100.9, [20/0], 14w1d, bgp-65238, external, tag 64846
192.168.1.0/24, ubest/mbest: 1/0, attached
    *via 192.168.1.2, Vlan7, [0/0], 37w6d, direct
192.168.1.1/32, ubest/mbest: 1/0, attached
    *via 192.168.1.1, Vlan7, [0/0], 37w6d, hsrp
```

### Sample Output

```json
{
  "data_type": "cisco_nexus_ip_route",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "vrf": "default",
    "network": "0.0.0.0/0",
    "prefix": "0.0.0.0",
    "prefix_length": 0,
    "ubest": 2,
    "mbest": 0,
    "route_type": "bgp",
    "next_hops": [
      {
        "via": "192.168.100.1",
        "interface": "",
        "preference": 20,
        "metric": 0,
        "age": "14w1d",
        "protocol": "bgp-65238",
        "attributes": ["external", "tag 64846"]
      },
      {
        "via": "192.168.100.9",
        "interface": "",
        "preference": 20,
        "metric": 0,
        "age": "14w1d",
        "protocol": "bgp-65238",
        "attributes": ["external", "tag 64846"]
      }
    ]
  }
}
{
  "data_type": "cisco_nexus_ip_route",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "vrf": "default",
    "network": "192.168.1.0/24",
    "prefix": "192.168.1.0",
    "prefix_length": 24,
    "ubest": 1,
    "mbest": 0,
    "route_type": "attached",
    "next_hops": [
      {
        "via": "192.168.1.2",
        "interface": "Vlan7",
        "preference": 0,
        "metric": 0,
        "age": "37w6d",
        "protocol": "direct",
        "attributes": []
      }
    ]
  }
}
```

## Validation

To validate that the parser output conforms to the standardized JSON structure, use the project's validation script:

```bash
# Validate parser output
./ip_route_parser -input show-ip-route.txt | /workspaces/arc-switch2/validate-parser-output.sh

# Or validate a saved output file
./ip_route_parser -input show-ip-route.txt -output route-results.json
/workspaces/arc-switch2/validate-parser-output.sh route-results.json
```

The validation script checks for:

- Presence of all required fields (`data_type`, `timestamp`, `date`, `message`)
- Correct timestamp format (ISO 8601)
- Correct date format (YYYY-MM-DD)
- Valid JSON structure

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed routing table output
- Complex multi-line route entries
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)
- Invalid preference/metric values

## Requirements

- Go 1.21 or later
- For direct switch access: `vsh` command-line tool available on Cisco Nexus switches