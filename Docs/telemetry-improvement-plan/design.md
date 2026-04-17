# Telemetry Improvement Plan — Design Document

## 1. Problem Statement

### Old Pipeline (CLI + Regex)

The legacy telemetry pipeline polls switch data every 5 minutes via cron:

1. **Collect**: `vsh -c "show ..."` CLI commands (14 sequential, `sleep 2` between each → ~30s per cycle)
2. **Parse**: Go regex parsers extract fields from semi-structured CLI text output
3. **Enrich**: Python script adds metadata (device name, timestamp, region)
4. **Ship**: Bash `curl` posts JSON to Azure Log Analytics HTTP API

**Pain points**: Regex parsers break when NX-OS output format changes across versions. Polling-only with 5-minute granularity — no event-driven data. Adding new telemetry requires writing a new text parser from scratch.

### New Pipeline (gNMI + YANG)

The new pipeline uses industry-standard gNMI/YANG for structured telemetry:

1. **Collect**: gNMI `Get` (poll) or `Subscribe` (stream) over a single gRPC connection
2. **Parse**: YANG models return structured JSON/JSON_IETF — no regex needed
3. **Transform**: Go transformer functions normalize fields to common schema
4. **Ship**: Go Azure logger posts JSON with HMAC-SHA256 auth

**Benefits**: Version-resilient YANG models, structured data by default, supports both poll and streaming modes, single binary for multiple platforms, adding new telemetry = add a Go file.

---

## 2. Architecture Overview

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          NETWORK SWITCH                                     │
│                                                                             │
│   ┌──────────────────────┐     ┌──────────────────────────────────────┐     │
│   │  Management VRF      │     │  Default VRF                         │     │
│   │  (mgmt0 interface)   │     │  (data interfaces, Azure reachable)  │     │
│   │                      │     │                                      │     │
│   │  SSH access (:22)    │     │  ┌─────────────────────────────┐     │     │
│   │                      │     │  │  gNMI Server (gRPC)         │     │     │
│   └──────────────────────┘     │  │  Cisco: port 50051          │     │     │
│                                │  │  SONiC: port 8080           │     │     │
│   ┌──────────────────────┐     │  │                             │     │     │
│   │  YANG Datastore      │     │  │  • TLS (self-signed cert)   │     │     │
│   │  ┌────────────────┐  │     │  │  • Username/password auth   │     │     │
│   │  │ OpenConfig     │  │◄────┤  │    via gRPC metadata        │     │     │
│   │  │ models         │  │     │  └─────────────┬───────────────┘     │     │
│   │  ├────────────────┤  │     │                │                     │     │
│   │  │ Native/vendor  │  │     └────────────────┼─────────────────────┘     │
│   │  │ models (Cisco) │  │                      │                           │
│   │  └────────────────┘  │                      │                           │
│   └──────────────────────┘                      │                           │
└─────────────────────────────────────────────────┼───────────────────────────┘
                                                  │
                              gRPC/TLS connectio  │
                              (single persistent  │
                               channel)           │
                                                  │
