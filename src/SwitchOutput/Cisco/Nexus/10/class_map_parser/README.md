# Cisco Nexus Class Map Parser

This tool parses Cisco Nexus `show class-map` output and converts it to structured JSON format for QoS policy analysis and monitoring.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Class Map Data**: Captures all class map types (QoS, queuing, network-qos) with match conditions
- **Multi-type Support**: Handles different class map types and match criteria
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each class map produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of descriptions and multiple match rules
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

```bash
# Build the binary
go build -o class_map_parser class_map_parser.go
```

## Usage

### Parse from Input File

```bash
# Parse class map data from a text file
./class_map_parser -input show-class-map.txt -output output.json

# Parse and output to stdout
./class_map_parser -input show-class-map.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./class_map_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show class-map` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show class-map` output organized in sections:

- **Type qos class-maps**: QoS classification maps
- **Type queuing class-maps**: Queue management maps
- **Type network-qos class-maps**: Network QoS maps

Each class map entry includes:
- Class map declaration with type and match mode
- Optional description
- Match conditions (cos, precedence, qos-group)

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each class map produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_class_map",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    // Class map-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_class_map"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-01-20T10:30:45Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all class map data

### Message Fields

The `message` field contains all class map-specific data:

```json
{
  "class_name": "RDMA",
  "class_type": "qos",
  "match_type": "match-all",
  "description": "",
  "match_rules": [
    {
      "match_type": "cos",
      "match_value": "3"
    }
  ]
}
```

#### Core Fields

- `class_name`: Name of the class map (e.g., "RDMA", "c-out-q3", "c-nq1")
- `class_type`: Type of class map (qos, queuing, network-qos)
- `match_type`: Match mode (match-all, match-any)

#### Optional Fields

- `description`: Optional description text
- `match_rules`: Array of match conditions

#### Match Rules Structure

Each match rule contains:
- `match_type`: Type of match (cos, precedence, qos-group)
- `match_value`: Value being matched

## Class Map Types

The parser handles three types of class maps:

- **qos**: Quality of Service classification
  - Match conditions: cos, precedence
  - Used for traffic classification

- **queuing**: Queue management
  - Match conditions: qos-group
  - Used for ingress/egress queue assignment

- **network-qos**: Network-wide QoS policies
  - Match conditions: qos-group
  - Used for system-level QoS configuration

## Data Handling

- **Empty Descriptions**: Class maps without descriptions have empty string values
- **Multiple Match Rules**: Class maps can have multiple match conditions captured in the match_rules array
- **Section Headers**: Type headers are used to categorize class maps correctly
- **Whitespace**: All values are trimmed of leading/trailing whitespace

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "class-map"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "class-map",
      "command": "show class-map"
    }
  ]
}
```

## Examples

### Sample Input

```text
  Type qos class-maps
  ====================

    class-map type qos match-all RDMA
      match cos 3

    class-map type qos match-any c-dflt-mpls-qosgrp1
      Description: This is an ingress default qos class-map that classify traffic with prec  1
      match precedence 1

  Type queuing class-maps
  ========================

    class-map type queuing match-any c-out-q3
      Description: Classifier for Egress queue 3
      match qos-group 3

  Type network-qos class-maps
  ===========================
  class-map type network-qos match-any c-nq1
      Description: Default class on qos-group 1
    match qos-group 1
```

### Sample Output

```json
{
  "data_type": "cisco_nexus_class_map",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "class_name": "RDMA",
    "class_type": "qos",
    "match_type": "match-all",
    "description": "",
    "match_rules": [
      {
        "match_type": "cos",
        "match_value": "3"
      }
    ]
  }
}
{
  "data_type": "cisco_nexus_class_map",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "class_name": "c-dflt-mpls-qosgrp1",
    "class_type": "qos",
    "match_type": "match-any",
    "description": "This is an ingress default qos class-map that classify traffic with prec  1",
    "match_rules": [
      {
        "match_type": "precedence",
        "match_value": "1"
      }
    ]
  }
}
{
  "data_type": "cisco_nexus_class_map",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "class_name": "c-out-q3",
    "class_type": "queuing",
    "match_type": "match-any",
    "description": "Classifier for Egress queue 3",
    "match_rules": [
      {
        "match_type": "qos-group",
        "match_value": "3"
      }
    ]
  }
}
```

## Validation

To validate that the parser output conforms to the standardized JSON structure, use the project's validation script:

```bash
# Validate parser output
./class_map_parser -input show-class-map.txt | /workspaces/arc-switch2/validate-parser-output.sh

# Or validate a saved output file
./class_map_parser -input show-class-map.txt -output class-map-results.json
/workspaces/arc-switch2/validate-parser-output.sh class-map-results.json
```

The validation script checks for:

- Presence of all required fields (`data_type`, `timestamp`, `date`, `message`)
- Correct timestamp format (ISO 8601)
- Correct date format (YYYY-MM-DD)
- Valid JSON structure

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed class map definitions
- Missing or incomplete match rules
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)
- Section header parsing

## Requirements

- Go 1.21 or later
- For direct switch access: `vsh` command-line tool available on Cisco Nexus switches