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

## Error Type Breakdown - Network Engineering Perspective

The Interface Error Counters Parser (`show interface counters errors`) tracks critical network health metrics. Understanding these errors is essential for maintaining network reliability and troubleshooting connectivity issues.

### Section 1: Basic Error Counters

**Align-Err (Alignment Errors)**
- **What it is**: Frames that don't end on an octet (8-bit) boundary
- **Why it matters**: Usually indicates physical layer problems like bad cables, duplex mismatches, or faulty NICs
- **Action threshold**: Any non-zero value warrants investigation
- **Common causes**: Duplex mismatches, failing transceivers, EMI interference

**FCS-Err (Frame Check Sequence Errors)**
- **What it is**: Frames with invalid CRC checksums
- **Why it matters**: One of the most common indicators of physical layer issues affecting data integrity
- **Action threshold**: > 0.1% of total traffic or consistently increasing
- **Common causes**: Bad cables, failing optics, noise on the line, excessive cable length

**Xmit-Err (Transmit Errors)**
- **What it is**: Errors encountered while transmitting frames
- **Why it matters**: Indicates problems on the local switch's transmit path
- **Action threshold**: Any sustained errors
- **Common causes**: Hardware failures, buffer exhaustion, internal switch issues

**Rcv-Err (Receive Errors)**
- **What it is**: Errors encountered while receiving frames
- **Why it matters**: Indicates problems receiving data from connected devices
- **Action threshold**: Any sustained errors
- **Common causes**: Connected device issues, cable problems, signal integrity issues

**UnderSize (Undersized Packets)**
- **What it is**: Frames smaller than 64 bytes (minimum Ethernet frame size)
- **Why it matters**: Often indicates collisions or malfunctioning equipment
- **Action threshold**: Sustained presence suggests network problems
- **Common causes**: Collisions, fragmented frames, faulty NICs

**OutDiscards (Output Discards)**
- **What it is**: Outbound packets dropped by the switch
- **Why it matters**: Indicates congestion or QoS policy enforcement - packets being intentionally dropped
- **Action threshold**: Increasing trend indicates capacity or QoS issues
- **Common causes**: Output queue full, QoS policing, bandwidth exhaustion

### Section 2: Collision Counters

**Single-Col (Single Collisions)**
- **What it is**: Frames that experienced exactly one collision before successful transmission
- **Why it matters**: Normal in half-duplex environments but should be zero in full-duplex
- **Action threshold**: Any value in full-duplex mode indicates duplex mismatch
- **Common causes**: Half-duplex operation, duplex mismatch (critical issue)

**Multi-Col (Multiple Collisions)**
- **What it is**: Frames that experienced multiple collisions before successful transmission
- **Why it matters**: Indicates network congestion or duplex issues
- **Action threshold**: Any value suggests problems, especially in modern networks
- **Common causes**: Duplex mismatch, overloaded half-duplex segment

**Late-Col (Late Collisions)**
- **What it is**: Collisions occurring after the first 64 bytes were transmitted
- **Why it matters**: Always indicates a problem - typically cable length violations or duplex mismatches
- **Action threshold**: Any non-zero value is critical - investigate immediately
- **Common causes**: Cable too long (>100m for copper), duplex mismatch, faulty NIC

**Exces-Col (Excessive Collisions)**
- **What it is**: Frames dropped after 16 consecutive collisions
- **Why it matters**: Severe network congestion or duplex mismatch
- **Action threshold**: Any value indicates serious problems
- **Common causes**: Severe duplex mismatch, extreme network congestion

**Carri-Sen (Carrier Sense Errors)**
- **What it is**: The interface didn't properly detect carrier signal before transmitting
- **Why it matters**: Physical layer issue affecting collision detection
- **Action threshold**: Any non-zero value warrants investigation
- **Common causes**: Cable issues, NIC problems, physical layer defects

**Runts (Runt Packets)**
- **What it is**: Frames smaller than minimum size (64 bytes) with bad FCS
- **Why it matters**: Usually indicates collisions or hardware problems
- **Action threshold**: Sustained presence suggests hardware issues
- **Common causes**: Collisions, faulty NICs, cable issues

### Section 3: MAC and Physical Layer Errors

**Giants (Giant Packets)**
- **What it is**: Frames exceeding maximum transmission unit (typically >1518 bytes without jumbo frames)
- **Why it matters**: Can indicate misconfigurations or malfunctioning devices
- **Action threshold**: Check if jumbo frames are expected; otherwise investigate
- **Common causes**: MTU mismatch, baby giant frames (1518-1522), malfunctioning equipment