┌─────────────────────────────────────────────────┼───────────────────────────┐
│  gnmi-collector  (static Linux amd64 binary, ~12 MB)                        │
│                                                 │                           │
│  ┌──────────────────────────────────────────────┼────────────────────────┐  │
│  │  gNMI Client (internal/gnmi/client.go)       │                        │  │
│  │                                              ▼                        │  │
│  │  grpc.DialContext(target:port)                                        │  │
│  │  ├─ TLS: TOFU (trust-on-first-use) or skip_verify                     │  │
│  │  ├─ Auth: gRPC metadata{username, password}                           │  │
│  │  ├─ Max msg: 64 MB                                                    │  │
│  │  │                                                                    │  │
│  │  ├─ Get()         ─── poll mode (one-shot request/response)           │  │
│  │  ├─ Subscribe()   ─── stream mode (persistent server-push)            │  │
│  │  └─ Capabilities()─── YANG model discovery                            │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                              │                                              │
│                              ▼                                              │
│  ┌──────────────┐   ┌───────────────────────────────────────────────────┐   │
│  │ Config       │   │  Collector / Subscriber                           │   │
│  │ (YAML)       │──►│  ┌─────────────────────────────────────────────┐  │   │
│  │              │   │  │  Shared Pipeline                            │  │   │
│  │ • device_type│   │  │                                             │  │   │
│  │ • paths[]    │   │  │  raw gNMI JSON                              │  │   │
│  │ • mode       │   │  │       │                                     │  │   │
│  │ • encoding   │   │  │       ▼                                     │  │   │
│  └──────────────┘   │  │  ┌──────────┐   ┌──────────┐   ┌─────────┐  │  │   │
│                     │  │  │Transform │──►│ Prefix   │──►│ Merge   │  │  │   │
│                     │  │  │(registry)│   │ (device_ │   │ByData   │  │  │   │
│                     │  │  └──────────┘   │  type)   │   │ Type    │  │  │   │
│                     │  │       ▲         └──────────┘   └────┬────┘  │  │   │
│                     │  └───────┼──────────────────────────────┼──────┘  │   │
│                     └──────────┼──────────────────────────────┼─────────┘   │
│                                │                              │             │
│  ┌─────────────────────────────┴──────┐                       │             │
│  │  Transformer Registry              │                       ▼             │
│  │  (self-registration via init())    │       ┌──────────────────────────┐  │
│  │                                    │       │  Azure Logger            │  │
│  │  16 OpenConfig (shared)            │       │  (internal/azure/)       │  │
│  │  ├─ interface-counters             │       │                          │  │
│  │  ├─ bgp-neighbors                  │       │  HTTPS POST → Log        │  │
│  │  ├─ system-cpus / system-memory    │       │  Analytics HTTP API      │  │
│  │  └─ ... (13 more)                  │       │  Auth: HMAC-SHA256       │  │
│  │                                    │       └────────────┬─────────────┘  │
│  │  9 Cisco-native                    │                    │                │
│  │  ├─ nx-sys-cpu / nx-sys-memory     │                    │                │
│  │  ├─ nx-bgp-peers / nx-arp          │                    ▼                │
│  │  └─ ... (5 more)                   │       ┌──────────────────────────┐  │
│  └────────────────────────────────────┘       │  Azure Log Analytics     │  │
│                                               │  (CommonFields JSON)     │  │
│                                               └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Project Structure

```
src/TelemetryClient/
├── cmd/gnmi-collector/main.go     # Entry point, CLI flags
├── internal/
│   ├── config/config.go           # YAML config with env var resolution
│   ├── gnmi/client.go             # gNMI client (TLS, auth, Get/Subscribe)
│   ├── azure/logger.go            # Azure Log Analytics HTTP API
│   ├── transform/                 # 25 transformers + registry
│   │   ├── registry.go            # Self-registration pattern
│   │   ├── interface_counters.go  # OpenConfig transformers
│   │   └── native_*.go            # Cisco-native transformers
│   └── collector/
│       ├── collector.go           # Poll orchestrator (RunOnce)
│       └── subscriber.go          # Subscribe orchestrator (RunStream)
├── config.cisco.yaml              # Cisco NX-OS config
├── config.sonic.yaml              # SONiC config
└── go.mod
```

---

## 3. Platform Support

| Platform | gNMI Support | Port | Encoding | YANG Models | Config File | Status |
|---|---|---|---|---|---|---|
| Cisco NX-OS | `feature grpc` | 50051 | JSON | OpenConfig + Cisco native | config.cisco.yaml | ✅ Validated |
| SONiC (Dell Enterprise) | `sonic-gnmi` container | 8080 | JSON_IETF | OpenConfig only | config.sonic.yaml | ✅ Validated |
| Dell OS10 | Requires SFD mode | — | — | — | — | ❌ Not supported |

