# Telemetry Improvement Plan: gNMI/YANG for Cisco Nexus Switches

## Overview

This document proposes replacing the current cron-based CLI scraping telemetry
pipeline with a **gRPC/gNMI + YANG model** approach for Cisco Nexus switches.
The new architecture eliminates fragile text parsing, enables real-time streaming
telemetry, and provides a more scalable and version-resilient data collection
framework.

## Current Architecture

### How Telemetry Works Today

The existing pipeline collects switch data through a cron job that runs every
5 minutes:

```text
cron (*/5) → vsh -c "show ..." → cisco-parser (text→JSON) → azure-logger (HTTP POST) → Log Analytics
```

#### Components

| Component | Location | Purpose |
|-----------|----------|---------|
| **cisco-parser** | `src/SwitchOutput/Cisco/Nexus/10/cisco-parser/` | Unified Go binary with 16 sub-parsers that convert CLI text output to standardized JSON |
| **Individual parsers** | `src/SwitchOutput/Cisco/Nexus/10/*_parser/` | Go packages for each `show` command (interface counters, MAC address, BGP, etc.) |
| **syslogwriter** | `src/SyslogTools/syslogwriter/` | Go library for writing JSON entries to Linux syslog |
| **syslog-client** | `src/SyslogTools/syslog-client/` | CLI tool for sending JSON to syslog |
| **Setup script** | `Docs/arcnet_onboarding_instructions/Arcnet_Cisco_Arc_Setup` | Installs all components on the switch |
| **Collector script** | Deployed at `/opt/cisco-parser-collector.sh` | Cron-triggered bash script that orchestrates data collection |
| **Azure Logger** | Deployed at `/opt/cisco-azure-logger-v2.sh` | Sends JSON to Azure Log Analytics HTTP Data Collector API |

#### Data Collection Flow

1. **Cron** triggers `/opt/cisco-parser-collector.sh` every 5 minutes
2. For each telemetry type, the collector:
   - Runs `vsh -c "<show command>"` to get CLI text output
   - Pipes output to `cisco-parser -p <type> -i <file> -o <output.json>`
   - The parser regex-matches CLI text lines into Go structs, then serializes
     to JSON
   - Calls `cisco-azure-logger-v2.sh send <TableName> <output.json>`
3. The logger:
   - Adds `hostname` and `device_type` metadata via Python
   - Generates an HMAC-SHA256 signature with the workspace key
   - POSTs JSON to
     `https://<workspace-id>.ods.opinsights.azure.com/api/logs`
4. Data appears in Log Analytics as custom tables (e.g.,
   `CiscoInterfaceCounter_CL`)

#### Commands Currently Collected

| Show Command | Parser Type | Azure Table |
|-------------|-------------|-------------|
| `show class-map` | class-map | CiscoClassMap |
| `show interface counter` | interface-counters | CiscoInterfaceCounter |
| `show inventory all` | inventory | CiscoInventory |
| `show ip arp` | ip-arp | CiscoIpArp |
| `show lldp neighbor detail` | lldp-neighbor | CiscoLldpNeighbor |
| `show interface transceiver` | transceiver | CiscoTransceiver |
| `show environment temperature` | environment-temperature | CiscoEnvTemp |
| `show interface counters errors` | interface-error-counters | CiscoInterfaceErrors |
| `show environment power detail` | environment-power | CiscoEnvPower |
| `show system resources` | system-resources | CiscoSystemResources |
| `show system uptime` | system-uptime | CiscoSystemUptime |
| `show bgp all summary` | bgp-all-summary | CiscoBgpSummary |
| `show interface status` | interface-status | CiscoInterfaceStatus |
| `show version` | version | CiscoVersion |

#### Standardized JSON Output Format

All parsers produce a standardized structure:

```json
{
  "data_type": "<vendor_device_type>",
  "timestamp": "<ISO 8601>",
  "date": "<YYYY-MM-DD>",
  "message": {
    "field1": "value1",
    "field2": 12345
  }
}
```

### Limitations of the Current Approach

- **Polling-only**: Data freshness is limited to the 5-minute cron interval
- **CLI text parsing is fragile**: Regex-based parsers break when NX-OS output
  format changes across versions
- **Inefficient**: Each collection cycle runs ~14 sequential `vsh` commands
  with `sleep 2` between them (~30s+ per cycle)
- **No streaming**: Cannot react to real-time events (link flaps, BGP state
  changes, etc.)
