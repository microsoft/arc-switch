# Dell OS10.5 Interface Data Capture

This Go script is designed to run on Dell switches using the Linux-based OS10.5 (Debian 10/Buster). Its primary function is to collect detailed interface statistics from the switch, convert the unstructured CLI output into structured JSON, and forward this data to syslog for integration with Azure Arc.
![Interface Counter Example](../../../../images/interface-counter.png)

## Key Features

- **Flexible Data Input:**  
  The script can either execute the `show interface` CLI command directly on the switch or process a provided text file containing the command's output (useful for testing or offline analysis).

- **Data Parsing & Structuring:**  
  It parses the raw, unstructured CLI output and extracts relevant interface details, statistics, and rates, organizing them into a well-defined JSON structure.

- **Syslog Integration for Azure Arc:**  
  Each interface's JSON data is sent to the local syslog using the `logger` command. This enables Azure Arc to collect and forward the data to Azure, where it can be stored and analyzed for operational insights.

- **Test Mode:**  
  A test mode is available to validate parsing and output without executing system commands or sending data to syslog.

- **Debian Package:**  
  The script can be packaged as a Debian package with systemd service integration for automatic data collection.

## Usage

### Manual Execution

- **Normal Mode:**  
  The script runs on the switch and collects live data:

  ```shell
  ./show_interface
  ```

- **Test Mode with Input File:**  
  For development or troubleshooting, provide a CLI output file:\

  ```shell
  ./show_interface -test -inputfile <path_to_cli_output.txt>
  ```

### Debian Package Installation

The interface syslog service can be packaged as a Debian package for easy installation and automatic systemd service management.

#### Building the Debian Package

```shell
# Build the package only
./build_interface_syslog_deb.sh

# Build and install the package
./build_interface_syslog_deb.sh --install
```

#### Installing the Package

```shell
# Install the package
sudo dpkg -i interface-syslog_<version>_amd64.deb

# Check service status
sudo systemctl status interface-syslog.timer
sudo systemctl status interface-syslog.service

# View service logs
sudo journalctl -u interface-syslog.service -f
```

#### Removing the Package

```shell
# Remove the package and stop services
sudo apt remove interface-syslog

# Or use dpkg directly
sudo dpkg -r interface-syslog
```

#### Package Details

- **Package name:** `interface-syslog`
- **Installation path:** `/opt/microsoft/interface-syslog`
- **Service files:** 
  - `/etc/systemd/system/interface-syslog.service`
  - `/etc/systemd/system/interface-syslog.timer`
- **Execution frequency:** Every 1 minute (controlled by systemd timer)
- **User:** root
- **Output:** Syslog integration for Azure Arc collection

## Requirements

- Go runtime (for building/running the script)
- Debian 10 (Buster) or compatible environment
- Syslog (`logger` command) available on the switch

## Build Command

```shell
go build show_interface.go
```

## Integration with Azure Arc

By forwarding structured interface data to syslog, this script enables seamless collection by Azure Arc agents, supporting centralized monitoring and analytics in Azure.

## KQL query

Data from the syslog table:

- [Data in raw CSV format](./azure-syslog-table-data.csv)
- [Data transformed by query to csv](./azure-syslog-data-transformed.csv)
- [KQL query](./example_interface.kql)

```kql
Syslog 
| where ProcessName == "interface" // Filter for the processname "interface"
| extend jsonData = parse_json(SyslogMessage)  // Parse the JSON data in the SyslogMessage field
| where jsonData.name == "Ethernet 1/1/47:1"  // Filter for the specific interface name
| project 
    TimeGenerated,  // Include the timestamp
    name = jsonData.name,  // Interface name
    pkts = tolong(jsonData.input_stat.pkts),  // Input packets
    bytes = tolong(jsonData.input_stat.bytes),  // Input bytes
    multicasts = tolong(jsonData.input_stat.multicasts),  // Input multicasts
    unicasts = tolong(jsonData.input_stat.unicasts)  // Input unicasts
| order by TimeGenerated asc  // Order by timestamp
| extend 
    prev_pkts = prev(pkts),  // Get the previous value of pkts
    prev_bytes = prev(bytes),  // Get the previous value of bytes
    prev_multicasts = prev(multicasts),  // Get the previous value of multicasts
    prev_unicasts = prev(unicasts)  // Get the previous value of unicasts
| extend 
    delta_pkts = pkts - prev_pkts,  // Calculate the difference in pkts
    delta_bytes = bytes - prev_bytes,  // Calculate the difference in bytes
    delta_multicasts = multicasts - prev_multicasts,  // Calculate the difference in multicasts
    delta_unicasts = unicasts - prev_unicasts  // Calculate the difference in unicasts
| project 
    TimeGenerated, 
    name, 
    delta_pkts, 
    delta_bytes, 
    delta_multicasts, 
    delta_unicasts  // Select the final columns to display
| render timechart
    with (
    title="Interface Ethernet 1/1/47:1 Counters",
    ytitle="Delta Values",
    xtitle="Time")
```

