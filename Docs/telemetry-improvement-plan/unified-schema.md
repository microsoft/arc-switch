# Unified Telemetry Schema

> **Last updated**: 2026-07-14  
> **Branch**: `users/camilose/unified-telemetry-schema`  
> **Purpose**: Document the unified table schema that consolidates Cisco and SONiC telemetry into shared tables.

## Overview

Previously, Cisco and SONiC telemetry landed in vendor-prefixed tables (e.g.,
`CiscoBgpSummary_CL` vs `SonicBgpSummary_CL`). Both platforms share the same
OpenConfig models and produce identical or superset schemas, yet operators had to
write vendor-specific queries.

The unified schema eliminates vendor prefixes for shared data types. Cross-vendor
queries now work against a single table, with `device_type` serving as the vendor
discriminator when needed.

## Design Principles

1. **Vendor-neutral `data_type` constants** — Transformers emit canonical names
   like `bgp_summary`, `interface_counters` instead of `cisco_nexus_bgp_summary`.

2. **Unified table names** — Both `config.cisco.yaml` and `config.sonic.yaml`
   point to the same Kusto table for shared data types.

3. **`device_type` as vendor discriminator** — Already injected by `azure.Logger`
   into every record. No new column needed.

4. **Sparse columns are fine** — Cisco native transformers emit extra fields
   (e.g., `hold_interval`, `connection_drops`). SONiC rows simply lack those
   columns. Kusto handles sparse data natively.

5. **Incompatible row shapes stay separate** — `EnvPower` has different grain
   (Cisco: one row with PSU array; SONiC: one row per PSU). These remain in
   vendor-specific tables until row shape is normalized.

---

## Unified Table Mapping

### Shared Tables (cross-vendor queries supported)

| Table | data_type | Cisco Source | SONiC Source | Notes |
|---|---|---|---|---|
| `InterfaceCounter_CL` | `interface_counters` | interface_counters.go | same | Identical schema |
| `InterfaceStatus_CL` | `interface_status` | interface_status.go | same | Identical schema |
| `InterfaceEthernet_CL` | `interface_ethernet` | interface_ethernet.go | same | Identical schema |
| `BgpNeighbor_CL` | `bgp_summary` | native_bgp.go (superset) | bgp_summary.go | Cisco adds hold_interval, connection_drops, etc. |
| `BgpGlobal_CL` | `bgp_global` | bgp_global.go | same | Identical schema |
| `ArpEntry_CL` | `arp_entry` | native_arp.go (superset) | arp.go | Cisco adds phy_interface, flags |
| `LldpNeighbor_CL` | `lldp_neighbor` | native_lldp.go (superset) | lldp_neighbor.go | Cisco adds mgmt_addr, mgmt_addr_type, ttl |
| `MacTable_CL` | `mac_table` | native_mac.go (superset) | mac_address.go | Cisco adds entry_type, is_static |
| `SystemUptime_CL` | `system_uptime` | system.go | same | Identical schema |
| `SystemResources_CL` | `system_resources` | native_system.go (superset) | system.go | Cisco adds kernel, user CPU, etc. |
| `Inventory_CL` | `inventory` | inventory.go | same | Identical schema |
| `Transceiver_CL` | `transceiver` | native_transceiver.go (superset) | transceiver.go | Cisco adds connector_type, ethernet_pmd |
| `TransceiverDom_CL` | `transceiver_dom` | transceiver_dom | transceiver_channel.go | same |
| `EnvTemperature_CL` | `environment_temperature` | native_environment.go | sonic_platform.go (split) | Cisco: major/minor thresholds; SONiC: high/critical_high/low |

### Vendor-Specific Tables (no cross-vendor equivalent)

| Table | data_type | Vendor | Reason |
|---|---|---|---|
| `CiscoEnvPower_CL` | `environment_power` | Cisco | One row with PSU array — incompatible with SONiC grain |
| `CiscoVersion_CL` | `version` | Cisco | NX-OS-specific version model |
| `CiscoInterfaceErrors_CL` | `interface_error_counters` | Cisco | No SONiC YANG model |
| `CiscoRouteSummary_CL` | `route_summary` | Cisco | No SONiC YANG model |
| `SonicDeviceMetadata_CL` | `device_metadata` | SONiC | SONiC-specific metadata |
| `SonicFan_CL` | `fan` | SONiC | SONiC-specific fan model |
| `SonicEnvPower_CL` | `psu` | SONiC | Per-PSU rows — incompatible with Cisco grain |