**SQETest-Err (SQE Test Errors)**
- **What it is**: Signal Quality Error test failures (legacy 10Base-5 heartbeat)
- **Why it matters**: Generally not applicable to modern Ethernet (shows as -- on most interfaces)
- **Action threshold**: N/A for modern equipment
- **Common causes**: Legacy equipment, not relevant for modern switches

**Deferred-Tx (Deferred Transmissions)**
- **What it is**: Transmissions delayed because the medium was busy
- **Why it matters**: Normal in half-duplex; indicates congestion
- **Action threshold**: High values in half-duplex environments suggest congestion
- **Common causes**: Normal half-duplex operation, network congestion

**IntMacTx-Er (Internal MAC Transmit Errors)**
- **What it is**: Internal switch MAC-level transmit errors
- **Why it matters**: Indicates internal switch hardware problems
- **Action threshold**: Any non-zero value may indicate hardware failure
- **Common causes**: ASIC problems, hardware defects, internal switch issues

**IntMacRx-Er (Internal MAC Receive Errors)**
- **What it is**: Internal switch MAC-level receive errors
- **Why it matters**: Indicates internal switch hardware problems
- **Action threshold**: Any non-zero value may indicate hardware failure
- **Common causes**: ASIC problems, hardware defects, internal buffer issues

**Symbol-Err (Symbol Errors)**
- **What it is**: Invalid data symbols received at physical layer
- **Why it matters**: Physical layer integrity issues, often transceiver or cable related
- **Action threshold**: Any sustained errors indicate physical problems
- **Common causes**: Bad optics, cable issues, signal integrity problems

### Section 4: Discard Counters

**InDiscards (Input Discards)**
- **What it is**: Inbound packets dropped by the switch
- **Why it matters**: Indicates resource exhaustion or policy drops on ingress
- **Action threshold**: Increasing trend suggests capacity issues
- **Common causes**: Input queue full, ACL drops, rate limiting, buffer exhaustion

### Section 5: CRC Errors

**Stomped-CRC (Stomped CRC Errors)**
- **What it is**: CRC intentionally invalidated by switch for internal signaling (Ethernet interfaces only)
- **Why it matters**: Switch-specific mechanism; usually not a concern unless excessive
- **Action threshold**: Vendor-specific; consult Cisco documentation
- **Common causes**: Internal switch operations, cut-through switching artifacts

## KQL Monitoring Queries for Azure Data Explorer

These queries help monitor Cisco Nexus interface errors when data is ingested into Azure Data Explorer (Kusto). Assumes the table name is `CiscoNexusInterfaceErrors` with the standard schema from the parser.

### Summary Dashboard View - Big Picture Health Check

```kql
// Overall Interface Health Summary - Shows all interfaces with any errors in last 24 hours
CiscoNexusInterfaceErrors
| where timestamp > ago(24h)
| extend message = parse_json(message)
| extend 
    interface_name = tostring(message.interface_name),
    interface_type = tostring(message.interface_type),
    total_errors = toint(message.align_err) + toint(message.fcs_err) + 
                   toint(message.xmit_err) + toint(message.rcv_err) + 
                   toint(message.out_discards) + toint(message.in_discards) +
                   toint(message.late_col) + toint(message.exces_col) +
                   toint(message.symbol_err) + toint(message.intmac_tx_er) + toint(message.intmac_rx_er)
| where total_errors > 0
| summarize 
    LastSeen = max(timestamp),
    TotalErrors = max(total_errors),
    MaxFCS = max(toint(message.fcs_err)),
    MaxAlignErr = max(toint(message.align_err)),
    MaxOutDiscards = max(toint(message.out_discards)),
    MaxInDiscards = max(toint(message.in_discards)),
    MaxLateCol = max(toint(message.late_col))
    by interface_name, interface_type
| order by TotalErrors desc
| project interface_name, interface_type, LastSeen, TotalErrors, MaxFCS, MaxAlignErr, MaxOutDiscards, MaxInDiscards, MaxLateCol
```

### Critical Physical Layer Issues - Actionable Alerts

```kql
// Critical Physical Layer Errors - FCS and Alignment Errors (Cable/Optics Issues)
CiscoNexusInterfaceErrors
| where timestamp > ago(1h)
| extend message = parse_json(message)
| extend 
    interface_name = tostring(message.interface_name),
    fcs_err = toint(message.fcs_err),
    align_err = toint(message.align_err),
    symbol_err = toint(message.symbol_err)
| where fcs_err > 0 or align_err > 0 or symbol_err > 0
| summarize 
    LastSeen = max(timestamp),
    FCS_Errors = max(fcs_err),
    Align_Errors = max(align_err),
    Symbol_Errors = max(symbol_err),
    ErrorCount = count()
    by interface_name
| order by FCS_Errors desc
| project interface_name, LastSeen, FCS_Errors, Align_Errors, Symbol_Errors, ErrorCount
// ACTION: Check cables, transceivers, and physical connections
```