## Example Switch Output

[Example Switch CLI Output](./example_show_interface.txt)

```shell
Ethernet 1/1/1 is up, line protocol is up
Description: CL01 Nodes NIC
Hardware is Eth, address is 0c:29:ef:c4:cf:c5
    Current address is 0c:29:ef:c4:cf:c5
Pluggable media present, SFP28 type is SFP28 25GBASE-CR-3.0M
    Wavelength is 256
    Configured media fec option is none
Interface index is 16
Internet address is not set
Mode of IPv4 Address Assignment: not set
    Interface IPv6 oper status: Disabled
IP Unreachables status: Disabled
MTU 9216 bytes, IP MTU 9184 bytes
LineSpeed 25G, Auto-Negotiation on, Link-Training on
Configured FEC is cl108-rs, Negotiated FEC is cl108-rs
Flowcontrol rx off tx off
ARP type: ARPA, ARP Timeout: 60
Tag Protocol IDentifier (TPID) value: 0x8100
Last clearing of "show interface" counters: 48 weeks 4 days 23:12:25
Queuing strategy: fifo
Input statistics:
     258756070469 packets, 224206026489044 octets
     1025053856 64-byte pkts, 45025679570 over 64-byte pkts, 14474324027 over 127-byte pkts
     4443391137 over 255-byte pkts, 5103135246 over 511-byte pkts, 1.88684486633e+11 over 1023-byte pkts
     3322520 Multicasts, 5843785 Broadcasts, 258746758082 Unicasts
     0 runts, 0 giants, 104 throttles
     0 CRC, 0 overrun, 0 discarded
Output statistics:
     464017689029 packets, 460659024245083 octets
     1003276903 64-byte pkts, 31761457350 over 64-byte pkts, 15485906560 over 127-byte pkts
     5113075813 over 255-byte pkts, 2082869404 over 511-byte pkts, 4.08571102999e+11 over 1023-byte pkts
     114945286 Multicasts, 27839366 Broadcasts, 463860550477 Unicasts
     0 throttles, 296897 discarded, 0 Collisions,  wred drops
Rate Info(interval 30 seconds):
     Input 11 Mbits/sec, 1806 packets/sec, 0% of line rate
     Output 14 Mbits/sec, 2102 packets/sec, 0% of line rate
Time since last interface status change: 48 weeks 4 days 23:11:15

Ethernet 1/1/9 is down, line protocol is down
Description: Unused Port
Hardware is Eth, address is 0c:29:ef:c4:cf:29
    Current address is 0c:29:ef:c4:cf:29
Pluggable media not present

Interface index is 24
Internet address is not set
Mode of IPv4 Address Assignment: not set
    Interface IPv6 oper status: Disabled
IP Unreachables status: Disabled
MTU 9216 bytes, IP MTU 9184 bytes
LineSpeed 0, Auto-Negotiation off, Link-Training off
Configured FEC is off, Negotiated FEC is off
Flowcontrol rx off tx off
ARP type: ARPA, ARP Timeout: 60
Tag Protocol IDentifier (TPID) value: 0x8100
Last clearing of "show interface" counters: 48 weeks 4 days 23:12:24
Queuing strategy: fifo
Input statistics:
     0 packets, 0 octets
     0 64-byte pkts, 0 over 64-byte pkts, 0 over 127-byte pkts
     0 over 255-byte pkts, 0 over 511-byte pkts, 0 over 1023-byte pkts
     0 Multicasts, 0 Broadcasts, 0 Unicasts
     0 runts, 0 giants, 0 throttles
     0 CRC, 0 overrun, 0 discarded
Output statistics:
     0 packets, 0 octets
     0 64-byte pkts, 0 over 64-byte pkts, 0 over 127-byte pkts
     0 over 255-byte pkts, 0 over 511-byte pkts, 0 over 1023-byte pkts
     0 Multicasts, 0 Broadcasts, 0 Unicasts
     0 throttles, 0 discarded, 0 Collisions,  wred drops
Rate Info(interval 30 seconds):
     Input 0 Mbits/sec, 0 packets/sec, 0% of line rate
     Output 0 Mbits/sec, 0 packets/sec, 0% of line rate
Time since last interface status change: 48 weeks 4 days 23:12:24

Ethernet 1/1/47:1 is up, line protocol is up
Description: P2P_Rack00/B2_To_Rack01/Tor1
Hardware is Eth, address is 0c:29:ef:c4:cf:4f
    Current address is 0c:29:ef:c4:cf:4f
Pluggable media present, SFP+ type is SFP+ 10GBASE-SR
    Wavelength is 850
    Configured media fec option is none
Interface index is 70
Internet address is 100.71.7.10/30
Mode of IPv4 Address Assignment: MANUAL
    Interface IPv6 oper status: Enabled
IP Unreachables status: Disabled
Link local IPv6 address: fe80::e29:efff:fec4:cf4f/64
MTU 9216 bytes, IP MTU 9184 bytes
LineSpeed 10G, Auto-Negotiation off, Link-Training off
Flowcontrol rx off tx off
ARP type: ARPA, ARP Timeout: 60
Tag Protocol IDentifier (TPID) value: 0x8100
Last clearing of "show interface" counters: 48 weeks 4 days 23:12:17
Queuing strategy: fifo
Input statistics:
     1524928866 packets, 2172543017872 octets
     41449632 64-byte pkts, 11036802 over 64-byte pkts, 5158896 over 127-byte pkts
     12771553 over 255-byte pkts, 8756099 over 511-byte pkts, 1445755884 over 1023-byte pkts
     3239848 Multicasts, 8 Broadcasts, 1521689010 Unicasts
     0 runts, 0 giants, 0 throttles
     0 CRC, 0 overrun, 0 discarded
Output statistics:
     504114665 packets, 483481084525 octets
     142502695 64-byte pkts, 23925746 over 64-byte pkts, 2912093 over 127-byte pkts
     16604400 over 255-byte pkts, 17726640 over 511-byte pkts, 300443091 over 1023-byte pkts
     4564451 Multicasts, 1 Broadcasts, 499550213 Unicasts
     0 throttles, 0 discarded, 0 Collisions,  wred drops
Rate Info(interval 30 seconds):
     Input 0 Mbits/sec, 1 packets/sec, 0% of line rate
     Output 0 Mbits/sec, 7 packets/sec, 0% of line rate
Time since last interface status change: 7 weeks 6 days 03:21:10


Ethernet 1/1/49 is up, line protocol is up
Hardware is Eth, address is 0c:29:ef:c4:cf:51
    Current address is 0c:29:ef:c4:cf:51
Pluggable media present, QSFP28-DD type is QSFP28-DD 200GBASE--1.0M
    Wavelength is 0
    Configured media fec option is none
Interface index is 40
Internet address is not set
Mode of IPv4 Address Assignment: not set
    Interface IPv6 oper status: Disabled
IP Unreachables status: Disabled
MTU 9216 bytes, IP MTU 9184 bytes
LineSpeed 100G, Auto-Negotiation off, Link-Training off
Configured FEC is off, Negotiated FEC is off
Flowcontrol rx off tx off
ARP type: ARPA, ARP Timeout: 60
Tag Protocol IDentifier (TPID) value: 0x0
Last clearing of "show interface" counters: 48 weeks 4 days 23:12:22
Queuing strategy: fifo
Input statistics:
     651340662125 packets, 632982195375472 octets
     202042142 64-byte pkts, 56951889381 over 64-byte pkts, 21262835316 over 127-byte pkts
     3965313138 over 255-byte pkts, 6727551195 over 511-byte pkts, 5.62231030953e+11 over 1023-byte pkts
     30592437 Multicasts, 3099182 Broadcasts, 651233331044 Unicasts
     0 runts, 0 giants, 0 throttles
     0 CRC, 0 overrun, 0 discarded
Output statistics:
     520124946595 packets, 482475641693829 octets
     240112960 64-byte pkts, 64365547020 over 64-byte pkts, 22289116833 over 127-byte pkts
     4535331559 over 255-byte pkts, 3256205485 over 511-byte pkts, 4.25438632738e+11 over 1023-byte pkts
     30271980 Multicasts, 3114248 Broadcasts, 519971468894 Unicasts
     0 throttles, 524703 discarded, 0 Collisions,  wred drops
Rate Info(interval 30 seconds):
     Input 28 Mbits/sec, 4063 packets/sec, 0% of line rate
     Output 17 Mbits/sec, 2670 packets/sec, 0% of line rate
Time since last interface status change: 48 weeks 4 days 23:07:52

Loopback 0 is up, line protocol is up
Description: Loopback_Rack01/Tor1
Hardware is Loopback
Interface index is 72
Internet address is 100.71.7.24/32
Mode of IPv4 Address Assignment: MANUAL
    Interface IPv6 oper status: Enabled
Link local IPv6 address: fe80::988d:f1ff:fe19:75b7/64
MTU 9216 bytes, IP MTU 9184 bytes
Flowcontrol rx off tx off
ARP type: ARPA, ARP Timeout: 60
Last clearing of "show interface" counters: 48 weeks 4 days 23:12:17
Queuing strategy: fifo
Input statistics:
     Input 0 packets, 0 bytes, 0 multicast
     Received 0 errors, 0 discarded
Output statistics:
     Output 1 packets, 86 bytes,  multicast
     Output 0 errors, Output  invalid protocol
Time since last interface status change: 48 weeks 4 days 23:12:16


Management 1/1/1 is up, line protocol is down
Hardware is Eth, address is 0c:29:ef:c4:cf:20
    Current address is 0c:29:ef:c4:cf:20
Interface index is 10
Internet address is not set
Mode of IPv4 Address Assignment: DHCP
    Interface IPv6 oper status: Enabled
Virtual-IP is not set
Virtual-IP IPv6 address is not set
MTU 1532 bytes, IP MTU 1500 bytes
LineSpeed 0
Flowcontrol rx off tx off
ARP type: ARPA, ARP Timeout: 60
Last clearing of "show interface" counters: 48 weeks 4 days 23:12:45
Queuing strategy: fifo
Input statistics:
     Input 0 packets, 0 bytes, 0 multicast
     Received 0 errors, 0 discarded
Output statistics:
     Output 0 packets, 0 bytes, 0 multicast
     Output 0 errors, Output  invalid protocol
Time since last interface status change: 48 weeks 4 days 23:12:16



Port-channel 1000 is up, line protocol is up
Address is 0c:29:ef:c4:cf:f6, Current address is 0c:29:ef:c4:cf:f6
Interface index is 87
Internet address is not set
Mode of IPv4 Address Assignment: not set
    Interface IPv6 oper status: Disabled
IP Unreachables status: Disabled
MTU 9216 bytes, IP MTU 9184 bytes
LineSpeed 400G
Minimum number of links to bring Port-channel up is 1
Maximum active members that are allowed in the portchannel is 32
Members in this channel:
ARP type: ARPA, ARP Timeout: 60
Tag Protocol IDentifier (TPID) value: 0x8100
Last clearing of "show interface" counters: 48 weeks 4 days 23:11:36
Queuing strategy: fifo
Input statistics:
     2549097110546 packets, 2468196758772796 octets
     769025132 64-byte pkts, 227885030122 over 64-byte pkts, 89178965053 over 127-byte pkts
     15622467516 over 255-byte pkts, 17961134860 over 511-byte pkts, 2.197680487863e+12 over 1023-byte pkts
     139156443 Multicasts, 11612512 Broadcasts, 2548674316248 Unicasts
     0 runts, 0 giants, 0 throttles
     0 CRC, 0 overrun, 0 discarded
Output statistics:
     2072896808054 packets, 1914711315598254 octets
     1011582909 64-byte pkts, 260414799774 over 64-byte pkts, 91087851163 over 127-byte pkts
     17969133261 over 255-byte pkts, 11483328085 over 511-byte pkts, 1.690930112862e+12 over 1023-byte pkts
     137422034 Multicasts, 24082491 Broadcasts, 2072213667294 Unicasts
     0 throttles, 2097709 discarded, 0 Collisions,  wred drops
Rate Info(interval 30 seconds):
     Input 124 Mbits/sec, 18546 packets/sec, 0% of line rate
     Output 102 Mbits/sec, 16437 packets/sec, 0% of line rate
Time since last interface status change: 48 weeks 4 days 23:07:52


Vlan 2 is down, line protocol is down
Description: Unused_Ports
Address is 0c:29:ef:c4:cf:c1, Current address is 0c:29:ef:c4:cf:c1
Mac Learning is enabled
Interface index is 85
Internet address is not set
Mode of IPv4 Address Assignment: not set
    Interface IPv6 oper status: Enabled
IP Unreachables status: Disabled
MTU 9216 bytes, IP MTU 9184 bytes
LineSpeed 0
ARP type: ARPA, ARP Timeout: 60
Last clearing of "show interface" counters: 48 weeks 4 days 23:11:49
Queuing strategy: fifo
Input statistics:
    14647866 packets, 1054646352 octets
Output statistics:
    0 packets, 0 octets
Time since last interface status change: 48 weeks 4 days 23:11:48


Vlan 7 is up, line protocol is up
Description: Rack01-CL01-SU01-Infra
Address is 0c:29:ef:c4:cf:c1, Current address is 0c:29:ef:c4:cf:c1
Mac Learning is enabled
Interface index is 83
Internet address is 100.68.80.2/24
Mode of IPv4 Address Assignment: MANUAL
    Interface IPv6 oper status: Enabled
IP Unreachables status: Disabled
Link local IPv6 address: fe80::e29:efff:fec4:cfc1/64
MTU 9216 bytes, IP MTU 9184 bytes
LineSpeed 10G
ARP type: ARPA, ARP Timeout: 60
Last clearing of "show interface" counters: 48 weeks 4 days 23:11:49
Queuing strategy: fifo
Input statistics:
    231458505085 packets, 228406157157713 octets
Output statistics:
    233954943476 packets, 232071489919154 octets
Time since last interface status change: 48 weeks 4 days 23:11:28


Vlan 107 is up, line protocol is up
Description: Rack01-CL01-SU01-Stor
Address is 0c:29:ef:c4:cf:c1, Current address is 0c:29:ef:c4:cf:c1
Mac Learning is enabled
Interface index is 74
Internet address is 100.73.0.2/25
Mode of IPv4 Address Assignment: MANUAL
    Interface IPv6 oper status: Enabled
IP Unreachables status: Disabled
Link local IPv6 address: fe80::e29:efff:fec4:cfc1/64
MTU 9216 bytes, IP MTU 9184 bytes
LineSpeed 10G
ARP type: ARPA, ARP Timeout: 60
Last clearing of "show interface" counters: 48 weeks 4 days 23:11:49
Queuing strategy: fifo
Input statistics:
    4938296732684 packets, 4661763365869042 octets
Output statistics:
    4938330231730 packets, 4661767155256777 octets
Time since last interface status change: 48 weeks 4 days 23:11:28
```