**Dell OS10 rejection**: gNMI requires SmartFabric Director mode, which is incompatible with production Full Switch mode. Dell OS10 continues using CLI parsers.

**Single binary, multi-platform**: The same `gnmi-collector` binary serves both Cisco and SONiC. The YAML config determines `device_type`, paths, encoding, and table name prefixes. 5 data types use identical OpenConfig paths on both platforms. 9 data types use Cisco-native `/System/...` paths on NX-OS but equivalent OpenConfig paths on SONiC.

---

## 4. Old vs New Pipeline Comparison

### Architecture Comparison

| Aspect | Old Pipeline | New Pipeline |
|---|---|---|
| Transport | SSH → `vsh -c "show ..."` CLI | gRPC → gNMI Get/Subscribe |
| Data Format | Semi-structured text | Structured JSON / JSON_IETF |
| Parsing | Go regex parsers (fragile) | YANG models (version-resilient) |
| Metadata | Python script | Go (built-in) |
| Shipping | Bash `curl` | Go Azure logger (HMAC-SHA256) |
| Modes | Poll only (5-min cron) | Poll (Get) + Stream (Subscribe) |
| Cycle Time | ~30s (14 commands × sleep 2) | ~2s (single gRPC connection) |
| New Telemetry | Write new text parser | Add Go file with transformer |

### Field Coverage by Table

| Table | Old Fields | Matched | Coverage | Notes |
|---|---|---|---|---|
| Interface Counters | 12 | 12 | 100% | +4 bonus (errors, discards) |
| Interface Status | 7 | 6 | 86% | VLAN via native YANG |
| BGP Summary | ~18 | ~10 | 55% | Flattened structure |
| LLDP Neighbor | 17 | 17 | 100% | 4 type mismatches |
| Env Temperature | 6 | 6 | 100% | Perfect match |
| Env Power | ~20 | ~12 | 60% | CLI-only aggregations missing |
| ARP Table | 13 | 9 | 69% | Missing flag parsing |
| MAC Table | 12 | 8 | 67% | CLI-only flags missing |
| Transceiver | 16 | 12 | 75% | DOM via separate path |
| System Uptime | 11 | 11 | 100% | Type mismatches (string↔int) |
| System Resources | 17 | 10 | 59% | Linux /proc fields missing |
| Inventory | 6 | 5 | 83% | version_id missing |

**Overall: ~140/168 fields matched (~83%).** ~15 fields genuinely unavailable via YANG (Linux `/proc` data: load_avg, vmalloc, processes).

---

## 5. Key Technical Decisions

- **VRF Isolation (Cisco)**: gNMI server runs in management VRF; Azure is reachable from default VRF. Solution: `grpc use-vrf default` runs gNMI on both VRFs. Requires security team approval.

- **Encoding**: Cisco uses JSON, SONiC uses JSON_IETF (RFC 7951). JSON_IETF adds module prefixes to keys (e.g., `openconfig-interfaces:name`). Fixed with `stripModulePrefixes()` in the gNMI client — single-point fix for all transformers.

- **Subscribe ONCE Fallback**: SONiC returns empty `{}` for bulk Get on list paths. The collector automatically falls back to Subscribe ONCE (one-shot stream) which returns per-entity data correctly.

- **Native vs OpenConfig**: OpenConfig covers ~60% of needed fields. Adding Cisco-native `/System/...` paths brings coverage to ~95%. SONiC uses OpenConfig only (no native Cisco paths available).

- **Self-Registration Pattern**: Transformers self-register via `init()` → `Register(name, factory)`. The collector calls `BuildMap()` to assemble the active set. Adding a new vendor = create new Go files, no changes to collector code.

- **device_type Required**: Config validation errors if `device_type` is empty. Prevents silent misconfigs where SONiC would produce `cisco_nexus_*` table prefixes.

- **mergeByDataType**: CPU and memory entries are merged into a single system-resources row in both poll and subscribe modes, matching the old pipeline's output format.

