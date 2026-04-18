# gNMI Collector — Progress Log

Challenges encountered during the gNMI/YANG telemetry migration and how
they were resolved. Serves as troubleshooting reference for future work.

---

## 1. Subscribe Mode — Seven Bugs Across Three Layers

### Layer 1: Notification Prefix (v1)

**Problem**: `DecodeSubscribeResponse()` discarded the gNMI notification
prefix, which contains entity keys (e.g., `[name=Ethernet0]`). Paths came
out empty, breaking interface name extraction.

**Fix**: Created `DecodeSubscribeResponseWithPrefix()` that preserves the
prefix and exported `NormalizeSubscribeNotifications()` to convert
leaf-level updates into tree maps consumable by transformers.

**Files**: `gnmi/client.go`, `collector/subscriber.go`

---

### Layer 2: Cisco Tree-Level Responses (v3–v4)

**Problem**: NX-OS subscribe STREAM returns entire subtrees at root level.
The path is `/interfaces` with the value containing the full tree:
`{interface: [{name: eth1/1, state: {counters: {...}}}]}`. Transformers
expected data scoped to the subscribed path level, not the root.

**Fix**: `drillDownToSubscribedPath()` navigates nested maps by following
YANG path segments. For list paths (arrays), it creates per-entity
notifications with proper `[name=X]` key selectors.

**Example**:
- Input: `path=/interfaces`, `value={interface: [...]}`
- Output: 66 notifications at `path=/interfaces/interface[name=eth1/1]/state/counters`

**Files**: `collector/subscriber.go`

---

### Layer 3: SONiC-Specific Issues (v5–v6)

**Key selectors in paths**: SONiC sends paths WITH YANG list keys
(e.g., `/interfaces/interface[name=Ethernet14]/state/counters`). Path
comparison against the subscribed path failed because the subscribed path
has no keys.
**Fix**: `stripKeySelectors()` removes `[key=value]` segments before
comparison.

**CPU array overwrite**: `setNestedValue()` stripped list keys
(`cpu[index=0]` → `cpu`), causing all CPU indices to overwrite each other
in the output map.
**Fix**: Made array-aware with `parseListKeySelector()` and
`findArrayElement()` to preserve per-index identity.

**Below-path wrapping**: Subscribe sends data at `/memory/state` (below
subscribed `/memory`). The value arrived unwrapped.
**Fix**: `wrapValueInPath()` wraps the value in the remaining path
segments so it matches the structure transformers expect.

**Files**: `gnmi/client.go`, `collector/subscriber.go`

---

### Layer 3b: Transformer Compatibility (v7)

**data_type prefix**: `applyDataTypePrefix()` was only called in poll
mode, not subscribe. SONiC showed `cisco_nexus_*` data_type values.
**Fix**: Added the call in the subscribe handler path.

**toFloat() types**: Only handled `float64` and `int`, but gNMI subscribe
sends `int64`/`uint64` from protobuf. CPU averages showed "0.0".
**Fix**: Expanded type switch to cover `int64`, `uint64`, `int32`,
`uint32`.

**getCpuId()**: Array-aware normalization puts the list key `index` at
the CPU entry level, not inside `state`. cpuid came back empty.
**Fix**: Check the entry level first, fall back to `state` level.

**Files**: `transform/system.go`, `collector/subscriber.go`

---

**Result**: 100% data parity between poll and subscribe on both platforms.

---

## 2. SONiC-Specific Challenges

**JSON_IETF encoding**: SONiC only supports JSON_IETF (RFC 7951), which
adds module prefixes to all top-level keys and cross-module augmentation
keys (e.g., `openconfig-if-ethernet:ethernet`). Fixed with
`stripModulePrefixes()` in the gNMI client — recursive key stripping at
decode time. Single-point fix covering all 15 transformers.

**Bulk Get returns empty**: SONiC returns `{}` for Get on list paths
without entity keys. Fixed with Subscribe ONCE fallback in the collector —
when Get returns empty or an error, automatically retries with Subscribe
ONCE (one-shot stream that delivers current state then closes).

**BGP requires VRF keys**: Unlike Cisco, SONiC requires explicit
`[name=default]` and `[identifier=BGP][name=bgp]` keys in BGP paths.
Fixed in `config.sonic.yaml` path definitions.

**Base64-encoded float32 values**: Power supply sensor values arrive as
base64-encoded IEEE 754 float32 strings. Fixed with
`DecodeBase64Float32()` helper in the environment transformer.

**Counter values as strings**: SONiC sends counter values as quoted
strings (`"0"` not `0`). Fixed with `toFloat()` type conversion handling
string inputs across transformers.

**Minimum subscribe interval**: SONiC enforces a 30-second minimum for
subscribe sample interval (server-side rejection below that).

---

## 3. Cisco-Specific Challenges

**VRF split**: The gNMI server runs in the management VRF, but Azure
endpoints are reachable only from the default VRF. Fixed with
`grpc use-vrf default` — runs gNMI server on both VRFs simultaneously.
Needs security team approval for production deployment.

