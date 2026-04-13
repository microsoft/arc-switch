# Telemetry Data Parity — Overview

> **Last updated**: 2026-04-10  
> **Purpose**: Cross-vendor summary of gNMI telemetry data collection parity

This directory contains per-vendor data parity reports for the `gnmi-collector`.
Each report documents what data the collector exposes and, where a CLI baseline
exists, compares field-by-field coverage.

## Vendor Reports

| Vendor | Report | Comparison Type | Coverage |
|--------|--------|-----------------|----------|
| **Cisco NX-OS** | [cisco-nxos-parity.md](cisco-nxos-parity.md) | CLI vs gNMI | **~97%** |
| **SONiC (Dell Enterprise)** | [sonic-gnmi-parity.md](sonic-gnmi-parity.md) | gNMI only | **~76%** vs Cisco CLI baseline |
| **Dell OS10** | *Planned* | CLI vs gNMI | — |
| **Arista EOS** | *Planned* | gNMI only | — |

---

## Cross-Vendor Summary Matrix

| Data Category | Cisco CLI | Cisco gNMI | SONiC gNMI | Notes |
|---|:---:|:---:|:---:|---|
| Interface Counters | ✅ 12 | ✅ 16 | ✅ 16 | **Parity achieved** |
| Interface Status | ✅ 7 | ✅ 7 | ✅ 7 | **Parity achieved** |
| Interface Ethernet | ✅ 12 | ✅ 12 | ✅ 12 | **Parity achieved** |
| Interface Errors | ✅ 6 | ✅ 14 | ❌ 0 | SONiC: no YANG model |
| BGP Summary | ✅ 12 | ✅ 25 | ✅ 22 | gNMI richer than CLI on both |
| BGP Global | ✅ 5 | ✅ 5 | ✅ 5 | **Parity achieved** |
| System Resources | ✅ 17 | ✅ 15 | ⚠️ 8 | Cisco ~90%; SONiC ~60% |
| System Uptime | ✅ 11 | ✅ 14 | ✅ 14 | **Parity achieved** |
| Inventory | ✅ 5 | ✅ 6 | ✅ 6 | **Parity achieved** |
| LLDP Neighbors | ✅ 13 | ✅ 16 | ✅ 13 | **Parity achieved** |
| ARP Table | ✅ 13 | ✅ 8 | ✅ 5 | Core fields match |
| MAC Table | ✅ 7 | ✅ 12 | ⚠️ 0* | *L3-only switch, not a code issue |
| Env Temperature | ✅ 6 | ✅ 6 | ✅ 6 | **Parity achieved** |
| Env Power Supply | ✅ 20 | ⚠️ 14 | ⚠️ 10 | CLI aggregations missing |
| Env Fan | ✅ 3 | — | ✅ 7 | SONiC-only via sonic-platform |
| Transceiver | ✅ 16 | ✅ 14 | ❌ 0 | SONiC: disabled (needs per-intf keys) |
| Transceiver DOM | ✅ 5 | ✅ 5 | ❌ 0 | SONiC: disabled |
| Route Summary | ✅ 3 | ✅ 4 | ❌ 0 | SONiC: no YANG model |
| Version | ✅ 12 | ✅ 12 | ⚠️ 5 | SONiC: partial via metadata |
| QoS / Class Map | ✅ 3 | ❌ 0 | ❌ 0 | No gNMI model on any platform |
| Device Metadata | — | — | ✅ 5 | SONiC-only |

**Legend**: ✅ Full parity | ⚠️ Partial | ❌ Missing

---

## Common Limitations (All Vendors)

| Item | Reason |
|------|--------|
| **QoS / Class Map** | OpenConfig QoS models exist but no vendor implements them via gNMI |
| **Power aggregations** (total_grid_*, total_power_*) | CLI-only computed values, not raw device data |
| **vmalloc** | Linux kernel metric, not exposed via any YANG model |
| **Table naming** — `GnmiTest*` prefix | Rename to production names when deploying to production |

---

## Architecture Notes

- **Single binary, multi-vendor**: The same `gnmi-collector` handles Cisco native YANG
  and SONiC/Arista OpenConfig via a transformer registry with self-registration.
- **Both poll and subscribe modes** are validated with 100% data parity between modes.
- **Zero config changes to switches** — read-only gNMI subscriptions only.
- **Arc-integrated** — runs as init.d (NX-OS) or systemd (SONiC/Linux) service.