- **Scalability**: Adding new telemetry requires writing a new text parser each
  time

---

## Proposed Architecture: gRPC/gNMI + YANG Models

### What is YANG?

YANG (RFC 7950) is a data modeling language for network configuration and state
data. Instead of parsing free-form CLI text, you query a switch using
well-defined, versioned data models that return structured data (typically JSON
or Protobuf).

Two families of YANG models are relevant to NX-OS:

1. **OpenConfig models** — Vendor-neutral, community-maintained (e.g.,
   `openconfig-interfaces`, `openconfig-bgp`)
2. **Cisco NX-OS native models** — Cisco-specific, richer detail (e.g.,
   `Cisco-NX-OS-device:System`)

### What is gNMI?

gNMI (gRPC Network Management Interface) is a gRPC-based protocol defined by
OpenConfig for:

- **Get** — One-shot retrieval of YANG model data (replaces `show` commands)
- **Set** — Configuration changes (replaces `configure terminal`)
- **Subscribe** — Real-time streaming of state changes (no CLI equivalent)
- **Capabilities** — Discover which YANG models the device supports

### How gNMI Replaces the Current Pipeline

```text
Current:  cron → vsh "show interface counter" → regex parser → JSON → Azure
Proposed: gNMI Subscribe/Get → structured YANG data (JSON/Protobuf) → transform → Azure
```

Key advantages:

- **No text parsing**: Data arrives already structured per the YANG model schema
- **Streaming support**: `Subscribe` mode pushes data on-change or at
  configured intervals
- **Version-resilient**: YANG models are versioned; no breakage on NX-OS
  upgrades
- **Efficient**: Single gRPC connection, binary Protobuf encoding, multiplexed
  streams

### NX-OS Prerequisites

gNMI requires enabling the gRPC feature on the switch:

```text
! Enable gRPC/gNMI server on the switch
configure terminal
feature grpc

! (Optional) Configure gRPC settings
grpc port 50051
grpc certificate <cert-name>

! Verify
show feature | grep grpc
show grpc gnmi service statistics
```

> [!NOTE]
> NX-OS 9.3(x)+ has good gNMI support. NX-OS 10.x has comprehensive support
> for both OpenConfig and native YANG models. NX-OS supports `JSON` and
> `PROTO` encodings — `JSON_IETF` is not supported.

### YANG Path Mapping

The following table maps each current `show` command to its approximate
YANG/gNMI paths. Exact paths depend on NX-OS version — use `gNMI Capabilities`
to discover available models on a specific device.

| Current Show Command | OpenConfig YANG Path | NX-OS Native YANG Path |
|---------------------|---------------------|----------------------|
| `show interface counter` | `/openconfig-interfaces:interfaces/interface/state/counters` | `/System/intf-items/phys-items/PhysIf-list/dbgIfIn-items` |
| `show interface status` | `/openconfig-interfaces:interfaces/interface/state` | `/System/intf-items/phys-items/PhysIf-list/operSt` |
| `show interface counters errors` | `/openconfig-interfaces:interfaces/interface/state/counters` | `/System/intf-items/phys-items/PhysIf-list/dbgIfIn-items` |
| `show ip arp` | `/openconfig-if-ip:ipv4/neighbors/neighbor` | `/System/arp-items/inst-items/dom-items/Dom-list/db-items/Db-list/adj-items/AdjEp-list` |
| `show bgp all summary` | `/openconfig-network-instance:network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state` | `/System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list` |
| `show lldp neighbor detail` | `/openconfig-lldp:lldp/interfaces/interface/neighbors` | `/System/lldp-items/inst-items/if-items/If-list` |
| `show environment temperature` | `/openconfig-platform:components/component/state/temperature` | `/System/ch-items/supslot-items/SupCSlot-list/sensor-items` |
| `show environment power` | `/openconfig-platform:components/component/power-supply` | `/System/ch-items/psuslot-items/PsuSlot-list` |
| `show system resources` | `/openconfig-system:system/cpus`, `/openconfig-system:system/memory` | `/System/procsys-items/syscpusummary-items`, `/System/procsys-items/sysmem-items` |
| `show system uptime` | `/openconfig-system:system/state/boot-time` | `/System/showversion-items/uptime` |
| `show inventory all` | `/openconfig-platform:components/component` | `/System/ch-items` |
| `show mac address-table` | `/openconfig-network-instance:network-instances/.../mac-table` | `/System/mac-items/table-items/Table-list` |
| `show interface transceiver` | `/openconfig-platform:components/component/transceiver` | `/System/intf-items/phys-items/PhysIf-list/phys-items` |
| `show version` | `/openconfig-system:system/state` | `/System/showversion-items` |

