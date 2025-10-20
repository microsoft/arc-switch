# BGP All Summary Parser

A Go-based parser for Cisco Nexus `show bgp all summary | json` command output. This parser extracts and converts all BGP summary data to a standardized JSON format suitable for log analysis and network dashboards.

## Features

### Field Extraction

**VRF-Level Fields:**
- `vrf-name-out`: VRF context identification
- `vrf-router-id`: Router ID stability and uniqueness
- `vrf-local-as`: ASN classification (private/public)
- `asn_type`: Automatically classified as "private" or "public"

**Address Family Fields:**
- `af-id`, `safi`, `af-name`: Address family separation and routing type
- `tableversion`: RIB changes, convergence health indicator
- `configuredpeers`, `capablepeers`: Peer status and alerting
- `totalnetworks`, `totalpaths`: Route table sizing and redundancy

**Memory Metrics:**
- `memoryused`: Scale/capacity monitoring
- `numberattrs/bytesattrs`: Route attribute diversity
- `numberpaths/bytespaths`: AS path diversity and topology
- `numbercommunities/bytescommunities`: Policy enforcement verification
- `numberclusterlist/bytesclusterlist`: Route reflector topology

**Route Dampening:**
- `dampening`: Flap suppression status ("yes"/"no")

**Per-Neighbor Fields:**
- `neighborid`: BGP neighbor identifier
- `neighborversion`: BGP protocol version
- `msgrecvd`, `msgsent`: Message counters for session health
- `neighbortableversion`: Peer's routing table version
- `inq`, `outq`: Queue depth monitoring
- `neighboras`: Peer's AS number
- `time`: Session uptime (ISO 8601 duration)
- `time_parsed`: Parsed duration breakdown (weeks, days, hours, etc.)
- `state`: BGP session state
- `prefixreceived`: Prefixes received from neighbor
- `session_type`: Automatically determined ("eBGP" or "iBGP")

### ISO 8601 Duration Parsing

The parser includes a robust ISO 8601 duration parser that handles Cisco's format:
- Supports formats like `P14W1D`, `P37W6D`, `P10W2DT5H30M`
- Breaks down duration into weeks, days, hours, minutes, seconds
- Calculates total duration in seconds for trend analytics
- Handles special value "never" for sessions that never established

## Usage

### Command Line

```bash
# Parse from input file
./bgp_all_summary_parser -input show-bgp-all-summary.txt

# Parse from input file and save to output file
./bgp_all_summary_parser -input show-bgp-all-summary.txt -output output.json

# Get data directly from switch using commands.json
./bgp_all_summary_parser -commands ../commands.json -output output.json
```

## Output Format

The parser outputs JSON Lines format (one JSON object per line), compatible with KQL queries and log analysis tools. The focus is on data extraction only - health monitoring and anomaly detection should be performed using KQL for better resource efficiency on network devices.

```json
{
  "data_type": "cisco_nexus_bgp_summary",
  "timestamp": "2025-10-20T23:00:00Z",
  "date": "2025-10-20",
  "message": {
    "vrf_name_out": "default",
    "vrf_router_id": "192.168.100.20",
    "vrf_local_as": 65238,
    "asn_type": "private",
    "address_families": [
      {
        "af_id": 1,
        "safi": 1,
        "af_name": "IPv4-Unicast",
        "table_version": 12345,
        "configured_peers": 4,
        "capable_peers": 4,
        "total_networks": 150,
        "total_paths": 300,
        "memory_used": 40960,
        "dampening": "yes",
        "neighbors": [
          {
            "neighbor_id": "192.168.100.1",
            "neighbor_version": 4,
            "msg_recvd": 15000,
            "msg_sent": 14500,
            "neighbor_table_version": 12345,
            "inq": 0,
            "outq": 0,
            "neighbor_as": 64846,
            "time": "P14W1D",
            "time_parsed": {
              "weeks": 14,
              "days": 1,
              "total_seconds": 8553600
            },
            "state": "Established",
            "prefix_received": 75,
            "session_type": "eBGP"
          }
        ]
      }
    ]
  }
}
```

