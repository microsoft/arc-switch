# Cisco Nexus MAC Address Table Parser

This tool parses the output of the `show mac address-table` command from Cisco Nexus switches and converts each entry to JSON format.

## 🚀 Quick Start - Download Pre-built Binaries

### Option 1: Using the Download Script (Recommended)

```bash
# Download and run the script
wget https://raw.githubusercontent.com/microsoft/arc-switch/main/src/SwitchOutput/Cisco/Nexus/10/mac_address_parser/download-latest.sh
chmod +x download-latest.sh

# Download latest release for your platform
./download-latest.sh

# Or specify version and platform
./download-latest.sh v0.0.3-alpha.1 linux-amd64
```

**Supported platforms:**

- `linux-amd64` - Linux 64-bit (default)
- `linux-arm64` - Linux ARM64


### Option 2: One-liner Download & Run

```bash
# Download and run in one command
bash <(wget -qO- https://raw.githubusercontent.com/microsoft/arc-switch/main/src/SwitchOutput/Cisco/Nexus/10/mac_address_parser/download-latest.sh)
```

### Option 3: Manual Download

Visit the [Releases page](https://github.com/microsoft/arc-switch/releases) to download pre-compiled binaries for your platform.

## 🔧 Quick Usage with Pre-built Binary

Once downloaded, you can use the MAC address parser:

```bash
# Extract the downloaded archive
tar -xzf mac_address_parser-v0.0.3-alpha.1-linux-amd64.tar.gz

# Process a file and output to stdout
./mac_address_parser -input show-mac-address-table.txt

# Process a file and write to another file
./mac_address_parser -input show-mac-address-table.txt -output mac-table.json

# Process a JSON commands file and run the MAC table command
./mac_address_parser -commands network-commands.json -output mac-table.json
```

## Features

- Parses Cisco Nexus MAC address table output
- Extracts all fields from the legend and table
- Outputs each entry as a JSON object
- Support for reading from file or stdin
- Support for writing to file or stdout
- Uses `vsh` CLI for Cisco Nexus Linux shell compatibility
- Cross-platform binaries available

## 🛠️ Building from Source

If you prefer to build from source:

```bash
# Clone the repository
git clone https://github.com/microsoft/arc-switch.git
cd arc-switch/src/SwitchOutput/Cisco/Nexus/10/mac_address_parser

# Build the binary
go build -o mac_address_parser mac_address_parser.go

# Run it
./mac_address_parser --help
```

## Usage

```bash
# Process a file and output to stdout
./mac_address_parser -input show-mac-address-table.txt

# Process a file and write to another file
./mac_address_parser -input show-mac-address-table.txt -output mac-table.json

# Process a JSON commands file and run the MAC table command
./mac_address_parser -commands network-commands.json -output mac-table.json

# Note: You must specify exactly one of -input or -commands, not both
```

## JSON Output Format

Each MAC address table entry is output as a JSON object with the following fields:

```json
{
  "data_type": "cisco_nexus_mac_table",     // Identifies the type of data for KQL queries
  "timestamp": "06/23/2025 03:04:05 PM",    // Timestamp in Windows/.NET format
  "date": "06/23/2025",                     // Date in MM/dd/yyyy format
  "primary_entry": true,                    // * indicates primary entry
  "gateway_mac": false,                     // G indicates Gateway MAC
  "routed_mac": false,                      // (R) indicates Routed MAC
  "overlay_mac": false,                     // O indicates Overlay MAC
  "vlan": "7",                              // VLAN ID
  "mac_address": "02ec.a004.0000",          // MAC address
  "type": "dynamic",                        // Type of entry (dynamic, static, etc.)
  "age": "NA",                              // Age (seconds since last seen)
  "secure": "F",                            // Secure flag (T/F)
  "ntfy": "F",                              // NTFY flag (T/F)
  "port": "Eth1/1",                         // Port identifier
  "vpc_peer_link": false                    // + indicates primary entry using vPC Peer-Link
}
```

## Commands JSON Format

When using the `-commands` option, provide a JSON file with the following structure:

```json
{
  "commands": [
    {
      "name": "mac-address-table",
      "command": "show mac address-table"
    },
    {
      "name": "other-command",
      "command": "show something else"
    }
  ]
}
```

The parser will look for a command with the name `"mac-address-table"` in the JSON file and execute it.

## Building

```bash
go build -o mac_address_parser mac_address_parser.go
```

## Testing

To test with the included sample file:

```bash
go build -o mac_address_parser mac_address_parser.go
./mac_address_parser -input ../show-mac-address-table.txt
```

## Legend Interpretation

The parser handles all the legend elements from the Cisco Nexus output:

```plaintext
Legend: 
        * - primary entry, G - Gateway MAC, (R) - Routed MAC, O - Overlay MAC
        age - seconds since last seen,+ - primary entry using vPC Peer-Link,
        (T) - True, (F) - False, C - ControlPlane MAC, ~ - vsan,
        (NA)- Not Applicable
```

Each of these elements is mapped to a corresponding field in the JSON output.

## KQL Query Examples

The `data_type` field enables easy filtering in Azure Log Analytics or other systems that use KQL (Kusto Query Language).

```kql
// Filter for Cisco Nexus MAC table entries only
| where data_type == "cisco_nexus_mac_table"

// Find all MAC addresses on a specific VLAN
| where data_type == "cisco_nexus_mac_table" and vlan == "7"

// Get MAC address distribution by port
| where data_type == "cisco_nexus_mac_table"
| summarize count() by port
| order by count_ desc

// Find gateway MACs
| where data_type == "cisco_nexus_mac_table" and gateway_mac == true

// Find entries from a specific date
| where data_type == "cisco_nexus_mac_table" and date == "06/23/2025"
```
