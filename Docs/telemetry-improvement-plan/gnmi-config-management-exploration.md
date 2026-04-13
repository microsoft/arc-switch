# gNMI Configuration Management via Azure Arc — Technical Exploration

> **Status**: Exploration / Research  
> **Date**: April 2026  
> **Author**: Architecture discussion — not yet approved for implementation

## 1. Vision

Today, the gnmi-collector is a **read-only telemetry agent**: it issues gNMI `Get` and
`Subscribe` RPCs to collect operational state from network switches and ships the data to
Azure Log Analytics. The switches are registered as Azure Arc-connected machines, but Arc
is used only for identity — the collector runs as a standalone daemon.

The idea is to **close the loop**: use gNMI `Set` operations to push configuration changes
_to_ the switch, orchestrated from Azure through the Arc control plane. A customer could,
for example, change an interface description, adjust a BGP peer policy, or shut down a
port — all from the Azure portal or an ARM template — and the change would be applied on
the physical switch via gNMI.

---

## 2. How Azure Arc Routes Requests to an Extension

Understanding the request flow is critical before designing a config agent.

### 2.1 Arc Agent Architecture (on the switch)

The Azure Connected Machine agent (`azcmagent`) runs three services:

| Service | Purpose |
|---------|---------|
| **HIMDS** (himds) | Heartbeat, Azure identity, Instance Metadata (IMDS endpoint at `localhost:40342`) |
| **Extension Service** (extd / gc_extension_service) | Installs, upgrades, and manages VM extensions |
| **Machine Configuration** (gcad / gc_arc_service) | Azure Policy / Guest Configuration compliance |

On **Cisco NX-OS** these run as init.d daemons (no systemd). On **SONiC** they use systemd.

### 2.2 Request Flow: Azure Portal → Switch

```
┌──────────────────┐
│  Azure Portal /  │
│  ARM Template /  │  1. PUT/PATCH on ARM resource
│  CLI / SDK       │     (e.g., Microsoft.HybridCompute/machines/{name}/extensions/{ext})
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Azure Resource  │  2. ARM persists desired state, enqueues operation
│  Manager (ARM)   │     for the Extension Service to pick up
└────────┬─────────┘
         │  (outbound HTTPS long-poll — the switch calls out, Azure never calls in)
         ▼
┌──────────────────┐
│  Extension       │  3. Extension Service on the switch polls ARM for pending
│  Service (extd)  │     operations, downloads extension package, invokes handler
│  on the switch   │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Extension       │  4. Handler script/binary runs with settings + protected
│  Handler         │     settings (JSON blobs). Reports status back via Extension
│  (our agent)     │     Service → ARM.
└──────────────────┘
```

**Key points:**

- **No inbound connectivity required.** The Arc agent on the switch initiates an outbound
  HTTPS connection and long-polls ARM for work. This is ideal for switches behind NAT or
  in restricted management VRFs.
- **Extensions are pull-based, not push-based.** There is inherent latency (seconds to
  low minutes) between an ARM operation and the handler executing on the switch.
- **Settings and Protected Settings** are the two JSON blobs passed to the handler.
  Protected settings are encrypted in transit and at rest — suitable for credentials.
- **Status reporting** is structured: the handler writes a status JSON file that the
  Extension Service uploads to ARM. The portal shows Succeeded / Failed / Transitioning.

### 2.3 Extension Handler Contract

An extension handler is a directory with a `HandlerManifest.json` and scripts/binaries:

```
/var/lib/waagent/Microsoft.MyPublisher.MyExtension-1.0.0/
├── HandlerManifest.json
├── bin/
│   └── gnmi-config-agent        # our binary
├── install.sh                    # called once on install
├── enable.sh                     # called to apply settings
├── disable.sh                    # called to deactivate
└── uninstall.sh                  # called on removal
```

When ARM delivers a new configuration, `enable.sh` is invoked with the settings JSON.
The handler reads the settings, performs work, and writes a status file.

### 2.4 Alternative: Custom Script Extension