### Duplex Mismatch Detection

```kql
// Duplex Mismatch Indicators - Late Collisions and Excessive Collisions
CiscoNexusInterfaceErrors
| where timestamp > ago(1h)
| extend message = parse_json(message)
| extend 
    interface_name = tostring(message.interface_name),
    late_col = toint(message.late_col),
    exces_col = toint(message.exces_col),
    single_col = toint(message.single_col),
    multi_col = toint(message.multi_col)
| where late_col > 0 or exces_col > 0 or single_col > 0 or multi_col > 0
| summarize 
    LastSeen = max(timestamp),
    Late_Collisions = max(late_col),
    Excessive_Collisions = max(exces_col),
    Single_Collisions = max(single_col),
    Multi_Collisions = max(multi_col)
    by interface_name
| order by Late_Collisions desc
| project interface_name, LastSeen, Late_Collisions, Excessive_Collisions, Single_Collisions, Multi_Collisions
// ACTION: Check duplex settings - Late collisions usually indicate duplex mismatch
```

### Congestion and Discard Monitoring

```kql
// Congestion Indicators - Output and Input Discards
CiscoNexusInterfaceErrors
| where timestamp > ago(6h)
| extend message = parse_json(message)
| extend 
    interface_name = tostring(message.interface_name),
    interface_type = tostring(message.interface_type),
    out_discards = toint(message.out_discards),
    in_discards = toint(message.in_discards)
| where out_discards > 0 or in_discards > 0
| summarize 
    LastSeen = max(timestamp),
    Output_Discards = max(out_discards),
    Input_Discards = max(in_discards),
    Total_Discards = max(out_discards) + max(in_discards)
    by interface_name, interface_type
| order by Total_Discards desc
| project interface_name, interface_type, LastSeen, Output_Discards, Input_Discards, Total_Discards
// ACTION: Check for bandwidth saturation or QoS policy issues
```

### Hardware Failure Detection

```kql
// Hardware Issues - Internal MAC Errors
CiscoNexusInterfaceErrors
| where timestamp > ago(24h)
| extend message = parse_json(message)
| extend 
    interface_name = tostring(message.interface_name),
    intmac_tx_er = toint(message.intmac_tx_er),
    intmac_rx_er = toint(message.intmac_rx_er)
| where intmac_tx_er > 0 or intmac_rx_er > 0
| summarize 
    LastSeen = max(timestamp),
    MAC_TX_Errors = max(intmac_tx_er),
    MAC_RX_Errors = max(intmac_rx_er),
    ErrorOccurrences = count()
    by interface_name
| order by MAC_TX_Errors desc
| project interface_name, LastSeen, MAC_TX_Errors, MAC_RX_Errors, ErrorOccurrences
// ACTION: Potential hardware failure - consider RMA or linecard replacement
```

### Trending Analysis - Error Rate Over Time

```kql
// Error Trending - Hourly Error Rates for Specific Interface
let targetInterface = "Eth1/1";  // Change to your interface
CiscoNexusInterfaceErrors
| where timestamp > ago(7d)
| extend message = parse_json(message)
| extend 
    interface_name = tostring(message.interface_name),
    fcs_err = toint(message.fcs_err),
    out_discards = toint(message.out_discards),
    in_discards = toint(message.in_discards)
| where interface_name == targetInterface
| summarize 
    FCS_Errors = max(fcs_err),
    Out_Discards = max(out_discards),
    In_Discards = max(in_discards)
    by bin(timestamp, 1h)
| order by timestamp desc
| render timechart
```

### Interface Type Comparison

```kql
// Error Distribution by Interface Type (Ethernet vs Port-Channel vs VLAN)
CiscoNexusInterfaceErrors
| where timestamp > ago(24h)
| extend message = parse_json(message)
| extend 
    interface_type = tostring(message.interface_type),
    total_errors = toint(message.align_err) + toint(message.fcs_err) + 
                   toint(message.out_discards) + toint(message.in_discards)
| where total_errors > 0
| summarize 
    InterfaceCount = dcount(tostring(message.interface_name)),
    TotalErrors = sum(total_errors),
    AvgErrorsPerInterface = avg(total_errors)
    by interface_type
| order by TotalErrors desc
```

### Top Problematic Interfaces

