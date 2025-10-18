# Cisco Nexus Interface Error Counters Parser

This tool parses Cisco Nexus `show interface counters errors` output and converts it to structured JSON format for monitoring and analysis.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Error Metrics**: Captures all interface error counter metrics across 5 sections
- **Interface Classification**: Automatically categorizes interfaces by type (ethernet, port-channel, vlan, management, tunnel)
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each interface produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of unavailable counters (`--`) and malformed data
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

```bash
# Build the binary
go build -o interface_counters_error_parser interface_counters_error_parser.go
```

## Usage

### Parse from Input File

```bash
# Parse interface error counters from a text file
./interface_counters_error_parser -input show-interface-counter-errors.txt -output output.json

# Parse and output to stdout
./interface_counters_error_parser -input show-interface-counter-errors.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./interface_counters_error_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show interface counters errors` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show interface counters errors` output, which includes five sections:

1. **Align-Err, FCS-Err, Xmit-Err, Rcv-Err, UnderSize, OutDiscards**: Basic error counters
2. **Single-Col, Multi-Col, Late-Col, Exces-Col, Carri-Sen, Runts**: Collision and runt counters
3. **Giants, SQETest-Err, Deferred-Tx, IntMacTx-Er, IntMacRx-Er, Symbol-Err**: MAC and physical layer errors
4. **InDiscards**: Input discards
5. **Stomped-CRC**: Stomped CRC errors (ethernet interfaces only)

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each interface produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_interface_error_counters",
  "timestamp": "2025-10-17T23:45:57Z",
  "date": "2025-10-17",
  "message": {
    // Interface-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_interface_error_counters"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-10-17T23:45:57Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all interface error counter data

### Message Fields

The `message` field contains all interface-specific data:

```json
{
  "interface_name": "Eth1/1",
  "interface_type": "ethernet",
  "align_err": 0,
  "fcs_err": 0,
  "xmit_err": 0,
  "rcv_err": 0,
  "under_size": 0,
  "out_discards": 0,
  "single_col": 0,
  "multi_col": 0,
  "late_col": 0,
  "exces_col": 0,
  "carri_sen": 0,
  "runts": 0,
  "giants": 0,
  "sqetest_err": -1,
  "deferred_tx": 0,
  "intmac_tx_er": 0,
  "intmac_rx_er": 0,
  "symbol_err": 0,
  "in_discards": 0,
  "stomped_crc": 0,
  "has_error_data": true
}
```

#### Core Fields

- `interface_name`: Interface name (e.g., Eth1/1, Po50, mgmt0)
- `interface_type`: Categorized interface type (ethernet, port-channel, vlan, management, tunnel, unknown)

#### Error Counter Fields (Section 1)

- `align_err`: Alignment errors
- `fcs_err`: Frame Check Sequence errors
- `xmit_err`: Transmit errors
- `rcv_err`: Receive errors
- `under_size`: Undersized packets
- `out_discards`: Output discards

#### Collision Counter Fields (Section 2)

- `single_col`: Single collisions
- `multi_col`: Multiple collisions
- `late_col`: Late collisions
- `exces_col`: Excessive collisions
- `carri_sen`: Carrier sense errors
- `runts`: Runt packets

#### MAC/Physical Layer Error Fields (Section 3)

- `giants`: Giant packets
- `sqetest_err`: SQE test errors
- `deferred_tx`: Deferred transmissions
- `intmac_tx_er`: Internal MAC transmit errors
- `intmac_rx_er`: Internal MAC receive errors
- `symbol_err`: Symbol errors

#### Additional Error Fields

- `in_discards`: Input discards (Section 4)
- `stomped_crc`: Stomped CRC errors (Section 5, ethernet interfaces only)

#### Status Fields

- `has_error_data`: True if error counters are available

## Interface Types

The tool automatically categorizes interfaces:

- **ethernet**: Physical Ethernet interfaces (Eth1/1, Eth1/2, etc.)
- **port-channel**: Port channel/LAG interfaces (Po50, Po101, etc.)
- **vlan**: VLAN interfaces (Vlan1, Vlan125, etc.)
- **management**: Management interfaces (mgmt0)
- **tunnel**: Tunnel interfaces (Tunnel1, etc.)
- **unknown**: Unrecognized interface types

## Data Handling

- **Unavailable Counters**: Represented as `-1` in JSON output (e.g., for management interface showing `--`)
- **Zero Values**: Actual zero counters are preserved as `0`
- **Status Flag**: `has_error_data` indicates data availability
- **Missing Sections**: Interfaces not appearing in a section will have `-1` for those counters

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "interface-error-counter"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "interface-error-counter",
      "command": "show interface counters errors"
    }
  ]
}
```

## Examples

### Sample Input

```text
show interface counters errors

--------------------------------------------------------------------------------
Port          Align-Err    FCS-Err   Xmit-Err    Rcv-Err  UnderSize OutDiscards
--------------------------------------------------------------------------------
mgmt0                   0          0         --         --         --          --
Eth1/1                  0          0          0          0          0           0
Po700                   0          0          0          0          0        4303
```

### Sample Output

```json
{
  "data_type": "cisco_nexus_interface_error_counters",
  "timestamp": "2025-10-17T23:45:57Z",
  "date": "2025-10-17",
  "message": {
    "interface_name": "Po700",
    "interface_type": "port-channel",
    "align_err": 0,
    "fcs_err": 0,
    "xmit_err": 0,
    "rcv_err": 0,
    "under_size": 0,
    "out_discards": 4303,
    "single_col": 0,
    "multi_col": 0,
    "late_col": 0,
    "exces_col": 0,
    "carri_sen": 0,
    "runts": 0,
    "giants": 0,
    "sqetest_err": -1,
    "deferred_tx": 0,
    "intmac_tx_er": 0,
    "intmac_rx_er": 0,
    "symbol_err": 0,
    "in_discards": 0,
    "stomped_crc": -1,
    "has_error_data": true
  }
}
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

## Complementary Tools

This parser complements the existing `interface_counters_parser` which collects traffic statistics (octets, packets). Together they provide comprehensive interface monitoring:

- **interface_counters_parser**: Traffic statistics (InOctets, OutOctets, packet counts)
- **interface_counters_error_parser**: Error statistics (alignment errors, CRC errors, discards, etc.)