## Health Monitoring with KQL

The parser focuses on data extraction and conversion. Health indicators and anomaly detection should be performed using KQL queries for better resource efficiency on network devices.

### KQL Query Examples for Health Monitoring

```kql
// Detect peers with non-zero queue depths (critical issue)
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| mv-expand neighbor = af.neighbors
| where neighbor.inq > 0 or neighbor.outq > 0
| project timestamp, message.vrf_name_out, af.af_name, neighbor.neighbor_id, 
          neighbor.state, neighbor.inq, neighbor.outq
| extend severity = iff(neighbor.inq > 0, "critical", "warning")

// Detect peer count mismatches (capable vs configured)
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| where af.capable_peers < af.configured_peers
| project timestamp, message.vrf_name_out, af.af_name, 
          af.configured_peers, af.capable_peers
| extend peers_down = af.configured_peers - af.capable_peers

// Detect sessions not in Established state
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| mv-expand neighbor = af.neighbors
| where neighbor.state != "Established"
| project timestamp, message.vrf_name_out, af.af_name, 
          neighbor.neighbor_id, neighbor.state, neighbor.outq

// Detect established sessions with no prefixes received
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| mv-expand neighbor = af.neighbors
| where neighbor.state == "Established" and neighbor.prefix_received == 0
| project timestamp, message.vrf_name_out, af.af_name, neighbor.neighbor_id

// Detect table version mismatches between local and neighbors
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| mv-expand neighbor = af.neighbors
| where neighbor.state == "Established" 
    and neighbor.neighbor_table_version > 0 
    and neighbor.neighbor_table_version != af.table_version
| project timestamp, message.vrf_name_out, af.af_name, neighbor.neighbor_id,
          local_version = af.table_version, neighbor_version = neighbor.neighbor_table_version

// Calculate path diversity ratio and detect low diversity
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| where af.total_networks > 0
| extend path_diversity_ratio = todouble(af.total_paths) / todouble(af.total_networks)
| where path_diversity_ratio < 1.5
| project timestamp, message.vrf_name_out, af.af_name, 
          af.total_networks, af.total_paths, path_diversity_ratio

// Detect excessive dependency on single peer (>50% of routes)
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| mv-expand neighbor = af.neighbors
| where af.total_networks > 0 and neighbor.state == "Established"
| extend dependency_percent = (todouble(neighbor.prefix_received) / todouble(af.total_networks)) * 100
| where dependency_percent > 50
| project timestamp, message.vrf_name_out, af.af_name, 
          neighbor.neighbor_id, neighbor.prefix_received, af.total_networks, dependency_percent

// Track session stability (sessions up for less than 1 day)
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| mv-expand neighbor = af.neighbors
| where neighbor.time_parsed.total_seconds < 86400 and neighbor.state == "Established"
| project timestamp, message.vrf_name_out, af.af_name, neighbor.neighbor_id, 
          neighbor.time, uptime_hours = neighbor.time_parsed.total_seconds / 3600

// Monitor BGP convergence by tracking table version changes
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| project timestamp, message.vrf_name_out, af.af_name, af.table_version
| order by timestamp desc
| serialize
| extend version_change = af.table_version - prev(af.table_version, 1)
| where version_change != 0
| project timestamp, message.vrf_name_out, af.af_name, 
          current_version = af.table_version, version_change
```

## Building

```bash
go build -o bgp_all_summary_parser
```

## Testing

```bash
go test -v
```

The test suite includes:
- Full BGP summary parsing validation
- ISO 8601 duration parsing tests
- ASN classification tests
- Session type detection tests
- Invalid input handling tests
- Commands JSON integration tests

## Reference Documentation

See `fieldbreakdown.md` for detailed field-by-field breakdown and operational relevance of each BGP summary field.

## Command Integration

The parser integrates with the Cisco Nexus commands.json configuration file. The following entry is included:

```json
{
    "name": "bgp-all-summary",
    "command": "show bgp all summary | json"
}
```

## License

This project is part of the microsoft/arc-switch repository and follows the same license terms.