```kql
// Top 10 Most Problematic Interfaces in Last Hour
CiscoNexusInterfaceErrors
| where timestamp > ago(1h)
| extend message = parse_json(message)
| extend 
    interface_name = tostring(message.interface_name),
    interface_type = tostring(message.interface_type),
    fcs_err = toint(message.fcs_err),
    align_err = toint(message.align_err),
    out_discards = toint(message.out_discards),
    in_discards = toint(message.in_discards),
    late_col = toint(message.late_col),
    symbol_err = toint(message.symbol_err)
| extend error_score = (fcs_err * 10) + (align_err * 10) + (late_col * 100) + 
                       (out_discards * 1) + (in_discards * 1) + (symbol_err * 5)
| where error_score > 0
| summarize 
    LastSeen = max(timestamp),
    ErrorScore = max(error_score),
    FCS = max(fcs_err),
    Align = max(align_err),
    OutDisc = max(out_discards),
    LateCol = max(late_col)
    by interface_name, interface_type
| top 10 by ErrorScore desc
| project interface_name, interface_type, LastSeen, ErrorScore, FCS, Align, OutDisc, LateCol
```

### Detailed Interface Investigation View

```kql
// Complete Error Profile for Specific Interface - Deep Dive
let targetInterface = "Eth1/1";  // Change to interface under investigation
CiscoNexusInterfaceErrors
| where timestamp > ago(24h)
| extend message = parse_json(message)
| extend interface_name = tostring(message.interface_name)
| where interface_name == targetInterface
| project 
    timestamp,
    interface_name,
    interface_type = tostring(message.interface_type),
    // Physical Layer
    align_err = toint(message.align_err),
    fcs_err = toint(message.fcs_err),
    symbol_err = toint(message.symbol_err),
    // Transmit/Receive
    xmit_err = toint(message.xmit_err),
    rcv_err = toint(message.rcv_err),
    // Discards
    out_discards = toint(message.out_discards),
    in_discards = toint(message.in_discards),
    // Collisions
    single_col = toint(message.single_col),
    multi_col = toint(message.multi_col),
    late_col = toint(message.late_col),
    exces_col = toint(message.exces_col),
    // Size Issues
    under_size = toint(message.under_size),
    runts = toint(message.runts),
    giants = toint(message.giants),
    // MAC Errors
    intmac_tx_er = toint(message.intmac_tx_er),
    intmac_rx_er = toint(message.intmac_rx_er),
    // Other
    carri_sen = toint(message.carri_sen),
    deferred_tx = toint(message.deferred_tx),
    stomped_crc = toint(message.stomped_crc)
| order by timestamp desc
```

### Alert Configuration Template

```kql
// Alert Query - Critical Errors Requiring Immediate Action
// Configure this as an Azure Monitor Alert
CiscoNexusInterfaceErrors
| where timestamp > ago(5m)
| extend message = parse_json(message)
| extend 
    interface_name = tostring(message.interface_name),
    late_col = toint(message.late_col),
    exces_col = toint(message.exces_col),
    intmac_tx_er = toint(message.intmac_tx_er),
    intmac_rx_er = toint(message.intmac_rx_er),
    fcs_err = toint(message.fcs_err)
| where late_col > 0 or exces_col > 0 or intmac_tx_er > 0 or intmac_rx_er > 0 or fcs_err > 100
| summarize 
    ErrorType = strcat(
        iff(late_col > 0, "Late Collisions ", ""),
        iff(exces_col > 0, "Excessive Collisions ", ""),
        iff(intmac_tx_er > 0, "MAC TX Errors ", ""),
        iff(intmac_rx_er > 0, "MAC RX Errors ", ""),
        iff(fcs_err > 100, "High FCS Errors ", "")
    ),
    LastSeen = max(timestamp)
    by interface_name
| project interface_name, ErrorType, LastSeen
// This will trigger when critical errors are detected
```

## Query Usage Tips

1. **Adjust time ranges** based on your monitoring needs and data retention policies
2. **Modify thresholds** in queries to match your environment's baseline
3. **Create dashboards** combining multiple queries for a comprehensive view
4. **Set up alerts** using the alert query template for proactive monitoring
5. **Correlate with traffic data** by joining with `CiscoNexusInterfaceCounters` table for complete analysis
6. **Export results** to CSV for reporting or further analysis in Excel/PowerBI

## Integration with Azure Monitor

These queries can be integrated with Azure Monitor to create:
- **Workbooks**: Interactive dashboards combining multiple views
- **Alerts**: Automated notifications when thresholds are exceeded
- **Log Analytics**: Long-term trending and capacity planning
- **Power BI**: Executive dashboards and reporting