- **TLS**: Default mode is TOFU (trust-on-first-use) — the server cert is fetched on startup and used for verification during the session. `skip_verify: true` and `ca_file` are also supported. NX-OS auto-generates a self-signed certificate; longer-lived certs can be imported manually.

---

## 6. gRPC Connection, TLS & VRF

### gRPC Connection

The collector establishes a **single persistent gRPC channel** to the
switch's gNMI server. All telemetry RPCs (Get, Subscribe, Capabilities)
are multiplexed over this one connection.

```
grpc.DialContext(ctx, "switch-ip:port", opts...)
  ├─ Max receive message: 64 MB (handles large subtree responses)
  ├─ Transport: TLS or insecure (config-driven)
  └─ Auth: username/password injected as gRPC metadata on every call
```

| Parameter | Cisco NX-OS | SONiC (Dell Enterprise) |
|-----------|-------------|------------------------|
| Port | 50051 | 8080 |
| TLS | Enabled | Enabled |
| Encoding | JSON | JSON_IETF (RFC 7951) |
| Auth | gRPC metadata (user/pass) | gRPC metadata (user/pass) |
| Max message | 64 MB | 64 MB |

Authentication is **not** certificate-based (mTLS). Credentials come from
environment variables (`GNMI_USER`, `GNMI_PASS`) resolved at startup via
the config's `username_env` / `password_env` fields. Each gRPC call
injects them as metadata headers:

```go
md := metadata.Pairs("username", c.username, "password", c.password)
ctx = metadata.NewOutgoingContext(ctx, md)
```

### TLS Certificate Lifecycle

Both platforms use **self-signed certificates**. The collector supports
three TLS modes, configured in the YAML:

```
┌──────────────────────────────────────────────────────────────────┐
│  TLS Decision Flow (gnmi/tls.go)                                 │
│                                                                  │
│  config.yaml                    tls.go                           │
│  ┌──────────────┐               ┌────────────────────────────┐   │
│  │ tls:         │               │ if ca_file set:            │   │
│  │   enabled: T │──────────────►│   load CA → verify server  │   │
│  │   ca_file: ""│               │ elif skip_verify:          │   │
│  │   skip_verify│               │   InsecureSkipVerify: true │   │
│  │     : false  │               │ else (default — TOFU):     │   │
│  └──────────────┘               │   fetch cert on connect,   │   │
│                                 │   pin for session lifetime  │   │
│                                 └────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

**Cisco NX-OS specifics**:
- `feature grpc` auto-generates a self-signed cert valid for **24 hours**
- TOFU (default) handles this automatically — the cert is fetched fresh
  on each collector startup and trusted for the session
- For longer validity, we manually generated an 825-day cert via:
  `openssl req → openssl x509 → openssl pkcs12 → NX-OS crypto import`
- **Production concern**: needs automated certificate rotation or a
  proper CA-signed cert distributed via fleet management

**SONiC specifics**:
- The `sonic-gnmi` container generates a self-signed cert at startup
- Cert persists across container restarts (stored in `/etc/sonic/`)
- TOFU handles this automatically

**Future improvements**:
- Support mTLS for environments requiring mutual authentication
- Integrate with fleet cert management for automated rotation

### VRF Routing

VRF (Virtual Routing and Forwarding) creates isolated routing domains on
the switch. This matters because the gNMI server and Azure endpoints may
live in different VRFs.

```
┌─────────────────────────────────────────────────────────────────┐
│                         Cisco NX-OS Switch                      │
│                                                                 │
│   ┌─────────────────────┐       ┌─────────────────────────┐    │
│   │  Management VRF     │       │  Default VRF            │    │
│   │                     │       │                          │    │
│   │  • mgmt0 interface  │       │  • Data interfaces      │    │
│   │  • SSH access       │       │  • Azure reachable      │    │
│   │  • OOB management   │       │  • Production traffic   │    │
│   │                     │       │                          │    │
│   │  gNMI server binds  │       │  gNMI server ALSO       │    │
│   │  here by default    │       │  binds here with:       │    │
│   │                     │       │  "grpc use-vrf default"  │    │
│   └─────────────────────┘       └──────────┬──────────────┘    │
│                                             │                   │
└─────────────────────────────────────────────┼───────────────────┘
                                              │
                              gRPC :50051     │
                                              ▼
                                    ┌──────────────────┐
                                    │  gnmi-collector   │
                                    │  (runs on switch  │
                                    │   or remote host) │
                                    │         │         │
                                    │         ▼         │
                                    │  HTTPS → Azure    │
                                    │  Log Analytics    │
                                    └──────────────────┘
