# Adding a New Switch Vendor to gnmi-collector

Guide for onboarding a new network switch vendor to the gNMI telemetry
collector. The architecture uses a self-registration pattern — adding a
new vendor requires NO changes to existing code.

## Prerequisites

- The switch must support gNMI (gRPC Network Management Interface)
- You need to know which YANG models the switch supports (use gNMI
  Capabilities RPC to discover)
- Access to a test switch for validation

## Overview

Adding a new vendor involves three steps:

1. Create transformer files (Go code)
2. Create a YAML config file
3. Test on a real switch

No changes are needed to `collector.go`, `registry.go`, `subscriber.go`,
or any existing transformer file.

## Step 1: Create Transformer Files

Each transformer converts raw gNMI YANG JSON data into the `CommonFields`
output format used by Azure Log Analytics.

### File structure

Create one `.go` file per data type in `src/TelemetryClient/internal/transform/`.
Use a vendor prefix convention (e.g., `arista_` for Arista EOS).

### Self-registration

Every transformer file MUST register itself in an `init()` function:

```go
package transform

func init() {
    Register("eos-bgp-peers", func() Transformer {
        return &EosBgpTransformer{}
    })
}

type EosBgpTransformer struct{}

func (t *EosBgpTransformer) Transform(pathCfg PathConfig, vals map[string]interface{}) ([]CommonFields, error) {
    // Extract fields from the YANG JSON response
    // Return CommonFields with data_type, timestamp, date, message
    entries := []CommonFields{{
        DataType:  "arista_bgp_summary",
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Date:      time.Now().UTC().Format("2006-01-02"),
        Message:   map[string]interface{}{
            "neighbor_id": GetString(vals, "neighbor-address"),
            "state":       GetString(vals, "session-state"),
            // ... more fields
        },
    }}
    return entries, nil
}
```

### Registration name convention

The registration name (e.g., `"eos-bgp-peers"`) must match the `name`
field in the YAML config's `paths` section. Use a vendor prefix for
vendor-specific paths:

- OpenConfig paths (shared): `interface-counters`, `bgp-neighbors`, etc.
- Vendor-specific paths: `eos-*`, `junos-*`, `nx-*`

If an OpenConfig transformer already handles the path (e.g.,
`interface-counters`), you can reuse it — no new transformer needed.

### Helper functions

Use these helpers from `transform/helpers.go`:
- `GetString(vals, "key")` — safely get a string value
- `GetMap(vals, "key")` — safely get a nested map
- `GetInt64(vals, "key")` — safely get an int64
- `GetFloat64(vals, "key")` — safely get a float64
- `GetArray(vals, "key")` — safely get an array

## Step 2: Create a YAML Config

Create `config.<vendor>.yaml` in `src/TelemetryClient/`:

```yaml
target:
  address: 127.0.0.1
  port: 50051                # Vendor's gNMI port
  tls:
    enabled: true
    skip_verify: true
  credentials:
    username_env: GNMI_USER
    password_env: GNMI_PASS

collection:
  mode: poll                 # poll or subscribe
  interval: 300s
  timeout: 30s
  encoding: JSON_IETF       # JSON or JSON_IETF (vendor-dependent)

azure:
  workspace_id_env: WORKSPACE_ID
  primary_key_env: PRIMARY_KEY
  device_type: arista-eos    # REQUIRED — controls data_type prefix

paths:
  - name: interface-counters           # Reuses existing OpenConfig transformer
    yang_path: /openconfig-interfaces:interfaces/interface/state/counters
    table: AristaInterfaceCounter_CL
    enabled: true
    mode: sample
    sample_interval: 60s

  - name: eos-bgp-peers               # Uses new vendor-specific transformer
    yang_path: /arista/bgp/peers       # Vendor-specific YANG path
    table: AristaBgpSummary_CL
    enabled: true
    mode: sample
    sample_interval: 300s
```

### Key config fields

| Field | Purpose | Example |
|-------|---------|---------|
| `device_type` | Controls `data_type` prefix in output. REQUIRED. | `arista-eos` → `arista_eos_*` |
| `encoding` | JSON or JSON_IETF (check vendor docs) | `JSON_IETF` |
| `paths[].name` | Must match a registered transformer name | `interface-counters` |
| `paths[].yang_path` | Full YANG path with module prefix | `/openconfig-interfaces:...` |
| `paths[].table` | Azure Log Analytics table name | `AristaInterfaceCounter_CL` |

## Step 3: Test

### Discovery — check supported YANG models

```bash
./gnmi-collector --config config.arista.yaml --capabilities
```

### Dry-run — validate transformers produce correct output

```bash
./gnmi-collector --config config.arista.yaml --dry-run --once
```

This prints all transformed entries to stdout without sending to Azure.
Verify:
- Correct `data_type` prefix (e.g., `arista_eos_interface_counters`)
- All expected fields populated in `message`
- Entry counts match expectations (e.g., one entry per interface)

### Subscribe mode validation

```bash
# Change mode to subscribe in config, then:
./gnmi-collector --config config.arista.yaml --dry-run --once
```

Verify subscribe produces same output as poll mode.

## Current Transformer Registry

All registered transformers (25 total):

| Name | File | Vendor | Type |
|------|------|--------|------|
| interface-counters | interface_counters.go | Shared | OpenConfig |
| interface-status | interface_status.go | Shared | OpenConfig |
| if-ethernet | interface_ethernet.go | Shared | OpenConfig |
| bgp-neighbors | bgp_summary.go | Shared | OpenConfig |
| bgp-global | bgp_global.go | Shared | OpenConfig |
| lldp-neighbors | lldp_neighbor.go | Shared | OpenConfig |
| mac-table | mac_address.go | Shared | OpenConfig |
| arp-table | arp.go | Shared | OpenConfig |
| temperature | environment.go | Shared | OpenConfig |
| power-supply | environment.go | Shared | OpenConfig |
| platform-inventory | inventory.go | Shared | OpenConfig |
| transceiver | transceiver.go | Shared | OpenConfig |
| transceiver-channel | transceiver_channel.go | Shared | OpenConfig |
| system-cpus | system.go | Shared | OpenConfig |
| system-memory | system.go | Shared | OpenConfig |
| system-state | system.go | Shared | OpenConfig |
| nx-sys-cpu | native_system.go | Cisco | Native |
| nx-sys-memory | native_system.go | Cisco | Native |
| nx-arp | native_arp.go | Cisco | Native |
| nx-bgp-peers | native_bgp.go | Cisco | Native |
| nx-env-sensor | native_environment.go | Cisco | Native |
| nx-env-psu | native_environment.go | Cisco | Native |
| nx-lldp | native_lldp.go | Cisco | Native |
| nx-mac-table | native_mac.go | Cisco | Native |
| nx-transceiver | native_transceiver.go | Cisco | Native |

OpenConfig ("Shared") transformers work with any vendor that supports
standard OpenConfig YANG models — you only need new transformers for
vendor-specific paths.
