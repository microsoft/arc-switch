# BGP All Summary Field Breakdown

## VRF-Level Fields

### vrf-name-out
- **Type**: String
- **Purpose**: VRF context identification
- **Usage**: Distinguish between different VRF routing tables
- **Example**: "default", "management", "customer-vrf"

### vrf-router-id
- **Type**: String (IP Address)
- **Purpose**: Router ID stability and uniqueness
- **Usage**: Identify the router in BGP sessions, track router identity changes
- **Validation**: Must be valid IPv4 address
- **Example**: "192.168.100.20"

### vrf-local-as
- **Type**: Integer
- **Purpose**: ASN classification (private/public)
- **Usage**: Identify AS ownership, detect private vs public AS ranges
- **Validation**: 1-4294967295 (16-bit: 1-65535, 32-bit: 1-4294967295)
- **Example**: 65238
- **Notes**: 
  - Private ASN ranges: 64512-65534 (16-bit), 4200000000-4294967294 (32-bit)
  - Public ASN: All others

## Address Family Fields

### af-id
- **Type**: Integer
- **Purpose**: Address family identifier
- **Values**: 1 (IPv4), 2 (IPv6)
- **Usage**: Separate IPv4 and IPv6 routing information

### safi
- **Type**: Integer
- **Purpose**: Subsequent Address Family Identifier
- **Values**: 1 (Unicast), 2 (Multicast), 128 (MPLS VPN)
- **Usage**: Determine routing type within address family

### af-name
- **Type**: String
- **Purpose**: Human-readable address family name
- **Example**: "IPv4-Unicast", "IPv6-Unicast"

## Routing Table Metrics

### tableversion
- **Type**: Integer
- **Purpose**: RIB changes, convergence health indicator
- **Usage**: Track routing table stability, detect convergence issues
- **Health Check**: Rapidly changing values indicate instability

### configuredpeers
- **Type**: Integer
- **Purpose**: Total BGP peers configured
- **Usage**: Inventory, capacity planning

### capablepeers
- **Type**: Integer
- **Purpose**: Peers in Established state
- **Usage**: Peer status and alerting
- **Health Check**: Should equal configuredpeers in healthy state
- **Alert**: If capablepeers < configuredpeers, investigate down peers

### totalnetworks
- **Type**: Integer
- **Purpose**: Total unique network prefixes in routing table
- **Usage**: Route table sizing, capacity monitoring

### totalpaths
- **Type**: Integer
- **Purpose**: Total paths to all networks (including ECMP)
- **Usage**: Route redundancy measurement
- **Analysis**: totalpaths/totalnetworks ratio indicates path diversity

## Memory Metrics

### memoryused
- **Type**: Integer (bytes)
- **Purpose**: Scale/capacity monitoring
- **Usage**: Track memory consumption for BGP RIB
- **Trending**: Monitor growth over time

### numberattrs / bytesattrs
- **Type**: Integer
- **Purpose**: Route attribute diversity
- **Usage**: Measure policy complexity
- **Analysis**: High values may indicate complex routing policies

### numberpaths / bytespaths
- **Type**: Integer
- **Purpose**: AS path diversity and topology
- **Usage**: Understand network topology complexity
- **Analysis**: Longer AS paths indicate more hops to destination

### numbercommunities / bytescommunities
- **Type**: Integer
- **Purpose**: Policy enforcement verification
- **Usage**: Track community tag usage for traffic engineering
- **Analysis**: High values indicate extensive policy tagging

### numberclusterlist / bytesclusterlist
- **Type**: Integer
- **Purpose**: Route reflector topology
- **Usage**: Verify route reflector design
- **Note**: Only relevant in route reflector deployments

## Route Dampening

### dampening
- **Type**: String
- **Values**: "yes", "no"
- **Purpose**: Flap suppression status
- **Usage**: Verify dampening policy is applied
- **Note**: Prevents unstable routes from causing network instability

## Per-Neighbor Fields

### neighborid
- **Type**: String (IP Address)
- **Purpose**: BGP neighbor identifier
- **Usage**: Identify specific BGP peer
- **Validation**: Must be valid IPv4 or IPv6 address

### neighborversion
- **Type**: Integer
- **Purpose**: BGP protocol version
- **Values**: Typically 4
- **Usage**: Verify protocol compatibility

### msgrecvd / msgsent
- **Type**: Integer
- **Purpose**: Message counter for session health
- **Usage**: Track BGP message exchange, detect communication issues
- **Analysis**: Steady growth indicates healthy session

### neighbortableversion
- **Type**: Integer
- **Purpose**: Peer's view of routing table version
- **Usage**: Verify synchronization between peers
- **Health Check**: Should match or be close to tableversion

### inq / outq
- **Type**: Integer
- **Purpose**: Queue depth monitoring
- **Usage**: Detect processing delays
- **Alert**: Non-zero values indicate backlog
- **Critical**: Sustained non-zero values indicate serious issues

### neighboras
- **Type**: Integer
- **Purpose**: Peer's AS number
- **Usage**: Identify eBGP vs iBGP sessions
- **Analysis**: Different AS = eBGP, same AS = iBGP

### time
- **Type**: String (ISO 8601 Duration)
- **Format**: PnYnMnDTnHnMnS or PnW (e.g., "P14W1D", "P37W6D")
- **Purpose**: Session uptime/stability
- **Usage**: Track session stability, identify flapping peers
- **Special Value**: "never" indicates session never established
- **Parsing**: Break down into weeks, days, hours for trend analytics

### state
- **Type**: String
- **Values**: "Idle", "Connect", "Active", "OpenSent", "OpenConfirm", "Established"
- **Purpose**: BGP session state
- **Usage**: Determine if neighbor is operational
- **Health Check**: "Established" is desired state
- **Alert**: States other than "Established" require investigation

### prefixreceived
- **Type**: Integer
- **Purpose**: Number of prefixes received from neighbor
- **Usage**: Measure route contribution, detect issues
- **Health Check**: 0 may indicate peer not advertising routes
- **Analysis**: Compare against expected values, detect excessive dependency on single peer

## Health Indicators and Anomalies

### Critical Issues
1. **Non-zero queue depths** (inq/outq): Processing delays
2. **capablepeers < configuredpeers**: Peers down
3. **neighbortableversion mismatch**: Synchronization issues
4. **prefixreceived = 0** on Established session: Peer not advertising
5. **state != "Established"**: Session problems
6. **Excessive dependency**: One peer contributing >50% of routes

### Warning Indicators
1. **Rapidly changing tableversion**: Network instability
2. **High memory usage**: Capacity concerns
3. **Short session uptime**: Frequent flapping
4. **Asymmetric msg counts**: Unidirectional issues

## KQL Query Considerations

- Use `data_type` field to filter BGP summary data
- Index on `vrf-name-out`, `af-name`, `state` for efficient queries
- Timestamp all entries for time-series analysis
- Store anomaly flags for dashboard visualization
- Track deltas in tableversion, msgrecvd, msgsent for change detection