**bootflash is noexec**: SCP uploads land on `/bootflash/`, which is
mounted noexec. Binaries must be copied to an executable path:
```
cp /bootflash/gnmi-collector /tmp/ && chmod +x /tmp/gnmi-collector
```
> **Note**: This was a development workaround. Production deployment uses
> `/opt/gnmi-collector/` (installed via setup scripts), which handles
> permissions correctly.

**SCP requires legacy protocol**: NX-OS lacks sftp support. Must use
`scp -O` (legacy SCP protocol) for file transfers.

**TLS cert expires in 1 day**: NX-OS auto-generates a self-signed gRPC
certificate valid for only 24 hours. Manually generated a longer-lived
certificate (825 days) via openssl + PKCS12 import into the switch
trustpoint. Needs automated rotation for production.

**SSH keyboard-interactive auth**: NX-OS uses keyboard-interactive (not
password) authentication. Handled via the `SSH_ASKPASS` mechanism in
automation scripts.

---

## 4. Architecture Refactoring

### Problems Identified

1. **Hardcoded transformer registry** in `collector.go` — 25 transformers
   listed in a literal map. Adding a vendor required modifying this
   central file.
2. **Cisco-as-default bias** — `device_type` defaulted to
   `"cisco-nx-os"` when omitted, masking configuration errors.
3. **Poll/subscribe asymmetry** — `mergeByDataType()` only called in poll
   mode, causing subscribe to emit un-merged batches.

### Solutions Applied (Option B — light refactor, zero functional risk)

1. **Self-registration pattern**: `transform/registry.go` provides
   `Register(name, factory)` called from `init()` in each transformer
   file. `BuildMap()` returns the complete map at startup. New vendors
   just add files — no central edits.
2. **`device_type` now required** — returns a clear error if empty,
   preventing silent misconfiguration.
3. **`flushBatch()` calls `mergeByDataType()`** — subscribe and poll now
   use identical merge logic.

### Verification

55+ unit tests pass, `go vet` clean, refactor-v1 binary deployed and
verified on both switches with zero regressions.

---

## 5. Known Issues / Remaining Work

| Issue | Status | Notes |
|---|---|---|
| TLS cert lifecycle (Cisco) | ⚠️ Manual | 825-day cert in place; needs automated rotation |
| Security approval for `grpc use-vrf default` | 🔲 Pending | Required before production deployment |
| Production table rename | ✅ Done | `GnmiTest*` → `Cisco*_CL` production table names |
| Credential management | 🔲 Pending | Move beyond env vars to vault/managed identity |
| ~15 fields from Linux /proc | ❌ Not available | load_avg, vmalloc, processes — not in YANG |
| Dell OS10 gNMI | ❌ Not supported | Requires SFD mode; no timeline |
| Arc onboarding automation | ✅ Skill created | `.github/skills/arc-onboarding/`; DCF still manual |

---

## 6. Tooling — Arc Onboarding Skill & SSH Refactoring

### Shared SSH Module

Extracted ~350 lines of duplicated SSH/SCP code from 5 skill scripts into
`.github/skills/ssh-helpers.ps1` (232 lines, 7 functions):

- `Find-SshBinary` / `Find-ScpBinary` — locate binaries on Windows/Linux
- `New-AskPassFile` / `Remove-AskPassFile` — temporary ASKPASS helper
- `Remove-SshNoise` — strip common SSH noise (warnings, MOTD)
- `Invoke-SshCommand` / `Send-ScpFile` — parameterized SSH/SCP wrappers

All 5 existing skill scripts (Cisco/SONiC SSH + SCP, onboarding) were
refactored to import this module. Total line count dropped by ~40%.

**Bug found**: PowerShell's `$Host` is a read-only automatic variable.
Using `-Host` as a function parameter silently fails in some contexts.
Renamed to `-HostName` across all 7 files.

### Arc Onboarding Skill

Created `.github/skills/arc-onboarding/` with `SKILL.md` + `onboard-switch.ps1`.

**8-step automated flow:**
1. Resolve SSH password (Key Vault or env var)
2. SSH in and auto-detect vendor (NX-OS vs SONiC)
3. Fetch switch hostname
4. Resolve Azure parameters (subscription, tenant, resource group, region)
5. Generate setup script from template (fills 10+ placeholders)
6. Upload and execute setup script on the switch
7. Validate gNMI collector + Arc agent installed
8. **Display `azcmagent connect` command** — user completes DCF in browser

**Known limitation**: `azcmagent connect` requires Device Code Flow (DCF)
— interactive browser sign-in. The skill pauses and instructs the user.
Future improvement: replace with `--service-principal-id` for full automation.

### wget → curl Migration

All 4 setup scripts (`Arcnet_Cisco_gNMI_Setup`, `Arcnet_Sonic_gNMI_Setup`,
`Arcnet_Cisco_Arc_Setup`, `Arcnet_Dell_Setup`) were updated to use `curl`
instead of `wget`, which is not available on NX-OS or SONiC switches.

> **WARNING**: The Azure Portal generates onboarding scripts that use `wget`.
> If you copy the Portal-generated script directly, it will fail on switches.
> See `README.md` for the curl-based version.
