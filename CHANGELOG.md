# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- **gNMI Telemetry Collector** — single Go binary that collects structured YANG
  data via gNMI and ships it to Azure Log Analytics. Replaces the cron-based CLI
  text scraping pipeline.
  - Cisco NX-OS support: 21 YANG paths (OpenConfig + Cisco native), ~97% parity
    with legacy CLI parser.
  - Dell Enterprise SONiC support: 16 YANG paths (OpenConfig + SONiC native),
    full environment coverage (temperature, PSU, fan).
  - Poll mode (periodic gNMI Get) — production-ready.
  - Subscribe mode (persistent gNMI stream) — supports both `sample` and
    `on_change` per-path modes. Validated on Cisco NX-OS and SONiC.
  - CLI flags: `--once`, `--dry-run`, `--output`, `--dump`, `--verbose`, `--version`.
- **Per-vendor configuration files**: `config.cisco.yaml` (21 paths) and
  `config.sonic.yaml` (16 paths) ship in the release tarball.
- **init.d service script** (`gnmi-collectord`) for Cisco NX-OS daemon management.
- **Automated onboarding Copilot skill** (`.github/skills/arc-onboarding/`) —
  SSH into a switch, auto-detect vendor, fill setup script, and execute.
- **Per-vendor data parity reports** documenting field-by-field coverage:
  - `cisco-nxos-parity.md` — CLI vs gNMI comparison (18 categories)
  - `sonic-gnmi-parity.md` — SONiC gNMI coverage (19 categories)
  - `data-parity-overview.md` — cross-vendor summary matrix
- **Customer-facing installation guide** (`Docs/arcnet_onboarding_instructions/README.md`)
  with platform decision tree and step-by-step instructions for all 4 paths:
  Cisco gNMI, Cisco CLI (legacy), SONiC gNMI, Dell OS10 CLI.
- **Adding new vendor guide** (`adding_new_vendor.md`) with self-registration
  pattern for transformers.
- **Build workflow** (`build-gnmi-collector.yml`) — builds, tests, packages, and
  creates GitHub releases with version scheme `gnmi-MAJOR.YYMM.INCREMENT`.

### Changed
- Renamed `config.example.yaml` → `config.cisco.yaml` for clarity.
- **Unified telemetry schema** — vendor-prefixed table names (`CiscoBgp*`, `SonicBgp*`)
  replaced with shared tables (`BgpNeighbor_CL`, `EnvTemperature_CL`, etc.). Both
  platforms write to the same tables; `device_type` distinguishes rows.
- **Normalized environment fields** — temperature, power supply, and fan transformers
  now emit consistent field names across Cisco and SONiC (e.g., `high_threshold`,
  `input_voltage`, `ps_name`).
- **Added Cisco fan transformer** (`nx-fan`) — Cisco NX-OS now reports fan health
  to `EnvFan_CL` alongside SONiC, closing the last environment parity gap.
- Root `README.md` rewritten to lead with Azure Arc + gNMI capabilities.
- Onboarding README restructured with per-platform installation sections.

### Fixed
- Bounds check on workspace ID display in `main.go` (prevents panic on short IDs).
- `json.Unmarshal` errors in `collector.go` now logged instead of silently ignored.

### Deprecated
- **Cisco CLI parser path** (`Arcnet_Cisco_Arc_Setup`) — use gNMI path instead.
  Will be removed in a future release.

### Known Limitations
- **Subscribe mode on SONiC**: Not all paths support `on_change` — paths backed
  by COUNTERS DB (e.g., interface-counters) are rejected. Use `sample` mode for
  those paths. See per-path comments in `config.sonic.yaml`.
- **SONiC gaps**: Interface errors, transceiver DOM, and route summary not yet
  available (~24% less coverage than Cisco). See `sonic-gnmi-parity.md`.
- **Azure Arc on NX-OS**: Requires repackaged RPM (`v0.0.2-alpha-rpm`) because
  the standard installer depends on systemd. The RPM tag reflects the packaging
  process; the agent itself is the standard Microsoft release.
- **Device Code Flow**: `azcmagent connect` requires interactive browser
  authentication. Service principal auth is planned but not yet implemented.

## [1.2602.2] - 2025-02-19

### CLI Parsers
- Unified parser architecture — single binary per vendor.
- Cisco Nexus: 14 parsers (bgp-all-summary, class-map, environment-power,
  interface-counters, interface-error-counters, interface-status, inventory,
  ip-arp, ip-route, lldp-neighbor, mac-address, system-uptime, transceiver,
  version).
- Dell OS10: 4 parsers (interface, interface-phy, lldp, version).
