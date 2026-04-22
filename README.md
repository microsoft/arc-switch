# Arc-Switch

**Azure Arc management and gNMI telemetry for network switches.**

Arc-Switch enables you to onboard network switches to [Azure Arc](https://learn.microsoft.com/en-us/azure/azure-arc/overview) and stream structured telemetry to [Azure Log Analytics](https://learn.microsoft.com/en-us/azure/azure-monitor/logs/log-analytics-overview) using gNMI/YANG — replacing legacy CLI text scraping with version-resilient, structured data collection.

## Supported Platforms

| Platform | Telemetry | Method | Status |
|----------|-----------|--------|--------|
| **Cisco NX-OS** | gNMI *(recommended)* | 21 YANG paths via gRPC | ✅ Production |
| **Cisco NX-OS** | CLI parser *(legacy)* | `vsh` text scraping + cron | ⚠️ Deprecated |
| **Dell Enterprise SONiC** | gNMI | 16 YANG paths via gRPC | ✅ Production |
| **Dell OS10** | CLI parser | `clish` text scraping + cron | ✅ Production |

## Quick Start

See the **[Installation Guide](Docs/arcnet_onboarding_instructions/README.md)** for complete step-by-step instructions. The guide covers:

1. Azure prerequisites (subscription, resource group, Log Analytics workspace)
2. Platform selection — choose the right setup script for your switch
3. Step-by-step installation with copy-paste commands
4. Azure Arc connection (Device Code Flow)
5. Verification and troubleshooting

## Repository Structure

```
arc-switch/
├── src/
│   ├── TelemetryClient/              # gNMI telemetry collector (Go)
│   │   ├── cmd/gnmi-collector/       # Main binary entry point
│   │   ├── internal/                 # Collector, transformers, Azure logger
│   │   ├── config.cisco.yaml         # Cisco NX-OS config (21 paths)
│   │   └── config.sonic.yaml         # SONiC config (16 paths)
│   └── SwitchOutput/                 # Legacy CLI parsers
│       ├── Cisco/Nexus/10/           # Cisco unified parser (Go)
│       └── DellOS/10/               # Dell OS10 unified parser (Go)
├── Docs/
│   ├── arcnet_onboarding_instructions/  # Installation guide + setup scripts
│   └── telemetry-improvement-plan/      # Data parity reports per vendor
├── .github/
│   ├── workflows/                    # CI/CD: build + release workflows
│   └── skills/                       # Copilot skills for automated onboarding
└── tools/                            # Dev tools (SSH helpers, gnmi-probe)
```

## Architecture

```
Network Switch (Cisco NX-OS or Dell SONiC)
    ├─> gNMI Get (gRPC, structured YANG data)
    ├─> gnmi-collector (transforms + ships in one binary)
    └─> Azure Log Analytics Workspace
            └─> Grafana / Azure Dashboards
```

The gNMI collector runs on the switch itself as a long-running service (init.d on NX-OS, systemd on SONiC). It periodically queries the local gNMI server, transforms responses into flat JSON using YANG-aware transformers, and POSTs them to Azure Log Analytics.

## Documentation

| Document | Description |
|----------|-------------|
| [Installation Guide](Docs/arcnet_onboarding_instructions/README.md) | Customer-facing setup instructions for all platforms |
| [Data Parity Overview](Docs/telemetry-improvement-plan/data-parity-overview.md) | Cross-vendor telemetry coverage matrix |
| [Cisco NX-OS Parity](Docs/telemetry-improvement-plan/cisco-nxos-parity.md) | CLI vs gNMI field-by-field comparison |
| [SONiC Parity](Docs/telemetry-improvement-plan/sonic-gnmi-parity.md) | SONiC gNMI coverage and known gaps |
| [Design Document](Docs/telemetry-improvement-plan/design.md) | Architecture, transformer registry, platform support |
| [Adding a New Vendor](Docs/arcnet_onboarding_instructions/adding_new_vendor.md) | How to add support for a new switch platform |

## Releases

Visit the [Releases page](https://github.com/microsoft/arc-switch/releases) for pre-compiled binaries.

- **gNMI Collector**: `gnmi-collector-VERSION-linux-amd64.tar.gz` — single binary for Cisco and SONiC
- **Cisco CLI Parser**: `cisco-nexus-unified-parser-VERSION-linux-amd64.tar.gz` — legacy CLI text parser
- **Dell CLI Parser**: `dell-os10-unified-parser-VERSION-linux-amd64.tar.gz` — Dell OS10 text parser

## Development

```bash
# Build gNMI collector
cd src/TelemetryClient
go build -o gnmi-collector ./cmd/gnmi-collector/

# Run tests
go test ./...

# Test locally with dry-run
./gnmi-collector --config config.cisco.yaml --once --dry-run
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.