---

## Proposed System Design

### High-Level Architecture

```text
┌─────────────────────────────────────────────────┐
│            Cisco Nexus Switch (NX-OS)            │
│                                                   │
│   gRPC/gNMI Server (port 50051)                  │
│   ├── OpenConfig YANG models                     │
│   └── Cisco native YANG models                   │
└──────────────┬──────────────────────────────────┘
               │ gRPC/TLS
               ▼
┌─────────────────────────────────────────────────┐
│         gNMI Telemetry Client (Go binary)        │
│  Runs on the switch itself (or a collector host) │
│                                                   │
│  ┌─────────────┐  ┌──────────────┐               │
│  │ gNMI Client │→ │ Transformer  │               │
│  │ (Subscribe/ │  │ (YANG JSON → │               │
│  │  Get)       │  │  arcnet JSON)│               │
│  └─────────────┘  └──────┬───────┘               │
│                          │                        │
│  ┌───────────────────────▼───────────────────┐   │
│  │  Azure Logger (HTTP Data Collector API)   │   │
│  │  - HMAC-SHA256 signing                    │   │
│  │  - POST to Log Analytics                  │   │
│  └───────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

### Project Structure

```text
src/TelemetryClient/
├── cmd/gnmi-collector/main.go     # Entry point, CLI flags, poll loop
├── internal/
│   ├── config/config.go           # YAML config loader with env var resolution
│   ├── gnmi/client.go             # gNMI client wrapper (TLS, auth, Get)
│   ├── azure/logger.go            # Azure Log Analytics HTTP Data Collector
│   ├── transform/                 # 11 transformers (YANG JSON → arcnet JSON)
│   └── collector/collector.go     # Orchestrator (path→transform→send cycle)
├── testdata/                      # Real switch JSON fixtures for unit tests
├── config.example.yaml            # Reference configuration
├── Makefile                       # Build targets (make build, make test)
└── go.mod
```

### Configuration

The collector is driven by a YAML config file that declares:

- gNMI connection settings (target, TLS, credentials)
- Azure Log Analytics credentials
- Subscription definitions (YANG paths, mode, intervals, target table)

```yaml
gnmi:
  target: "localhost:50051"
  tls:
    skip_verify: true    # Self-signed cert on switch
  credentials:
    username: "${GNMI_USER}"
    password: "${GNMI_PASS}"

azure:
  workspace_id: "${WORKSPACE_ID}"
  primary_key: "${PRIMARY_KEY}"
  secondary_key: "${SECONDARY_KEY}"

collection:
  interval: "300s"       # Match current 5-minute cron cycle

paths:
  - name: "interface-counters"
    yang_path: "/openconfig-interfaces:interfaces/interface/state/counters"
    table: "CiscoInterfaceCounter"
    enabled: true

  - name: "bgp-neighbors"
    yang_path: "/openconfig-network-instance:network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state"
    table: "CiscoBgpSummary"
    enabled: true

  # ... 11 more paths (see config.example.yaml for full list)
