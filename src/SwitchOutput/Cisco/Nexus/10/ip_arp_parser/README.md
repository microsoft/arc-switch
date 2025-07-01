# Cisco Nexus IP ARP Parser

This parser extracts and formats IP ARP table entries from Cisco Nexus switch output.

## 🚀 Quick Start - Download Pre-built Binaries

### Option 1: Using the Download Script (Recommended)

```bash
# Download and run the script
wget https://raw.githubusercontent.com/microsoft/arc-switch/main/src/SwitchOutput/Cisco/Nexus/10/ip_arp_parser/download-latest.sh
chmod +x download-latest.sh

# Download latest release for your platform
./download-latest.sh

# Or specify version and platform
./download-latest.sh v0.0.6-alpha.1 linux-amd64
```

**Supported platforms:**

- `linux-amd64` - Linux 64-bit (default)
- `linux-arm64` - Linux ARM64
- `windows-amd64` - Windows 64-bit
- `darwin-amd64` - macOS Intel
- `darwin-arm64` - macOS Apple Silicon

### Option 2: One-liner Download & Run

```bash
# Download and run in one command
bash <(wget -qO- https://raw.githubusercontent.com/microsoft/arc-switch/main/src/SwitchOutput/Cisco/Nexus/10/ip_arp_parser/download-latest.sh)
```

### Option 3: Manual Download

Visit the [Releases page](https://github.com/microsoft/arc-switch/releases) to download pre-compiled binaries for your platform.

## 🔧 Quick Usage with Pre-built Binary

Once downloaded, you can use the IP ARP parser:

```bash
# Extract the downloaded archive
tar -xzf ip_arp_parser-v0.0.6-alpha.1-linux-amd64.tar.gz

# Process a file and output to stdout
./ip_arp_parser -input show-ip-arp.txt

# Process a file and write to another file
./ip_arp_parser -input show-ip-arp.txt -output arp-results.json
```

## Description

The IP ARP parser processes the output of the `show ip arp` command from Cisco Nexus switches and converts it into structured JSON format. Each ARP entry becomes a separate JSON line, making it suitable for log analysis tools and databases.

## Installation

```bash
# Build the binary
go build -o ip_arp_parser ip_arp_parser.go
```

## Input Format

The parser expects output from the `show ip arp` command, which typically looks like:

```
RR1-S46-R14-93180hl-22-1b# show ip arp 

Flags: * - Adjacencies learnt on non-active FHRP router
       + - Adjacencies synced via CFSoE
       # - Adjacencies Throttled for Glean
       CP - Added via L2RIB, Control plane Adjacencies
       PS - Added via L2RIB, Peer Sync
       RO - Re-Originated Peer Sync Entry
       D - Static Adjacencies attached to down interface

IP ARP Table for context default
Total number of entries: 102
Address         Age       MAC Address     Interface       Flags
100.69.161.1    00:17:58  0000.0c9f.f0c9  Vlan201                  
100.69.161.75   00:03:39  02ec.a040.0001  Vlan201         +        
100.71.83.17    00:16:49  5ca6.2dbb.64a7  port-channel50           
```

## Usage

### Command Line Options

- `-input <file>`: Input file containing 'show ip arp' output
- `-output <file>`: Output file for JSON results (default: stdout)  
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

### Examples

```bash
# Parse from file to stdout
./ip_arp_parser -input show-ip-arp.txt

# Parse from file and save to output file
./ip_arp_parser -input show-ip-arp.txt -output arp-results.json

# Get data directly from switch using commands.json
./ip_arp_parser -commands ../commands.json -output arp-results.json

# Parse from file without output file (outputs to stdout)
./ip_arp_parser -input show-ip-arp.txt
```

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each ARP entry produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_arp_entry",
  "timestamp": "2025-07-01T22:55:29Z",
  "date": "2025-07-01",
  "message": {
    // ARP-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_arp_entry"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-07-01T22:55:29Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all ARP-specific data

### Message Fields

The `message` field contains all ARP-specific data:

#### Core ARP Data

- `ip_address`: IP address from ARP entry
- `age`: Age of the entry (HH:MM:SS or decimal seconds)
- `mac_address`: MAC address in Cisco format (xxxx.xxxx.xxxx)
- `interface`: Interface name (e.g., Vlan201, Ethernet1/47, port-channel50)
- `interface_type`: Categorized interface type (vlan, ethernet, port-channel, management, tunnel, loopback, other)

#### Flag Fields (only present when true)

- `non_active_fhrp`: * flag - Adjacencies learnt on non-active FHRP router
- `cfsoe_sync`: + flag - Adjacencies synced via CFSoE
- `throttled_glean`: # flag - Adjacencies Throttled for Glean
- `control_plane_l2rib`: CP flag - Added via L2RIB, Control plane Adjacencies
- `peer_sync_l2rib`: PS flag - Added via L2RIB, Peer Sync
- `re_originated_peer_sync`: RO flag - Re-Originated Peer Sync Entry
- `static_down_interface`: D flag - Static Adjacencies attached to down interface

#### Additional Fields

- `flags_raw`: Raw flags field for debugging (optional)

## Sample Output

```json
{"data_type":"cisco_nexus_arp_entry","timestamp":"2025-07-01T22:55:29Z","date":"2025-07-01","message":{"ip_address":"192.168.2.1","age":"00:17:58","mac_address":"1111.1111.0001","interface":"Vlan201","interface_type":"vlan"}}
{"data_type":"cisco_nexus_arp_entry","timestamp":"2025-07-01T22:55:29Z","date":"2025-07-01","message":{"ip_address":"192.168.2.200","age":"00:03:46","mac_address":"2222.a0c0.0001","interface":"Vlan201","cfsoe_sync":true,"interface_type":"vlan","flags_raw":"+"}}
```

## Building

```bash
go build -o ip_arp_parser ip_arp_parser.go
```

## Testing

```bash
go test
```

## Direct Switch Integration

The parser can get data directly from the switch using the commands JSON file. This requires:

1. A `commands.json` file with command definitions
2. Network connectivity to the switch
3. Proper VSH (Virtual Shell) access

When using the `-commands` option, the parser will:
1. Load the commands JSON file
2. Find the command with name "arp-table" 
3. Execute the command using VSH
4. Parse the output and generate JSON results

## Error Handling

The parser handles various error conditions:
- Invalid input files
- Malformed ARP table entries
- VSH execution failures
- JSON encoding errors

## Interface Type Detection

The parser automatically categorizes interfaces based on naming patterns:
- `Vlan*` → vlan
- `Ethernet*`, `Eth*` → ethernet  
- `port-channel*`, `Po*` → port-channel
- `mgmt*` → management
- `Tunnel*` → tunnel
- `Loopbook*` → loopback
- Others → other

## Validation

To validate that the parser output conforms to the standardized JSON structure, use the project's validation script:

```bash
# Validate parser output
./ip_arp_parser -input show-ip-arp.txt | /workspaces/arc-switch2/validate-parser-output.sh

# Or validate a saved output file
./ip_arp_parser -input show-ip-arp.txt -output arp-results.json
/workspaces/arc-switch2/validate-parser-output.sh arp-results.json
```

The validation script checks for:
- Presence of all required fields (`data_type`, `timestamp`, `date`, `message`)
- Correct timestamp format (ISO 8601)
- Correct date format (YYYY-MM-DD)
- Valid JSON structure

## Compatibility

Tested with Cisco Nexus switches running NX-OS. The parser should work with various Nexus models that support the standard `show ip arp` command format.