```

**The problem**: By default, NX-OS gNMI (`feature grpc`) only listens in
the **management VRF**. But the collector needs to reach Azure Log
Analytics, which is routable from the **default VRF**. If the collector
runs on the switch itself, it can't reach both.

**The solution**: `grpc use-vrf default` tells NX-OS to bind the gRPC
server in the default VRF **in addition to** management VRF. The
collector connects via default VRF (port 50051) and can also reach Azure.

```
! Cisco NX-OS configuration
feature grpc
grpc use-vrf default        ← enables gNMI on both VRFs
grpc certificate-lifetime 825
```

> ⚠️ **Security note**: Exposing gRPC on the default (data) VRF
> requires security team approval. This broadens the attack surface
> beyond out-of-band management. Mitigation: ACLs restricting gRPC
> access to known collector IPs.

**SONiC**: No VRF split issue. The `sonic-gnmi` container listens on all
interfaces (port 8080). Azure is reachable from the same network
namespace.

**VRF in YANG paths**: VRF is also relevant in data queries — BGP, MAC,
and ARP paths require a network-instance (VRF) selector:
```yaml
# SONiC config — explicit VRF key in YANG path
yang_path: /openconfig-network-instance:network-instances/network-instance[name=default]/...
```

---

## 7. YANG Path Reference

| Config Key | YANG Path (abbreviated) | Source | Used By |
|---|---|---|---|
| interface-counters | `.../interface/state/counters` | OpenConfig | Both |
| interface-status | `.../interface/state` | OpenConfig | Both |
| if-ethernet | `.../interface/ethernet/state` | OpenConfig | Both |
| system-state | `/system/state` | OpenConfig | Both |
| system-cpus | `/system/cpus` | OpenConfig | SONiC |
| system-memory | `/system/memory` | OpenConfig | SONiC |
| platform-inventory | `.../components/component` | OpenConfig | Both |
| bgp-neighbors | `.../bgp/neighbors/neighbor/state` | OpenConfig | SONiC |
| bgp-global | `.../bgp/global/state` | OpenConfig | Both |
| lldp-neighbors | `.../lldp/.../neighbors` | OpenConfig | SONiC |
| arp-table | `.../subinterface/ipv4/neighbors` | OpenConfig | SONiC |
| mac-table | `.../network-instance/fdb/mac-table` | OpenConfig | SONiC |
| nx-sys-cpu | `/System/procsys-items/syscpusummary-items` | Cisco Native | Cisco |
| nx-sys-memory | `/System/procsys-items/sysmem-items` | Cisco Native | Cisco |
| nx-arp | `/System/arp-items/.../AdjEp-list` | Cisco Native | Cisco |
| nx-bgp-peers | `/System/bgp-items/.../Peer-list` | Cisco Native | Cisco |
| nx-lldp | `/System/lldp-items/.../If-list` | Cisco Native | Cisco |
| nx-env-sensor | `/System/ch-items/.../sensor-items` | Cisco Native | Cisco |
| nx-env-psu | `/System/ch-items/.../PsuSlot-list` | Cisco Native | Cisco |
| nx-mac-table | `/System/mac-items` | Cisco Native | Cisco |
| nx-transceiver | `/System/intf-items/.../phys-items` | Cisco Native | Cisco |
