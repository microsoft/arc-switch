# arc-switch

ARC enabled switches - Network Tools and Parsers

This repository contains multiple tools and parsers for ARC-enabled network switches, including Cisco Nexus parsers and other network utilities.

## � Available Tools

### 1. Cisco Nexus MAC Address Table Parser

**Location:** `src/SwitchOutput/Cisco/Nexus/10/mac_address_parser/`

This tool parses the output of the `show mac address-table` command from Cisco Nexus switches and converts each entry to JSON format.

**Quick Download:**

```bash
# Download the parser-specific script
wget https://raw.githubusercontent.com/microsoft/arc-switch/main/src/SwitchOutput/Cisco/Nexus/10/mac_address_parser/download-latest.sh
chmod +x download-latest.sh
./download-latest.sh
```

**Documentation:** See [MAC Address Parser README](src/SwitchOutput/Cisco/Nexus/10/mac_address_parser/README.md)

### 2. Future Tools

Additional network parsing tools will be added in their respective directories with their own download scripts and documentation.

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

## � Repository Structure

```text
arc-switch/
├── src/SwitchOutput/
│   ├── Cisco/Nexus/10/
│   │   └── mac_address_parser/     # MAC address table parser
│   └── DellOS10.5/                 # Dell OS parsers
└── .github/workflows/              # CI/CD automation
```

## ⚡ Automated Releases

This repository uses GitHub Actions to automatically build and release binaries for the following platforms:

- Linux (AMD64, ARM64)

Each release includes checksums for integrity verification and is available through tool-specific download scripts.
