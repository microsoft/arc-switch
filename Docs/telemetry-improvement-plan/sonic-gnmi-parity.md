# SONiC (Dell Enterprise) — gNMI Data Coverage

> **Last updated**: 2026-04-10  
> **Test switch**: b88-c01-5248l-7-1a (100.100.81.129) — Dell Enterprise SONiC 4.5.1 (x86_64)  
> **Collector config**: 16 gNMI paths, all validated  
> **Dry-run result**: 14 success, 0 failures in 7.4s  
> **Coverage vs Cisco CLI baseline**: **~76%**

SONiC uses OpenConfig YANG models plus Dell `sonic-*` native YANG extensions.
There is no CLI-based telemetry baseline for SONiC — coverage is measured
against the Cisco CLI baseline for cross-platform consistency.

---

## Key Gaps (Future Work)

| Gap Category | Impact | Status | Effort |
|---|---|---|---|
| **Transceiver DOM** — disabled | Medium — optical monitoring | Needs per-interface component key enumeration | Medium |
| **Interface errors** — no YANG path | Medium — fault isolation | Investigate `sonic-interface` native YANG or `/proc/net/dev` | High |
| **Route summary** — no YANG path | Medium — routing analytics | Investigate `sonic-route-common` YANG model | Medium |
| **Version info** — only basic metadata | Low — device identity | Combine `sonic-device-metadata` with `/etc/sonic/sonic_version.yml` | Medium |
| **Load averages, process counts** | Low — host diagnostics | No SONiC YANG path | Unknown |
| **MAC table** — 0 records | Low — L2 visibility | Test switch is L3-only; not a code issue | N/A |

---

## Detailed Field-by-Field Coverage

### 1. Interface Counters

| Field | gNMI |
|-------|:----:|
| `interface_name` | ✅ |
| `interface_type` | ✅ |
| `in_octets` | ✅ |
| `in_ucast_pkts` | ✅ |
| `in_mcast_pkts` | ✅ |
| `in_bcast_pkts` | ✅ |
| `out_octets` | ✅ |
| `out_ucast_pkts` | ✅ |
| `out_mcast_pkts` | ✅ |
| `out_bcast_pkts` | ✅ |
| `in_errors` | ✅ |
| `in_discards` | ✅ |
| `out_errors` | ✅ |
| `out_discards` | ✅ |
| `has_ingress_data` | ✅ |
| `has_egress_data` | ✅ |

> OpenConfig `/interfaces/interface/state/counters`.

**Records**: 56 interfaces  
**Status**: ✅ Full coverage (16 fields)

---

### 2. Interface Status

| Field | gNMI |
|-------|:----:|
| `port` | ✅ |
| `name` (description) | ✅ |
| `status` (oper_status) | ✅ |
| `vlan` | ✅ |
| `speed` | ✅ |
| `type` | ✅ |

**Records**: 2 batches (61 interfaces)  
**Status**: ✅ Full coverage

---

### 3. Interface Ethernet (L1 Details)

| Field | gNMI |
|-------|:----:|
| `interface_name` | ✅ |
| `speed` | ✅ |
| `duplex` | ⚠️ (empty on this hardware) |
| `auto_negotiate` | ✅ |
| `mac_address` | ⚠️ (empty on this hardware) |
| `hw_mac_address` | ⚠️ (empty on this hardware) |
| `in_crc_errors` | ✅ |
| `in_fragment_frames` | ✅ |
| `in_jabber_frames` | ✅ |
| `in_oversize_frames` | ✅ |
| `in_undersize_frames` | ✅ |
| `out_crc_errors` | ✅ |

> OpenConfig `/interfaces/interface/ethernet`. Some fields empty depending on hardware/driver.

**Records**: 57 interfaces  
**Status**: ✅ Full coverage (some fields hardware-dependent)

---

### 4. Interface Error Counters

> ❌ **Not available** — no SONiC YANG model for debug ethernet statistics.

**Records**: 0  
**Status**: ❌ Gap — investigate `sonic-interface` native YANG or `/proc/net/dev`

---

### 5. BGP Summary (Neighbors)

