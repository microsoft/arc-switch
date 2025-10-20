# BGP All Summary Parser

A comprehensive Go-based parser for Cisco Nexus `show bgp all summary | json` command output. This parser extracts, validates, and enriches BGP summary data with health indicators and anomaly detection.

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
- `path_diversity_ratio`: Calculated ratio of paths to networks

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
- `health_status`: Calculated health status ("healthy", "warning", "critical")
- `health_issues`: List of detected issues

### Health Indicators and Anomaly Detection

**Critical Issues Detected:**
1. Non-zero queue depths (inq/outq) - Processing delays
2. capablepeers < configuredpeers - Peers down
3. neighbortableversion mismatch - Synchronization issues
4. prefixreceived = 0 on Established session - Peer not advertising
5. state != "Established" - Session problems
6. Excessive dependency - One peer contributing >50% of routes

**Warning Indicators:**
1. Output queue depth - Potential processing issues
2. No prefixes on established session
3. Table version mismatches

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

### As a Library

```go
import "github.com/microsoft/arc-switch/src/SwitchOutput/Cisco/Nexus/10/bgp_all_summary_parser"

func main() {
    jsonInput := `{"TABLE_vrf": {...}}`
    entries, err := bgp_all_summary_parser.parseBGPSummary(jsonInput)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, entry := range entries {
        fmt.Printf("VRF: %s, AS: %d (%s)\n", 
            entry.Message.VRFNameOut,
            entry.Message.VRFLocalAS,
            entry.Message.ASNType)
        
        // Check for anomalies
        if len(entry.Anomalies) > 0 {
            fmt.Println("Anomalies detected:")
            for _, anomaly := range entry.Anomalies {
                fmt.Printf("  - %s\n", anomaly)
            }
        }
    }
}
```

## Output Format

The parser outputs JSON Lines format (one JSON object per line), compatible with KQL queries and log analysis tools:

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
        "path_diversity_ratio": 2.0,
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
            "session_type": "eBGP",
            "health_status": "healthy",
            "health_issues": []
          }
        ]
      }
    ]
  },
  "anomalies": []
}
```

## Field Validation

The parser includes comprehensive field validation:
- Type checking for all numeric fields
- IP address format validation for router IDs and neighbor IDs
- AS number range validation (1-4294967295)
- BGP state validation
- Duration format validation

## KQL Query Examples

```kql
// Find all BGP entries with anomalies
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| where array_length(anomalies) > 0
| project timestamp, message.vrf_name_out, anomalies

// Monitor neighbor health
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand neighbor = message.address_families[0].neighbors
| where neighbor.health_status != "healthy"
| project timestamp, message.vrf_name_out, neighbor.neighbor_id, neighbor.health_status, neighbor.health_issues

// Track table version changes over time
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| project timestamp, message.vrf_name_out, af.af_name, af.table_version
| order by timestamp desc

// Identify peers with low uptime (potential flapping)
BGPSummaryLogs
| where data_type == "cisco_nexus_bgp_summary"
| mv-expand af = message.address_families
| mv-expand neighbor = af.neighbors
| where neighbor.time_parsed.total_seconds < 86400  // Less than 1 day
| project timestamp, neighbor.neighbor_id, neighbor.time, neighbor.state
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
- Neighbor health analysis tests
- Anomaly detection tests
- Invalid input handling tests
- Commands JSON integration tests

## Reference Documentation

See `fieldbreakdown.md` for detailed field-by-field breakdown, operational relevance, and expert notes on each BGP summary field.

## Command Integration

The parser integrates with the Cisco Nexus commands.json configuration file. Add this entry to your commands.json:

```json
{
    "name": "bgp-all-summary",
    "command": "show bgp all summary | json"
}
```

## License

This project is part of the microsoft/arc-switch repository and follows the same license terms.

## Contributing

Contributions are welcome! Please ensure:
1. All tests pass
2. New features include tests
3. Code follows Go best practices
4. Documentation is updated

## Support

For issues, questions, or contributions, please use the GitHub issue tracker for the arc-switch repository.
