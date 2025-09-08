# Cisco Nexus Inventory Parser

This tool parses Cisco Nexus `show inventory all` output and converts it to structured JSON format for hardware inventory tracking and asset management.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Inventory Data**: Captures all hardware components including chassis, modules, power supplies, fans, and transceivers
- **Component Classification**: Automatically categorizes components by type
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each component produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of empty fields and N/A values
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

```bash
# Build the binary
go build -o inventory_parser inventory_parser.go
```

## Usage

### Parse from Input File

```bash
# Parse inventory data from a text file
./inventory_parser -input show-inventory.txt -output output.json

# Parse and output to stdout
./inventory_parser -input show-inventory.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./inventory_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show inventory all` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show inventory all` output with two-line entries for each component:

1. **NAME and DESCR line**: Component name and description
2. **PID, VID, and SN line**: Product ID, Version ID, and Serial Number

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each component produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_inventory",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    // Component-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_inventory"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-01-20T10:30:45Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all inventory data

### Message Fields

The `message` field contains all component-specific data:

```json
{
  "name": "Chassis",
  "description": "Nexus9000 C93180YC-FX Chassis",
  "product_id": "N9K-C93180YC-FX",
  "version_id": "V04",
  "serial_number": "FKE24000X1A",
  "component_type": "chassis"
}
```

#### Core Fields

- `name`: Component name (e.g., "Chassis", "Slot 1", "Power Supply 1", "Fan 1", "Ethernet1/1")
- `description`: Component description
- `product_id`: Product identifier (PID)
- `version_id`: Version identifier (VID)
- `serial_number`: Serial number (SN)
- `component_type`: Automatically categorized type

## Component Types

The parser automatically categorizes components:

- **chassis**: Main chassis components
- **slot**: Line cards and modules
- **power_supply**: Power supply units
- **fan**: Fan modules
- **transceiver**: SFP/QSFP transceivers (Ethernet interfaces)
- **unknown**: Unrecognized component types

## Data Handling

- **Quoted Values**: Automatically removes surrounding quotes from NAME and DESCR fields
- **Empty Fields**: Empty PID fields are preserved as empty strings
- **N/A Values**: Serial numbers showing "N/A" are preserved as-is
- **Ethernet Names**: Trailing commas in Ethernet interface names are automatically cleaned
- **Whitespace**: All values are trimmed of leading/trailing whitespace

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "inventory"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "inventory",
      "command": "show inventory all"
    }
  ]
}
```

## Examples

### Sample Input

```text
CONTOSO-DC1-TOR-01# show inventory all 
NAME: "Chassis",  DESCR: "Nexus9000 C93180YC-FX Chassis"         
PID: N9K-C93180YC-FX     ,  VID: V04 ,  SN: FKE24000X1A          

NAME: "Power Supply 1",  DESCR: "Nexus9000 C93180YC-FX Chassis Power Supply"
PID: NXA-PAC-500W-PE     ,  VID: V01 ,  SN: ABC24001X2B          

NAME: "Fan 1",  DESCR: "Nexus9000 C93180YC-FX Chassis Fan Module"
PID: NXA-FAN-30CFM-F     ,  VID: V01 ,  SN: N/A                  

NAME: Ethernet1/1,  DESCR: CISCO-AMPHENOL                          
PID: SFP-H25G-CU3M       ,  VID: NDCCGJ-C403,  SN: XYZ24001A1B
```

### Sample Output

```json
{
  "data_type": "cisco_nexus_inventory",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "name": "Chassis",
    "description": "Nexus9000 C93180YC-FX Chassis",
    "product_id": "N9K-C93180YC-FX",
    "version_id": "V04",
    "serial_number": "FKE24000X1A",
    "component_type": "chassis"
  }
}
{
  "data_type": "cisco_nexus_inventory",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "name": "Power Supply 1",
    "description": "Nexus9000 C93180YC-FX Chassis Power Supply",
    "product_id": "NXA-PAC-500W-PE",
    "version_id": "V01",
    "serial_number": "ABC24001X2B",
    "component_type": "power_supply"
  }
}
{
  "data_type": "cisco_nexus_inventory",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "name": "Fan 1",
    "description": "Nexus9000 C93180YC-FX Chassis Fan Module",
    "product_id": "NXA-FAN-30CFM-F",
    "version_id": "V01",
    "serial_number": "N/A",
    "component_type": "fan"
  }
}
{
  "data_type": "cisco_nexus_inventory",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "name": "Ethernet1/1",
    "description": "CISCO-AMPHENOL",
    "product_id": "SFP-H25G-CU3M",
    "version_id": "NDCCGJ-C403",
    "serial_number": "XYZ24001A1B",
    "component_type": "transceiver"
  }
}
```

## Validation

To validate that the parser output conforms to the standardized JSON structure, use the project's validation script:

```bash
# Validate parser output
./inventory_parser -input show-inventory.txt | /workspaces/arc-switch2/validate-parser-output.sh

# Or validate a saved output file
./inventory_parser -input show-inventory.txt -output inventory-results.json
/workspaces/arc-switch2/validate-parser-output.sh inventory-results.json
```

The validation script checks for:

- Presence of all required fields (`data_type`, `timestamp`, `date`, `message`)
- Correct timestamp format (ISO 8601)
- Correct date format (YYYY-MM-DD)
- Valid JSON structure

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed inventory output
- Multi-line parsing issues
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)
- Empty or N/A field values

## Requirements

- Go 1.21 or later
- For direct switch access: `vsh` command-line tool available on Cisco Nexus switches