# Arc-Switch

ARC-enabled switches - Network Tools and Parsers

This repository contains multiple tools and parsers for ARC-enabled network switches, including Cisco Nexus parsers and other network utilities.

## Repository Structure

```plaintext
arc-switch2/
├── src/
│   ├── SwitchOutput/
│   │   ├── Cisco/Nexus/10/
│   │   │   ├── interface_counters_parser/  # Interface counters parser
│   │   │   ├── ip_arp_parser/             # IP ARP table parser
│   │   │   └── mac_address_parser/        # MAC address table parser
│   │   ├── DellOS/
│   │   │   ├── README.md                  # Dell OS documentation
│   │   │   └── 10/
│   │   │       ├── interface/             # Dell interface parsers
│   │   │       ├── interface_phyeth/      # Physical ethernet interface parser
│   │   │       ├── lldp-sylog/           # LLDP syslog parser
│   │   │       └── Version/               # Version information parser
│   │   └── SnmpMonitor/
│   │       └── PollSnmp/                  # SNMP polling utilities
│   └── SyslogTools/
│       ├── syslog-client/                 # Syslog client utility
│       └── syslogwriter/                  # Syslog writer library
└── images/                                # Documentation images
```

## Available Tools

### Cisco Nexus Parsers

**Location:** `src/SwitchOutput/Cisco/Nexus/10/`

#### MAC Address Table Parser

Parses the output of the `show mac address-table` command from Cisco Nexus switches and converts each entry to JSON format.

**Quick Download:**

```bash
wget https://raw.githubusercontent.com/microsoft/arc-switch/main/src/SwitchOutput/Cisco/Nexus/10/mac_address_parser/download-latest.sh
chmod +x download-latest.sh
./download-latest.sh
```

#### Interface Counters Parser

Parses interface counter data from Cisco Nexus switches with comprehensive testing coverage.

**Quick Download:**

```bash
wget https://raw.githubusercontent.com/microsoft/arc-switch/main/src/SwitchOutput/Cisco/Nexus/10/interface_counters_parser/download-latest.sh
chmod +x download-latest.sh
./download-latest.sh
```

#### IP ARP Parser

Parses ARP table information from Cisco Nexus switches.

**Quick Download:**

```bash
wget https://raw.githubusercontent.com/microsoft/arc-switch/main/src/SwitchOutput/Cisco/Nexus/10/ip_arp_parser/download-latest.sh
chmod +x download-latest.sh
./download-latest.sh
```

### Cisco Nexus Interface Error Types and Monitoring

The Interface Error Counters Parser (`show interface counters errors`) tracks critical network health metrics. Understanding these errors is essential for maintaining network reliability and troubleshooting connectivity issues.

#### Error Type Breakdown

##### Section 1: Basic Error Counters

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

##### Section 2: Collision Counters

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

##### Section 3: MAC and Physical Layer Errors

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

##### Section 4: Discard Counters

**InDiscards (Input Discards)**
- **What it is**: Inbound packets dropped by the switch
- **Why it matters**: Indicates resource exhaustion or policy drops on ingress
- **Action threshold**: Increasing trend suggests capacity issues
- **Common causes**: Input queue full, ACL drops, rate limiting, buffer exhaustion

##### Section 5: CRC Errors

**Stomped-CRC (Stomped CRC Errors)**
- **What it is**: CRC intentionally invalidated by switch for internal signaling (Ethernet interfaces only)
- **Why it matters**: Switch-specific mechanism; usually not a concern unless excessive
- **Action threshold**: Vendor-specific; consult Cisco documentation
- **Common causes**: Internal switch operations, cut-through switching artifacts

### KQL Monitoring Queries for Azure Data Explorer

These queries help monitor Cisco Nexus interface errors when data is ingested into Azure Data Explorer (Kusto). Assumes the table name is `CiscoNexusInterfaceErrors` with the standard schema from the parser.

#### Summary Dashboard View - Big Picture Health Check

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

#### Critical Physical Layer Issues - Actionable Alerts

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

#### Duplex Mismatch Detection

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

#### Congestion and Discard Monitoring

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

#### Hardware Failure Detection

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

#### Trending Analysis - Error Rate Over Time

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

#### Interface Type Comparison

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

#### Top Problematic Interfaces

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

#### Detailed Interface Investigation View

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

#### Alert Configuration Template

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

### Query Usage Tips

1. **Adjust time ranges** based on your monitoring needs and data retention policies
2. **Modify thresholds** in queries to match your environment's baseline
3. **Create dashboards** combining multiple queries for a comprehensive view
4. **Set up alerts** using the alert query template for proactive monitoring
5. **Correlate with traffic data** by joining with `CiscoNexusInterfaceCounters` table for complete analysis
6. **Export results** to CSV for reporting or further analysis in Excel/PowerBI

### Integration with Azure Monitor

These queries can be integrated with Azure Monitor to create:
- **Workbooks**: Interactive dashboards combining multiple views
- **Alerts**: Automated notifications when thresholds are exceeded
- **Log Analytics**: Long-term trending and capacity planning
- **Power BI**: Executive dashboards and reporting

### Dell OS Parsers

**Location:** `src/SwitchOutput/DellOS/10/`

- **Interface Parser**: Dell interface configuration and status parsing
- **LLDP Syslog Parser**: LLDP neighbor discovery via syslog
- **Version Parser**: Dell OS version information extraction

### Syslog Tools

**Location:** `src/SyslogTools/`

#### Syslogwriter Library

A comprehensive Go library for writing JSON entries to Linux syslog systems with:

- Production-ready code quality
- Comprehensive unit test coverage (95%+)
- Statistics tracking and monitoring
- Flexible configuration options

#### Syslog Client

Utility for syslog client operations.

### SNMP Monitoring

**Location:** `src/SwitchOutput/SnmpMonitor/`

SNMP polling utilities for network device monitoring.

## Getting Started

1. **Choose your tool** from the available tools above
2. **Navigate to its directory** for specific documentation
3. **Use the tool-specific download script** for pre-built binaries
4. **Or build from source** using the instructions in each tool's README

## Releases

Visit the [Releases page](https://github.com/microsoft/arc-switch/releases) to see all available pre-compiled binaries for different platforms.

## Development

Each tool in this repository is self-contained with its own:

- Source code and documentation
- Build instructions
- Download script for pre-built binaries
- Test files and examples

## Automated Releases

This repository uses GitHub Actions to automatically build and release binaries for the following platforms:

- Linux (AMD64, ARM64)

Each release includes checksums for integrity verification and is available through tool-specific download scripts.