Instead of publishing a full custom extension (which requires Microsoft partner
registration), we can use the **Custom Script Extension** (`CustomScript` for Linux)
as an envelope:

```bash
az connectedmachine extension create \
  --machine-name "switch01" \
  --name "ApplyBGPConfig" \
  --type "CustomScript" \
  --publisher "Microsoft.Azure.Extensions" \
  --settings '{"commandToExecute": "/opt/gnmi-config-agent/apply --config /tmp/desired.json"}' \
  --protected-settings '{"fileUris": ["https://storage.blob.core.windows.net/configs/desired.json"]}'
```

This is simpler to start with but less structured than a dedicated extension.

---

## 3. gNMI Set — Protocol Capabilities

The gNMI specification (v0.10.0+) defines the `Set` RPC for configuration management.

### 3.1 SetRequest Operations

A single `SetRequest` can contain three types of mutations, processed in this order:

| Operation | Semantics |
|-----------|-----------|
| **delete** | Remove the subtree at the specified path |
| **replace** | Replace the entire subtree — anything not in the payload is deleted |
| **update** | Merge the payload into the existing tree — additive, existing values preserved |

All operations in a single `SetRequest` are **transactional**: either all succeed or the
target rolls back to the pre-request state.

### 3.2 Example: Set an Interface Description

```protobuf
SetRequest {
  prefix: { origin: "openconfig", elem: [{name: "interfaces"}] }
  update: [{
    path: { elem: [
      {name: "interface", key: {"name": "Ethernet1/1"}},
      {name: "config"},
      {name: "description"}
    ]}
    val: { string_val: "Uplink to spine-01" }
  }]
}
```

### 3.3 Example: Shut Down an Interface

```protobuf
SetRequest {
  update: [{
    path: { elem: [
      {name: "interfaces"},
      {name: "interface", key: {"name": "Ethernet1/1"}},
      {name: "config"},
      {name: "enabled"}
    ]}
    val: { bool_val: false }
  }]
}
```

### 3.4 The Config vs State Distinction

In YANG, every path is either:
- **`/config`** — read-write intended configuration (what the operator wants)
- **`/state`** — read-only operational state (what the device is actually doing)

Our telemetry collector reads `/state` paths. Configuration management would target
`/config` paths. For example:

| Telemetry (Get) | Configuration (Set) |
|-----------------|---------------------|
| `/interfaces/interface/state/oper-status` | `/interfaces/interface/config/enabled` |
| `/interfaces/interface/state/counters` | `/interfaces/interface/config/description` |
| `/bgp/neighbors/neighbor/state/session-state` | `/bgp/neighbors/neighbor/config/peer-as` |

---

## 4. Vendor-Specific gNMI Set Support — The Hard Part

This is where the biggest challenges lie. Each vendor implements gNMI Set differently.

### 4.1 Cisco NX-OS

| Aspect | Details |
|--------|---------|
| **Set support** | ✅ Supported since NX-OS 9.3(x) |
| **OpenConfig config paths** | ⚠️ Partial — many OC config paths are not writable |
| **Native config paths** | ✅ Better coverage via `/System/...` native YANG |
| **Encoding** | JSON (not JSON_IETF) |
| **Transactions** | ✅ Full rollback on failure within a SetRequest |
| **Replace semantics** | ⚠️ Replace at container level can wipe child config |
| **Candidate datastore** | ❌ No candidate/commit model — Set applies directly to running config |
| **Known quirks** | • Set on some `/openconfig-*` paths silently succeeds but has no effect<br>• Native paths (`/System/intf-items/...`) are more reliable for config<br>• VRF awareness: must target correct VRF context<br>• ACL and route-map paths may not be Set-writable |

**Risk**: Cisco's OpenConfig implementation is primarily tuned for telemetry (Get/Subscribe).
Config paths that _look_ writable in the YANG model may not actually be implemented for Set.
Each path must be individually validated.

### 4.2 Dell Enterprise SONiC