## Example of converted JSON data

[Example JSON Data](./example_show_interface.json)

```JSON
{
  "interfaces": [
    {
      "name": "Loopback 0",
      "status": "up",
      "line_protocol": "up",
      "description": "Loopback_Rack01/Tor1",
      "hardware": "Loopback",
      "interface_index": 72,
      "ipv4_address": "100.71.7.24/32",
      "ipv4_assignment_mode": "MANUAL",
      "ipv6_oper_status": "Enabled",
      "ipv6_link_local_address": "fe80::988d:f1ff:fe19:75b7/64",
      "mtu": 9216,
      "ip_mtu": 9184,
      "flowcontrol": {
        "rx": "off",
        "tx": "off"
      },
      "arp": {
        "type": "ARPA",
        "timeout": 60
      },
      "counters_last_cleared": "48 weeks 4 days 23:12:17",
      "queuing_strategy": "fifo",
      "input_statistics": {
        "packets": 0,
        "bytes": 0,
        "multicast": 0,
        "errors": 0,
        "discarded": 0
      },
      "output_statistics": {
        "packets": 1,
        "bytes": 86,
        "multicast": 0,
        "errors": 0,
        "invalid_protocol": 0
      },
      "time_since_last_status_change": "48 weeks 4 days 23:12:16"
    },
    {
      "name": "Management 1/1/1",
      "status": "up",
      "line_protocol": "down",
      "hardware": "Eth",
      "mac_address": "0c:29:ef:c4:cf:20",
      "interface_index": 10,
      "ipv4_address": null,
      "ipv4_assignment_mode": "DHCP",
      "ipv6_oper_status": "Enabled",
      "virtual_ip": null,
      "virtual_ipv6_address": null,
      "mtu": 1532,
      "ip_mtu": 1500,
      "line_speed": 0,
      "flowcontrol": {
        "rx": "off",
        "tx": "off"
      },
      "arp": {
        "type": "ARPA",
        "timeout": 60
      },
      "counters_last_cleared": "48 weeks 4 days 23:12:45",
      "queuing_strategy": "fifo",
      "input_statistics": {
        "packets": 0,
        "bytes": 0,
        "multicast": 0,
        "errors": 0,
        "discarded": 0
      },
      "output_statistics": {
        "packets": 0,
        "bytes": 0,
        "multicast": 0,
        "errors": 0,
        "invalid_protocol": 0
      },
      "time_since_last_status_change": "48 weeks 4 days 23:12:16"
    },
    {
      "name": "Port-channel 1000",
      "status": "up",
      "line_protocol": "up",
      "mac_address": "0c:29:ef:c4:cf:f6",
      "interface_index": 87,
      "ipv4_address": null,
      "ipv4_assignment_mode": "not set",
      "ipv6_oper_status": "Disabled",
      "ip_unreachables_status": "Disabled",
      "mtu": 9216,
      "ip_mtu": 9184,
      "line_speed": "400G",
      "minimum_links": 1,
      "maximum_active_members": 32,
      "arp": {
        "type": "ARPA",
        "timeout": 60
      },
      "tpid_value": "0x8100",
      "counters_last_cleared": "48 weeks 4 days 23:11:36",
      "queuing_strategy": "fifo",
      "input_statistics": {
        "packets": 2549097110546,
        "octets": 2468196758772796,
        "64_byte_pkts": 769025132,
        "over_64_byte_pkts": 227885030122,
        "over_127_byte_pkts": 89178965053,
        "over_255_byte_pkts": 15622467516,
        "over_511_byte_pkts": 17961134860,
        "over_1023_byte_pkts": 2197680487863,
        "multicasts": 139156443,
        "broadcasts": 11612512,
        "unicasts": 2548674316248,
        "runts": 0,
        "giants": 0,
        "throttles": 0,
        "crc": 0,
        "overrun": 0,
        "discarded": 0
      },
      "output_statistics": {
        "packets": 2072896808054,
        "octets": 1914711315598254,
        "64_byte_pkts": 1011582909,
        "over_64_byte_pkts": 260414799774,
        "over_127_byte_pkts": 91087851163,
        "over_255_byte_pkts": 17969133261,
        "over_511_byte_pkts": 11483328085,
        "over_1023_byte_pkts": 1690930112862,
        "multicasts": 137422034,
        "broadcasts": 24082491,
        "unicasts": 2072213667294,
        "throttles": 0,
        "discarded": 2097709,
        "collisions": 0,
        "wred_drops": 0
      },
      "rate_info": {
        "input": {
          "mbits_per_sec": 124,
          "packets_per_sec": 18546,
          "line_rate_percent": 0
        },
        "output": {
          "mbits_per_sec": 102,
          "packets_per_sec": 16437,
          "line_rate_percent": 0
        }
      },
      "time_since_last_status_change": "48 weeks 4 days 23:07:52"
    }
  ]
}
```