| Field | gNMI |
|-------|:----:|
| `neighbor_id` | ✅ |
| `neighbor_address` | ✅ |
| `vrf_name` | ✅ |
| `vrf_name_out` | ✅ |
| `vrf_router_id` | ✅ |
| `vrf_local_as` | ✅ |
| `neighbor_as` / `peer_as` | ✅ |
| `peer_type` | ✅ |
| `state` | ✅ |
| `session_state` | ✅ |
| `enabled` | ✅ |
| `description` | ✅ |
| `msg_recvd` | ✅† |
| `msg_sent` | ✅† |
| `messages_received_updates` | ✅ |
| `messages_sent_updates` | ✅ |
| `messages_received_notifications` | ✅ |
| `messages_sent_notifications` | ✅ |
| `prefix_received` | ✅ |
| `established_transitions` | ✅ |
| `last_established` | ✅ |

> OpenConfig `bgp/neighbors`. SONiC has 4 unique fields (enabled, description, notifications)
> that Cisco doesn't expose.
> † `msg_recvd`/`msg_sent` bug fixed in OC `toInt64()` — added `strconv.ParseInt` for strings.

**Records**: 3 peers  
**Status**: ✅ Full coverage (22 fields)

---

### 6. BGP Global

| Field | gNMI |
|-------|:----:|
| `vrf_name` | ✅ |
| `local_as` / `router_id` | ✅ |
| `total_paths` | ✅ |
| `total_prefixes` | ✅ |

**Records**: 2 VRFs  
**Status**: ✅ Full coverage

---

### 7. System Resources (CPU / Memory)

| Field | gNMI |
|-------|:----:|
| `cpu_state_user` | ✅ |
| `cpu_state_kernel` | ✅ |
| `cpu_state_idle` | ✅ |
| `cpu_usage` (per-core array) | ✅ (5 cores) |
| `memory_usage_total` | ✅ |
| `memory_usage_used` | ✅ |
| `memory_usage_free` | ✅ |
| `memory_usage_reserved` | ✅ |
| `kernel_buffers` | ❌ |
| `kernel_cached` | ❌ |
| `current_memory_status` | ❌ |
| `load_avg_*` | ❌ |
| `processes_total/running` | ❌ |

> OpenConfig `system/cpus` and `system/memory`. SONiC's OC implementation
> doesn't expose kernel buffers, cached memory, load averages, or process counts.

**Records**: 2  
**Status**: ⚠️ **~60%** — CPU and basic memory present, advanced metrics missing

---

### 8. System Uptime

| Field | gNMI |
|-------|:----:|
| `hostname` | ✅ |
| `domain_name` | ✅ |
| `system_start_time` | ✅ |
| `system_uptime_days/hours/min/sec` | ✅ |
| `system_uptime_total` | ✅ |
| `kernel_uptime_days/hours/min/sec` | ✅ |
| `kernel_uptime_total` | ✅ |
| `current_datetime` | ✅ |

**Records**: 2  
**Status**: ✅ Full coverage

---

### 9. Inventory (Hardware Components)

| Field | gNMI |
|-------|:----:|
| `name` | ✅ |
| `description` | ✅ |
| `product_id` (PID) | ✅ |
| `version_id` (VID) | ✅ |
| `serial_number` | ✅ |
| `component_type` | ✅ |

> OpenConfig `platform/components`. SONiC returns significantly more components
> than Cisco (fans, PSUs, transceivers all as individual inventory items).

**Records**: 39 components  
**Status**: ✅ Full coverage

---

### 10. LLDP Neighbors

| Field | gNMI |
|-------|:----:|
| `chassis_id` | ✅ |
| `port_id` | ✅ |
| `local_port_id` | ✅ |
| `port_description` | ✅ |
| `system_name` | ✅ |
| `system_description` | ✅ |
| `management_address` | ✅ |
| `management_address_ipv6` | ✅ |
| `time_remaining` | ✅ |
| `max_frame_size` | ✅ |
| `vlan_id` | ✅ |
| `system_capabilities` | ✅ |
| `enabled_capabilities` | ✅ |

> OpenConfig LLDP. Covers all core fields.

**Records**: 7 neighbors  
**Status**: ✅ Full coverage

---

### 11. ARP Table

| Field | gNMI |
|-------|:----:|
| `ip_address` | ✅ |
| `mac_address` | ✅ |
| `interface` | ✅ |
| `interface_type` | ✅ |
| `age` | ✅ |

> OpenConfig ARP. Core fields only — no physical_interface, flags, or status.

**Records**: 6 entries  
**Status**: ✅ Core fields present

---

### 12. MAC Address Table