---

## Breaking Changes

### Table Name Migration

If you have dashboards or alerts querying the old table names, update them:

| Old Cisco Table | Old SONiC Table | New Unified Table |
|---|---|---|
| `CiscoInterfaceCounters_CL` | `SonicInterfaceCounters_CL` | `InterfaceCounter_CL` |
| `CiscoInterfaceStatus_CL` | `SonicInterfaceStatus_CL` | `InterfaceStatus_CL` |
| `CiscoInterfaceEthernet_CL` | `SonicInterfaceEthernet_CL` | `InterfaceEthernet_CL` |
| `CiscoBgpSummary_CL` | `SonicBgpSummary_CL` | `BgpNeighbor_CL` |
| `CiscoBgpGlobal_CL` | `SonicBgpGlobal_CL` | `BgpGlobal_CL` |
| `CiscoArpEntry_CL` | `SonicArpEntry_CL` | `ArpEntry_CL` |
| `CiscoLldpNeighbor_CL` | `SonicLldpNeighbor_CL` | `LldpNeighbor_CL` |
| `CiscoMacTable_CL` | `SonicMacTable_CL` | `MacTable_CL` |
| `CiscoSystemUptime_CL` | `SonicSystemUptime_CL` | `SystemUptime_CL` |
| `CiscoSystemResources_CL` | `SonicSystemResources_CL` | `SystemResources_CL` |
| `CiscoInventory_CL` | `SonicInventory_CL` | `Inventory_CL` |
| `CiscoTransceiver_CL` | — | `Transceiver_CL` |
| `CiscoTransceiverDom_CL` | — | `TransceiverDom_CL` |
| `CiscoEnvTemperature_CL` | `SonicEnvTemperature_CL` | `EnvTemperature_CL` |

> **Note**: Historical data in Kusto remains under old table names. New data flows
> to the unified names after deployment.

### data_type Field Changes

The `data_type` field in every record has been renamed from vendor-prefixed to
vendor-neutral. Queries filtering on `data_type` must be updated:

| Old value | New value |
|---|---|
| `cisco_nexus_interface_counters` | `interface_counters` |
| `cisco_nexus_bgp_summary` | `bgp_summary` |
| `cisco_nexus_arp_entry` | `arp_entry` |
| `cisco_nexus_lldp_neighbor` | `lldp_neighbor` |
| `cisco_nexus_mac_table` | `mac_table` |
| ... (all `cisco_nexus_*` prefixes removed) | |

---

## Querying Unified Tables

### Cross-vendor query (all BGP neighbors)

```kql
BgpNeighbor_CL
| where TimeGenerated > ago(1h)
| summarize count() by device_type, neighbor_address
```

### Filter by vendor

```kql
InterfaceCounter_CL
| where device_type == "cisco_nexus"
| summarize avg(in_octets) by interface_name, bin(TimeGenerated, 5m)
```

### Handle sparse fields

```kql
BgpNeighbor_CL
| extend hold_interval = coalesce(hold_interval, 0)
| project device_type, neighbor_address, hold_interval
```

---

## Implementation Details

### sonic_platform Split

The SONiC `sonic-platform` gNMI path returns temperature, PSU, and fan data in a
single response. Since the collector routes all entries from one path to one table,
the transformer was split into three:

- `SonicTemperatureTransformer` → registers as `sonic-temperature` → `EnvTemperature_CL`
- `SonicPsuTransformer` → registers as `sonic-psu` → `SonicEnvPower_CL`
- `SonicFanTransformer` → registers as `sonic-fan` → `SonicFan_CL`

All three query the same gNMI path (`/sonic-platform:sonic-platform`) but extract
only their relevant data type from the response.

### Prefix System Removed

The `applyDataTypePrefix()` function in `collector.go` previously rewrote data_type
values at runtime (e.g., prepending `cisco_nexus_` to `bgp_summary`). This is no
longer needed since transformer constants are already vendor-neutral. Functions
removed: `applyDataTypePrefix()`, `DataTypePrefix()`, `DeviceTypeToPrefix()`.