| Aspect | Details |
|--------|---------|
| **Set support** | ✅ Full gNMI Set support (SONiC Management Framework) |
| **OpenConfig config paths** | ✅ Good coverage — SONiC's mgmt framework maps OC → internal DB |
| **SONiC native paths** | ✅ `/sonic-*` paths map directly to ConfigDB |
| **Encoding** | JSON_IETF (RFC 7951 module-prefixed) |
| **Transactions** | ✅ Transactional via ConfigDB transaction pipeline |
| **Replace semantics** | ✅ Proper replace — but **replace at root can wipe entire config** |
| **Candidate datastore** | ⚠️ SONiC 4.x+ supports candidate datastore (not all deployments) |
| **Known quirks** | • Module prefixes required in JSON_IETF values (e.g., `"openconfig-interfaces:interfaces"`)<br>• Some paths require restart of specific daemons after Set<br>• BGP config changes may require FRR restart depending on scope<br>• `/sonic-*` native paths often easier than OC equivalents |

**Advantage**: SONiC was designed with config management in mind. Its entire architecture
(ConfigDB → *mgr daemons → *syncd → ASIC) is built around declarative configuration, making
gNMI Set a natural fit.

### 4.3 Arista EOS (Future)

| Aspect | Details |
|--------|---------|
| **Set support** | ✅ Full gNMI Set — most mature implementation in the industry |
| **OpenConfig config paths** | ✅ Excellent — Arista is a founding member of OpenConfig |
| **Native paths** | ✅ `eos_native:` origin for full CLI-equivalent config |
| **Encoding** | JSON_IETF preferred |
| **Transactions** | ✅ Full candidate + commit model |
| **Replace semantics** | ✅ Proper subtree replace with rollback |
| **CLI origin** | ✅ Supports `origin: "cli"` — send raw CLI commands via gNMI Set |

Arista would be the easiest vendor to support for config management.

### 4.4 Vendor Comparison Matrix

| Capability | Cisco NX-OS | SONiC | Arista EOS |
|------------|:-----------:|:-----:|:----------:|
| gNMI Set supported | ✅ | ✅ | ✅ |
| OpenConfig config paths | ⚠️ Partial | ✅ Good | ✅ Excellent |
| Native config paths | ✅ | ✅ | ✅ |
| Transactional rollback | ✅ | ✅ | ✅ |
| Candidate datastore | ❌ | ⚠️ 4.x+ | ✅ |
| Dry-run / validate | ❌ | ⚠️ Limited | ✅ |
| Get-after-Set verification | ✅ | ✅ | ✅ |
| Config replace at subtree | ⚠️ Risky | ✅ | ✅ |

---

## 5. Technical Challenges

### 5.1 🔴 Critical: Path Portability Is Not Guaranteed for Config

Our telemetry collector works because **all vendors implement the same OpenConfig
`/state` paths** (with minor encoding differences). Configuration is different:

- The same OpenConfig `/config` path may be **implemented on SONiC** but **not on Cisco**
- Even when both vendors support a path, the **accepted values may differ**
  (e.g., interface naming: `Ethernet1/1` vs `Ethernet0`)
- **Vendor-native paths are completely different** between platforms

This means a "universal config Set" is unrealistic. The agent must be **vendor-aware**
for configuration, even if it uses OpenConfig paths where possible.

### 5.2 🔴 Critical: No Candidate Datastore on Cisco NX-OS

Without a candidate datastore, `Set` applies directly to the running configuration.
There is no way to:
- Preview changes before applying
- Stage multiple changes and commit atomically (beyond a single SetRequest)
- Roll back to a named checkpoint

**Mitigation**: Always Get the current config before Set. Store the "before" snapshot
so we can attempt a manual rollback if the Set causes issues. But this is best-effort.

### 5.3 🟡 High: Latency of Arc Extension Delivery

The Arc extension model is pull-based:
1. ARM enqueues the operation
2. Extension Service on the switch polls (interval: ~minutes)
3. Handler executes
4. Status is reported back

**Total latency: 30 seconds to 5 minutes** from portal click to config applied.