```

### Key Dependencies

The collector is written in Go and uses the official `openconfig/gnmi` protobuf
definitions with `google.golang.org/grpc`. Azure authentication uses Go
standard library (`crypto/hmac`, `crypto/sha256`). Configuration is loaded from
YAML. The binary cross-compiles to a static Linux amd64 executable (~12 MB)
with no runtime dependencies.

---

## Implementation Phases

### Phase 1 — Discovery and Validation ✅

- Enabled `feature grpc` on a test Nexus 9336 switch; validated gNMI
  Capabilities (gNMI 0.8.0, 29 OpenConfig + 3 native models)
- Tested `Get` requests for all 13 YANG paths; documented which paths work
  and discovered NX-OS-specific quirks (JSON encoding only, BGP requires
  network-instance path, base64-encoded float values in power-supply data)
- Compared gNMI response structure with current parser JSON output for each
  telemetry type; documented coverage gaps

### Phase 2 — Core gNMI Client and Transformers ✅

- Created `src/TelemetryClient/` Go project with gNMI client wrapper (TLS,
  auth, Get, Capabilities), YAML config loader, and Azure Log Analytics
  logger (HMAC-SHA256 signing, key failover)
- Implemented 11 transformers covering all 13 YANG paths, producing JSON
  output compatible with existing Azure tables and Grafana dashboards
- 35 unit tests pass; binary cross-compiles to 12 MB Linux amd64 executable

### Phase 3 — On-Switch Validation (in progress)

- Deploy collector to switch, run side-by-side with existing cron pipeline
- Compare `GnmiTest*_CL` tables against `Cisco*_CL` production data
- Resolve VRF split (see proposal above)
- Validate data quality over 24–48 hours

### Phase 4 — Subscription Engine (planned)

- Implement subscription manager supporting SAMPLE and ON_CHANGE modes
- Implement batching/buffering for Azure Log Analytics API
- Implement reconnection, retry, and dead-letter logging for failed sends

### Phase 5 — Integration and Deployment (planned)

- Update `Arcnet_Cisco_Arc_Setup` to optionally deploy the gNMI collector
  instead of the cron-based collector
- Test on NX-OS 9.x and 10.x; validate all telemetry tables receive data
- Dual-mode transition period: both collectors run in parallel

---

## PROPOSAL: VRF Split — Challenges and Solutions

> **Problem**: The gNMI server on NX-OS is bound to the **management VRF** (a
> separate Linux network namespace). Azure Log Analytics is only reachable from
> the **default VRF**. A process running in one namespace cannot reach endpoints
> in the other. This section evaluates design options for bridging the gap.

### Background

On Cisco NX-OS, VRFs are implemented as Linux network namespaces. The gNMI
server listens in the management namespace while Azure endpoints are only
routable from the default namespace. Cross-VRF routing is not configured and
is typically disallowed by network policy.

```text
┌──────────────────────────┐      ┌──────────────────────────┐
│  management VRF (netns)  │      │  default VRF (netns)     │
│                          │      │                          │
│  ✅ gNMI 127.0.0.1:50051│      │  ✅ Azure DNS/HTTP       │
│  ✅ SSH mgmt interface   │      │  ✅ vsh, cron, parsers   │
│  ❌ Azure DNS/HTTP       │      │  ❌ gNMI (not listening) │
└──────────────────────────┘      └──────────────────────────┘
          │                                   │
          └─── NO cross-VRF routing ──────────┘
```

The current cron pipeline works because it runs entirely in default VRF: `vsh`
is accessible from both namespaces, and Azure HTTP is reachable from default.
The gNMI collector cannot do the same because the gNMI server only listens in
the management namespace.

---

### Option A: Enable gNMI on Default VRF (`grpc use-vrf default`)

**NX-OS supports running gNMI on two VRFs simultaneously.** The `grpc use-vrf
default` command (documented in the [NX-OS Programmability Guide](https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/programmability/cisco-nexus-9000-series-nx-os-programmability-guide-104x/m-grpc-agent.html))
starts a second gRPC server in the default VRF:

```text
configure terminal
  grpc use-vrf default
end
copy running-config startup-config
```

With this, the collector runs entirely in default VRF:

```text
┌──────────────────────────────────────────┐
│           default VRF (netns)            │
│                                          │
│  gNMI server ──→ 127.0.0.1:50051        │
│  gnmi-collector ──→ gNMI Get             │
│  gnmi-collector ──→ Azure HTTP POST      │
│                                          │
│  Single process, single namespace.       │
└──────────────────────────────────────────┘
```

| Criteria | Assessment |
|----------|------------|
| **Complexity** | Very low — one config line on the switch, no code changes |
| **Streaming support** | ✅ Yes — single process can do Subscribe + HTTP |
| **Deployment** | No wrapper script needed, binary runs standalone |
| **Risk** | Known NX-OS bug: disabling `use-vrf default` later breaks management VRF gNMI (workaround: toggle `feature grpc`). Must verify on our NX-OS version (10.4). Security team must approve exposing gRPC in default VRF. |
| **Recommendation** | **Preferred long-term solution** if approved |

#### Action items to validate

1. Test `grpc use-vrf default` on rr1-n42-r07-9336hl-13-1a
2. Verify `gNMI Capabilities` works from default VRF
3. Confirm no disruption to management VRF gNMI
4. Get security team sign-off on gRPC in default VRF
5. Check if default VRF gNMI uses the same TLS cert or needs a separate one

---

### Option B: Two-Phase Wrapper Script (Current Workaround)

The collector writes transformed JSON to disk from management VRF; a wrapper
script sends them from default VRF using the existing azure-logger.

```text
┌── management VRF ──────────────────────┐
│  gnmi-collector -once -output /tmp/out │
│    → GnmiTestInterfaceCounter.json     │
│    → GnmiTestBgpSummary.json           │
│    → (13 files total)                  │
└──────────────┬─────────────────────────┘
               │  (filesystem, shared across namespaces)
