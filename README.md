# arc-switch

ARC enabled switches - Network Tools and Parsers

This repository contains multiple tools and parsers for ARC-enabled network switches, including Cisco Nexus parsers and other network utilities.

## 📁 Repository Structure

```text
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
├── images/                                # Documentation images
├── TEST_SUMMARY.md                        # Unit test execution report
├── CODE_REVIEW.md                         # Code quality assessment
└── docs/                                  # Project documentation
```

## 🛠️ Available Tools

### 1. Cisco Nexus Parsers

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

#### IP ARP Parser
Parses ARP table information from Cisco Nexus switches.

### 2. Dell OS Parsers

**Location:** `src/SwitchOutput/DellOS/10/`

- **Interface Parser**: Dell interface configuration and status parsing
- **LLDP Syslog Parser**: LLDP neighbor discovery via syslog  
- **Version Parser**: Dell OS version information extraction

### 3. Syslog Tools

**Location:** `src/SyslogTools/`

#### Syslogwriter Library
A comprehensive Go library for writing JSON entries to Linux syslog systems with:
- Production-ready code quality
- Comprehensive unit test coverage (95%+)
- Statistics tracking and monitoring
- Flexible configuration options

#### Syslog Client
Utility for syslog client operations.

### 4. SNMP Monitoring

**Location:** `src/SwitchOutput/SnmpMonitor/`

SNMP polling utilities for network device monitoring.

## 🚀 Getting Started

1. **Choose your tool** from the available tools above
2. **Navigate to its directory** for specific documentation
3. **Use the tool-specific download script** for pre-built binaries
4. **Or build from source** using the instructions in each tool's README

## 🔗 Releases

Visit the [Releases page](https://github.com/microsoft/arc-switch/releases) to see all available pre-compiled binaries for different platforms.

## 🛠️ Development

Each tool in this repository is self-contained with its own:

- Source code and documentation
- Build instructions
- Download script for pre-built binaries
- Test files and examples

## ⚡ Automated Releases

This repository uses GitHub Actions to automatically build and release binaries for the following platforms:

- Linux (AMD64, ARM64)

Each release includes checksums for integrity verification and is available through tool-specific download scripts.