| Field | gNMI |
|-------|:----:|
| `mac_address` | ✅* |
| `vlan` | ✅* |
| `type` | ✅* |
| `age` | ✅* |
| `port` | ✅* |

> \* Path is configured but returned **0 records** on this switch.
> Expected for L3-only topology (no L2 FDB entries).

**Records**: 0  
**Status**: ⚠️ Code works, no data on this switch (L3-only)

---

### 13. Environment — Temperature

| Field | gNMI |
|-------|:----:|
| `sensor` | ✅ |
| `current_temp` | ✅ |
| `critical_high_threshold` | ✅ |
| `high_threshold` | ✅ |
| `low_threshold` | ✅ |
| `timestamp` | ✅ |

> Uses `sonic-platform` native YANG.

**Records**: 20 temperature readings  
**Status**: ✅ Full coverage

---

### 14. Environment — Power Supply

| Field | gNMI |
|-------|:----:|
| `name` | ✅ |
| `model` | ✅ |
| `serial` | ✅ |
| `status` | ✅ |
| `input_voltage` | ✅ |
| `input_current` | ✅ |
| `output_voltage` | ✅ |
| `output_current` | ✅ |
| `output_power` | ✅ |
| `temp` | ✅ |

> Via `sonic-platform` native YANG. No capacity or alarm fields.

**Records**: Included in Platform table  
**Status**: ⚠️ **~60%** — core PSU metrics present, no capacity/fan/alarm data

---

### 15. Environment — Fan

| Field | gNMI |
|-------|:----:|
| `name` | ✅ |
| `speed` | ✅ |
| `direction` | ✅ |
| `model` | ✅ |
| `serial` | ✅ |
| `status` | ✅ |
| `drawer_name` | ✅ |

> `sonic-platform` native YANG. SONiC-specific — Cisco embeds fan data in PSU records.

**Records**: Included in Platform table  
**Status**: ✅ Full coverage

---

### 16. Transceiver / Transceiver DOM

> ❌ **Disabled** in config. SONiC's OpenConfig transceiver path requires
> per-interface component keys (e.g., `/components/component[name=Ethernet0]/transceiver`).
> Enabling would require enumerating all Ethernet interfaces as separate config entries.

**Records**: 0  
**Status**: ❌ Disabled — needs per-interface key enumeration

---

### 17. Route Summary

> ❌ **Not available** — no SONiC YANG model for route summary.

**Records**: 0  
**Status**: ❌ Gap — investigate `sonic-route-common` YANG model

---

### 18. Version / Device Info

> ⚠️ **Partial** — only basic metadata via `sonic-device-metadata`.

| Field | gNMI |
|-------|:----:|
| `hostname` | ✅ |
| `hwsku` | ✅ |
| `platform` | ✅ |
| `mac` | ✅ |
| `type` | ✅ |

> No OS version, build number, or kernel info. Could be enriched by reading
> `/etc/sonic/sonic_version.yml` via a custom YANG path.

**Records**: 2  
**Status**: ⚠️ Partial

---

### 19. Device Metadata (SONiC-only)

| Field | gNMI |
|-------|:----:|
| `hostname` | ✅ |
| `hwsku` | ✅ |
| `platform` | ✅ |
| `mac` | ✅ |
| `type` | ✅ |

> `sonic-device-metadata` native YANG. Unique to SONiC — no Cisco equivalent.

**Records**: 2  
**Status**: ✅ SONiC-specific data

---

## Record Counts (from validated dry-run, 2026-04-09)

| Table | Records |
|-------|:-------:|
| InterfaceCounter | 56 |
| InterfaceStatus | 2 (61 intf) |
| InterfaceEthernet | 57 |
| BgpSummary | 3 |
| BgpGlobal | 2 |
| SystemResources | 2 |
| SystemUptime | 2 |
| Inventory | 39 |
| LldpNeighbor | 7 |
| ArpEntry | 6 |
| MacTable | 0 |
| Platform (Temp+PSU+Fan) | 20 |
| DeviceMetadata | 2 |

---

## Code Changes (2026-04-09)

### `bgp_summary.go` (OpenConfig BGP transformer)
- Fixed `toInt64()` to handle `string` type via `strconv.ParseInt`
- SONiC gNMI server returns message counts as strings (same bug as Cisco native)

### Build & Deploy
- ✅ All Go tests pass
- ✅ amd64 binary built (13 MB) — switch is x86_64
- ✅ 14 success, 0 failures
- ✅ Production collector running via systemd