┌── default VRF ┴────────────────────────┐
│  for f in /tmp/out/*.json; do          │
│    azure-logger send "$table" "$f"     │
│  done                                  │
└────────────────────────────────────────┘
```

The wrapper script (run from cron in default VRF, matching the current
pipeline):

```bash
#!/bin/bash
# /opt/gnmi-collector-wrapper.sh
OUTPUT_DIR="/tmp/gnmi-out"
LOGGER="/opt/cisco-azure-logger-v2.sh"

rm -rf "$OUTPUT_DIR"

# Phase 1: collect in management VRF
chvrf management /tmp/gnmi-collector \
  -config /tmp/gnmi-config.yaml \
  -once -output "$OUTPUT_DIR"

# Phase 2: send from default VRF
for f in "$OUTPUT_DIR"/*.json; do
  [ -f "$f" ] || continue
  table=$(basename "$f" .json)
  "$LOGGER" send "$table" "$f"
done

rm -rf "$OUTPUT_DIR"
```

| Criteria | Assessment |
|----------|------------|
| **Complexity** | Low — shell script, reuses existing azure-logger |
| **Streaming support** | ❌ No — inherently polling (one-shot + cron) |
| **Deployment** | Familiar pattern (same cron model as current pipeline) |
| **Risk** | Low — already tested and working on the switch |
| **Recommendation** | **Use as the immediate production solution** |

---

### Recommendation

```text
                          Streaming    Complexity    Risk
                          ─────────    ──────────    ────
A. use-vrf default           ✅         Very Low      Med     ★ Long-term goal
B. Two-phase wrapper         ❌         Low           Low     ★ Immediate / production
```

**Phased approach:**

1. **Now → Validation**: Use **Option B** (wrapper script). It mirrors the
   existing cron pipeline, is already tested, and lets us validate data quality
   in the `GnmiTest*` Azure tables with zero risk.

2. **After validation**: Test **Option A** (`grpc use-vrf default`) on the
   test switch. If it works and is approved by security, migrate the collector
   to run standalone in default VRF. This eliminates the wrapper script, temp
   files, and enables future streaming.

---

## Risks and Considerations

1. **NX-OS gRPC support varies by version** — Older NX-OS 9.2.x may have
   limited YANG model support. Needs verification on actual hardware.
2. **OpenConfig vs native models** — Not all data is available via OpenConfig.
   Some telemetry types (class-map, version) may require Cisco native YANG
   paths.
3. **Resource constraints** — Cisco Nexus switches have limited CPU/memory for
   guest processes. The gNMI client must be lightweight.
4. **On-switch vs off-switch** — The gNMI client can run directly on the
   switch (like current tools) or on an external collector host. On-switch is
   simpler but more resource-constrained.
5. **Dual-mode transition** — During migration, both cron-based and gNMI
   collectors may need to coexist to avoid data gaps.

### Open Challenges

The following operational challenges have been identified during prototyping
and need to be resolved before production deployment:

1. **VRF isolation** — The gNMI server runs in the management VRF while Azure
   is only reachable from the default VRF. Two viable solutions are documented
   in the VRF proposal section above. This must be resolved as part of Phase 3.

2. **TLS certificate management** — NX-OS auto-generates a self-signed gRPC
   certificate valid for only **1 day**. During prototyping we manually
   generated a longer-lived cert (825 days) via openssl and imported it as
   PKCS12. A production deployment needs an automated certificate lifecycle:
   rotation schedule, CA-signed certificates vs self-signed, distribution
   across the fleet, and monitoring for expiration. If using `skip_verify` in
   the client (as we do today), the security implications must be accepted.

3. **gRPC credential management** — gNMI authentication requires NX-OS
   username and password passed as gRPC metadata on every connection. Currently
   these are stored as environment variables (`$GNMI_USER`, `$GNMI_PASS`) on
   the switch. For production, we need a secure credential strategy: dedicated
   service account with least-privilege RBAC role, secret storage (e.g., Azure
   Key Vault retrieval at startup, encrypted config file, or integration with
   the switch's AAA/TACACS+ infrastructure), and credential rotation without
   service interruption.

4. **Daemon lifecycle management** — Once the collector moves to Subscribe
   (streaming) mode, it becomes a long-running daemon rather than a cron job.
   NX-OS does not have real systemd, but the existing ArcNet onboarding script
   already creates init.d shims to manage Arc agent services. The collector
   will need the same treatment: an `/etc/init.d/gnmi-collector` script
   supporting `start|stop|restart|status`, a PID file for tracking, automatic
   restart on crash, and persistence across switch reboots. Until then, the
   process is managed manually via `nohup` / `kill` and the binary already
   handles `SIGTERM` for graceful shutdown.

---

## Future Opportunity: Configuration Management via gNMI Set

> This section explores using the same gRPC/gNMI infrastructure we are building
> for telemetry to also **push configuration** to switches, replacing manual CLI
> sessions and ad-hoc scripts.

### What We Have Today

The entire ArcNet project is **read-only**. Both the legacy cron pipeline
(`vsh -c "show ..."`) and the new gNMI collector (Get only) observe switch
state but never modify it. There are no configuration push mechanisms, no
Ansible playbooks, no Terraform providers, and no NETCONF/RESTCONF usage in
the codebase. Switch configuration changes are done manually via CLI.

### What gNMI Set Provides

The gNMI protocol includes a `Set` RPC that supports three operations within a
single **atomic transaction**:

| Operation | Behavior | Use Case |
|-----------|----------|----------|
| **Update** | Create or modify a leaf or subtree | Set an interface description, enable a feature |
| **Replace** | Overwrite an entire subtree; children not present in the payload are deleted | Push a complete BGP neighbor config |
| **Delete** | Remove a path and all its children | Remove a static route, disable an interface |

All operations in a single `Set` request execute atomically — if any fails, the
entire request is rolled back. This is a fundamental improvement over CLI
scripting where partial failures can leave a switch in an inconsistent state.

```text
Example: gNMI Set request (single atomic transaction)
┌─────────────────────────────────────────────────────────────────────┐
│ delete:  /interfaces/interface[name=Eth1/48]/config/description    │
│ update:  /interfaces/interface[name=Eth1/48]/config/enabled = true │
│ update:  /interfaces/interface[name=Eth1/48]/config/mtu = 9216     │
│                                                                     │
│ → All succeed or all fail. No partial state.                        │
└─────────────────────────────────────────────────────────────────────┘
```

### Configuration Paths Available on NX-OS

YANG models separate **config** (read-write) and **state** (read-only)
containers. Our telemetry collector reads from `state` paths; configuration
management would write to `config` paths on the same models:

| Use Case | YANG Path (`config` container) |
|----------|-------------------------------|
| Enable/disable interface | `/openconfig-interfaces:interfaces/interface[name=...]/config/enabled` |
| Set interface description | `.../interface[name=...]/config/description` |
| Set MTU | `.../interface[name=...]/config/mtu` |
| Configure BGP peer | `/openconfig-network-instance:.../bgp/neighbors/neighbor[neighbor-address=...]/config` |
| Set BGP peer ASN | `.../config/peer-as` |
| Add static route | `/openconfig-network-instance:.../static-routes/static[prefix=...]/config` |
| Create/delete VLAN | `/openconfig-network-instance:.../vlans/vlan[vlan-id=...]/config` |
| Set system hostname | `/openconfig-system:system/config/hostname` |
| Configure NTP server | `/openconfig-system:system/ntp/servers/server[address=...]/config` |
| Configure DNS | `/openconfig-system:system/dns/servers/server[address=...]/config` |
| Set LLDP admin status | `/openconfig-lldp:lldp/config/enabled` |

> **Note**: Not all config paths are writable on every NX-OS version. OpenConfig
> models may be read-only for some features on Cisco; the Cisco native YANG
> models (`Cisco-NX-OS-device`) tend to have broader write support. This must
> be validated per switch model and NX-OS version.

### Advantages Over CLI-Based Configuration

| Aspect | CLI (current) | gNMI Set (proposed) |
|--------|--------------|---------------------|
| **Atomicity** | None — commands execute sequentially; partial failure leaves inconsistent state | Full — all-or-nothing transaction per Set request |
| **Validation** | Runtime only — syntax errors caught one-at-a-time | YANG schema enforces types, ranges, and enums before applying |
| **Idempotency** | Imperative — "add this", "remove that" | Declarative — "desired state is X" |
| **Versioning** | Output format changes across NX-OS versions break scripts | YANG models are versioned; paths are stable |
| **Auditability** | Requires scraping `show configuration session` | Every Set is a structured gRPC call with full request/response logging |
| **Automation** | Expect scripts, SSH screen-scraping | Native gRPC client in any language (Go, Python, etc.) |
| **Rollback** | `checkpoint` / `rollback running-config` (NX-OS native) | Re-issue a Set with previous state (no native rollback RPC) |

### Risks and Considerations

1. **Write access is high-risk** — A bad Set can take down interfaces, break
   BGP peering, or cause a network outage. Must have robust testing, staging,
   and approval workflows.

2. **RBAC enforcement** — The gNMI user credentials must have appropriate NX-OS
   role privileges. Consider a dedicated read-write user with scoped
   permissions, separate from the telemetry read-only user.

3. **Writable path coverage** — OpenConfig `config` containers may be read-only
   on NX-OS for some features. Cisco native YANG paths typically have better
   write coverage. Must audit which paths are actually writable.

4. **No native rollback** — gNMI does not have a `Rollback` RPC. NX-OS
   `checkpoint`/`rollback` is a CLI feature. A config management tool would
   need to implement its own rollback by storing previous state and re-issuing
   Set on failure.

5. **Collision with other config sources** — If operators also make changes via
   CLI, there's no conflict detection. Need a clear ownership model: either
   gNMI owns certain subtrees, or changes are coordinated through a single
   source of truth.

6. **Startup config persistence** — gNMI Set modifies running-config. A
   `copy running-config startup-config` equivalent may be needed. NX-OS
   supports this via `Cisco-NX-OS-device` YANG model or gNOI (gRPC Network
   Operations Interface).

### Incremental Path from Telemetry to Configuration

The gNMI client, TLS setup, authentication, and connection management we built
for telemetry in `src/TelemetryClient/` are reusable. Adding Set support is
incremental:

```text
Phase    Scope                          Risk
─────    ─────                          ────
Current  gNMI Get (telemetry)           Read-only, safe
  ↓
Next     gNMI Set (description, MTU)    Low-risk leaf changes
  ↓
Later    gNMI Set (BGP, VLANs, routes)  High-risk, needs staging
  ↓
Future   gNMI Subscribe + Set (closed   Full automation loop
         loop automation)
```

### Validation Steps Before Implementation

1. **Audit writable paths** — Run `gNMI Set` with dry-run / test-only against
   non-critical paths (e.g., interface description) on the test switch to
   confirm which OpenConfig and native paths accept writes.

2. **Compare with gNOI** — NX-OS also supports gNOI (gRPC Network Operations
   Interface) for operational tasks like `System.Reboot`, `OS.Install`,
   `Cert.Rotate`. Evaluate whether gNOI complements gNMI Set for our needs.

3. **Define ownership model** — Decide which config domains gNMI Set would own
   vs. what remains CLI-managed. Start with low-risk, high-frequency changes
   (descriptions, enable/disable).

4. **Build a read-before-write pattern** — For any Set, first Get the current
   state, store it as a checkpoint, then Set the new state. This enables
   rollback.

---

## References

- [gNMI Specification (OpenConfig)](https://github.com/openconfig/gnmi)
- [OpenConfig YANG Models](https://github.com/openconfig/public)
- [Cisco: Configure and Verify gRPC gNMI on Nexus 9k](https://www.cisco.com/c/en/us/support/docs/switches/nexus-9000-series-switches/220640-configure-and-verify-grpc-gnmi-on-nexus.html)
- [Cisco NX-OS Programmability Guide — gNMI](https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/programmability/cisco-nexus-9000-series-nx-os-programmability-guide-104x/m-grpc-agent.html)
- [OpenConfig RPM packages for NX-OS 9.3.x](https://devhub.cisco.com/ui/native/open-nxos-agents/)
- [gnmic — gNMI CLI Client](https://gnmic.openconfig.net/)
- [Cisco YANG Model Explorer](https://developer.cisco.com/yangsuite/)
- [Azure Log Analytics HTTP Data Collector API](https://learn.microsoft.com/en-us/azure/azure-monitor/logs/data-collector-api)