This is acceptable for planned configuration changes but **not suitable for**:
- Emergency interface shutdowns
- Real-time remediation
- Closed-loop automation (detect → react in seconds)

### 5.4 🟡 High: Idempotency and Drift

If a config Set fails mid-way (e.g., network partition), ARM may retry the extension.
The agent must be **idempotent**: applying the same config twice must produce the same
result. gNMI `update` is naturally idempotent, but `replace` and `delete` are not
(deleting something that doesn't exist, or replacing with stale data).

**Drift detection**: The switch config can be changed out-of-band (SSH, console, another
NMS). The agent should be able to detect and report drift — using periodic gNMI `Get` on
`/config` paths and comparing against the desired state stored in Azure.

### 5.5 🟡 High: Authentication and Authorization

- **gNMI auth**: The config agent needs a gNMI user with **write privileges**. Our
  telemetry collector uses read-only credentials. A config agent would need elevated
  credentials — stored in Protected Settings (encrypted by Arc) or Azure Key Vault.
- **RBAC**: Who can trigger a config change from Azure? This needs Azure RBAC integration.
  Today, anyone with `Microsoft.HybridCompute/machines/extensions/write` could invoke
  the extension.
- **Audit trail**: Every config change must be logged: who requested it, what was changed,
  what was the previous value, and whether it succeeded.

### 5.6 🟡 Medium: NX-OS Init.d Limitations

On Cisco NX-OS, there is no systemd. The Arc Extension Service runs under init.d, and
extension handler invocation may behave differently than on a standard Linux host.
We've already seen quirks with the Arc agent on NX-OS (custom RPM, library relocation).
A config agent adds more complexity to this constrained environment.

### 5.7 🟡 Medium: Encoding Mismatch

- **Cisco**: Accepts JSON for Set (not JSON_IETF)
- **SONiC**: Requires JSON_IETF with module prefixes

The agent must normalize config payloads to the correct encoding per vendor, including
adding/stripping YANG module prefixes (e.g., `"openconfig-interfaces:config"` for SONiC
vs `"config"` for Cisco).

---

## 6. Proposed Architecture

```
                        ┌─────────────────────────────┐
                        │      Azure Portal / ARM      │
                        │                              │
                        │  Extension Settings:         │
                        │  {                           │
                        │    "operation": "update",    │
                        │    "paths": [                │
                        │      {                       │
                        │        "path": "/interfaces/ │
                        │          interface[name=Eth1/ │
                        │          1]/config/desc",    │
                        │        "value": "Uplink"     │
                        │      }                       │
                        │    ]                         │
                        │  }                           │
                        └──────────────┬──────────────┘
                                       │ ARM extension delivery
                                       ▼
                     ┌─────────────────────────────────┐
                     │   Arc Extension Service (extd)   │
                     │   on the switch                  │
                     └──────────────┬──────────────────┘
                                    │ invokes handler
                                    ▼
┌───────────────────────────────────────────────────────────────┐
│                  gnmi-config-agent                             │
│                                                               │
│  1. Parse settings JSON (desired config)                      │
│  2. Vendor detection (Cisco / SONiC / Arista)                 │
│  3. gNMI Get current /config state (snapshot for rollback)    │
│  4. Validate: is the Set safe? (vendor-specific checks)       │
│  5. gNMI Set (update/replace/delete)                          │
│  6. gNMI Get to verify the change took effect                 │
│  7. Report status back to Arc Extension Service               │
│                                                               │
│  On failure: attempt rollback using saved snapshot             │
└───────────────────────────────────────────────────────────────┘
         │                                    │
         │ gNMI Set                          │ gNMI Get (verify)
         ▼                                    ▼
   ┌──────────────────────────────────────────────┐
   │         Switch gNMI Server                    │
   │  (running config / config DB)                 │
   └──────────────────────────────────────────────┘
```

### 6.1 Agent Modes

| Mode | Trigger | Use Case |
|------|---------|----------|
| **One-shot** | Arc Extension `enable.sh` | Apply a specific config change from Azure |
| **Drift watch** | Periodic timer (like gnmi-collector) | Continuously compare actual vs desired config |
| **Remediate** | Drift detected | Auto-apply desired config when drift is found |

### 6.2 Reusable Components from gnmi-collector

| Component | Path | Reusable? |
|-----------|------|-----------|
| gNMI client (Get, Subscribe, Capabilities) | `internal/gnmi/client.go` | ✅ Yes — add `Set()` method |
| Config YAML loader | `internal/config/` | ✅ Yes — extend schema |
| Azure Log Analytics logger | `internal/azure/logger.go` | ✅ Yes — for audit logging |
| Vendor-aware encoding | `internal/gnmi/client.go` | ✅ Yes — JSON vs JSON_IETF |
| Transform registry | `internal/transform/` | ❌ Not applicable to config |

---

## 7. What Would a Minimal Proof of Concept Look Like?

### Phase 1: Standalone CLI tool (no Arc integration)

Build `gnmi-config-set` as a CLI tool that:
1. Reads a desired-config YAML file
2. Connects to the switch via gNMI
3. Issues a `SetRequest`
4. Verifies with a subsequent `Get`
5. Prints diff (before vs after)

**Scope**: Interface description changes only (safest, lowest risk).

### Phase 2: Arc Custom Script integration

Wrap the CLI tool with a Custom Script Extension invocation:
- Azure side: `az connectedmachine extension create --type CustomScript`
- Switch side: The script calls `gnmi-config-set` with the desired config

### Phase 3: Dedicated Arc Extension

Publish a proper extension with `HandlerManifest.json`, structured settings,
drift detection, and audit logging to Log Analytics.

---

## 8. Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| Set applies wrong config and disrupts traffic | 🔴 Critical | Always Get-before-Set, store rollback snapshot, start with non-disruptive paths only (descriptions, not shutdown) |
| OpenConfig Set path not implemented on Cisco | 🔴 Critical | Maintain per-vendor allowlist of validated Set paths; fallback to native paths |
| Arc delivery latency too high for operational needs | 🟡 High | Document that this is for planned changes, not emergency response |
| Credential escalation (read-only → read-write) | 🟡 High | Separate credential sets; write credentials in Protected Settings only |
| Config drift between Azure desired state and switch | 🟡 High | Drift watch mode with alerting |
| NX-OS init.d quirks break extension handler | 🟡 Medium | Extensive testing on NX-OS before release |
| Encoding mismatch causes silent config errors | 🟡 Medium | Vendor detection + encoding normalization layer |

---

## 9. Open Questions

1. **Should the config agent be the same binary as gnmi-collector or a separate binary?**
   - Same binary: simpler deployment, shared gNMI client
   - Separate binary: clearer security boundary (read vs write)

2. **What's the desired state source of truth?**
   - Azure Resource Manager (extension settings)?
   - A Git repository (GitOps model)?
   - Azure Policy (compliance-driven)?

3. **How do we handle multi-step configs?** (e.g., create a prefix list, then reference it
   in a route-map, then apply the route-map to a BGP neighbor)
   - Single SetRequest with ordered operations?
   - Sequential extension invocations?

4. **Do we need a "plan" mode?** (show what would change without applying — like Terraform plan)
   - Cisco has no native dry-run; we'd need to simulate by diffing Get vs desired
   - SONiC 4.x has candidate datastore that could support this

5. **Should we support CLI-origin Set?** (send raw CLI commands via gNMI)
   - Arista supports `origin: "cli"` natively
   - Would bypass YANG validation but cover 100% of config surface

---

## 10. Recommendation

**Start with a read-only proof of concept**: build a `gnmi-config-diff` tool that:
1. Takes a desired-config YAML
2. Issues gNMI `Get` on the corresponding `/config` paths
3. Shows the diff between desired and actual

This validates which `/config` paths actually work on each vendor **without any write
risk**. Once we have a validated path inventory, we can proceed to Phase 1 (Set tool).

The Arc Extension integration (Phase 2-3) should only happen after we're confident in
the gNMI Set behavior on both Cisco and SONiC.
