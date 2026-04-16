# Azure Arc for Network Switches

## Overview

This project enables Azure Arc management and monitoring for network switches. By onboarding switches to Azure Arc, you can:

- **Centralized Management**: Manage network devices alongside other Azure resources
- **Monitoring & Telemetry**: Collect and analyze switch telemetry in Azure Log Analytics
- **Dashboards & Alerts**: Create custom dashboards and alerts in Azure or Grafana
- **Compliance & Governance**: Apply Azure policies and track configuration changes
- **Hybrid Infrastructure**: Unified view of cloud and on-premises network devices

## Supported Platforms

| Platform | Telemetry Method | Setup Script | Status |
|----------|-----------------|-------------|--------|
| **Cisco NX-OS** | gNMI *(recommended)* | [`Arcnet_Cisco_gNMI_Setup`](Arcnet_Cisco_gNMI_Setup) | ✅ Production |
| **Cisco NX-OS** | CLI parser *(legacy)* | [`Arcnet_Cisco_Arc_Setup`](Arcnet_Cisco_Arc_Setup) | ⚠️ Deprecated |
| **Dell Enterprise SONiC** | gNMI | [`Arcnet_Sonic_gNMI_Setup`](Arcnet_Sonic_gNMI_Setup) | ✅ Production |
| **Dell OS10** | CLI parser | [`Arcnet_Dell_Setup`](Arcnet_Dell_Setup) | ✅ Production |

### Which path should I choose?

```
What platform is the switch running?
├── Cisco NX-OS ──────────> Use Arcnet_Cisco_gNMI_Setup (recommended)
│                           Or Arcnet_Cisco_Arc_Setup (legacy CLI, being phased out)
├── Dell Enterprise SONiC ─> Use Arcnet_Sonic_gNMI_Setup
└── Dell OS10 ─────────────> Use Arcnet_Dell_Setup
```

> **Note**: The legacy CLI parser path for Cisco (`Arcnet_Cisco_Arc_Setup`) is
> deprecated. All new onboardings should use the gNMI path. The CLI path will
> be removed in a future release once all existing switches are migrated.

### ⚠️ CLI Parser and gNMI Collector Are Mutually Exclusive (Cisco Only)

Cisco NX-OS is the only platform with both a CLI parser and gNMI collector path.
Each Cisco setup script includes a **pre-flight check** that detects if the other
collector type is already installed. If a conflict is found, the script prints
specific uninstall instructions and exits. You must remove the existing collector
before installing the other.

If you need to migrate from CLI parser to gNMI (recommended), see
[Uninstalling a Collector](#uninstalling-a-collector) below.

> Dell OS10 only supports CLI parser. SONiC only supports gNMI. No conflict
> checks are needed for these platforms.

---

## Prerequisites

### Azure Requirements

Before installing on any platform, you need:

1. **Azure Subscription** with permissions to create resources
2. **Resource Group** for Arc-enabled devices (e.g., `ARCNET`)
3. **Log Analytics Workspace** for telemetry storage

#### Create a Log Analytics Workspace

```bash
# Using Azure CLI
az monitor log-analytics workspace create \
  --resource-group "ARCNET" \
  --workspace-name "SwitchTelemetry" \
  --location "eastus"
```

Or via the Azure Portal: Create a resource → Search "Log Analytics Workspace" → fill in details.

#### Get Workspace Credentials

You'll need three values:

```bash
# Workspace ID
az monitor log-analytics workspace show \
  --resource-group "ARCNET" \
  --workspace-name "SwitchTelemetry" \
  --query "customerId" -o tsv

# Primary and Secondary Keys
az monitor log-analytics workspace get-shared-keys \
  --resource-group "ARCNET" \
  --workspace-name "SwitchTelemetry"
```

### Network Requirements

The switch must have outbound HTTPS connectivity to:
- Azure Arc services: `*.his.arc.azure.com`
- Azure Log Analytics: `*.ods.opinsights.azure.com`
- GitHub Releases: `github.com` (for binary downloads during setup)

---

## Installation

Each platform has a self-contained setup script. The workflow is the same for all:

1. **Edit** the configuration variables at the top of the script
2. **SSH** into the switch and become root
3. **Paste** the script into the bash shell
4. **Connect** to Azure Arc (Device Code Flow authentication)
5. **Verify** telemetry is flowing to Log Analytics

### Platform-Specific Instructions

- [**Cisco NX-OS (gNMI)**](#cisco-nx-os-gnmi-recommended) — Recommended for all new Cisco deployments
- [**Cisco NX-OS (CLI)**](#cisco-nx-os-cli-legacy) — Legacy, for existing deployments only
- [**Dell Enterprise SONiC (gNMI)**](#dell-enterprise-sonic-gnmi) — Only option for SONiC
- [**Dell OS10 (CLI)**](#dell-os10-cli) — Only option for Dell OS10

---

### Cisco NX-OS — gNMI (Recommended)

**Script**: [`Arcnet_Cisco_gNMI_Setup`](Arcnet_Cisco_gNMI_Setup)

**What it installs**: Azure Arc agent (via RPM) + gNMI telemetry collector + init.d services

> **Note on Azure Arc agent RPM**: The setup script uses an Arc agent RPM
> packaged at tag `v0.0.2-alpha-rpm`. Despite the tag name, this is the
> validated and tested version (`azcmagent 1.54.03131`) for NX-OS switches.
> The "alpha" label reflects the packaging process, not the agent stability —
> the agent itself is the standard Microsoft Azure Arc release. NX-OS requires
> a repackaged RPM because the standard installer (`install_linux_azcmagent.sh`)
> depends on `systemd` and `apt`/`dnf`, which are not available on NX-OS.

**Platform requirements**:
- NX-OS 9.3(x) or later
- `feature grpc` enabled on the switch
- `grpc use-vrf default` configured (for single-VRF deployment)

#### Step 1: Enable gNMI on the Switch

```
! On the NX-OS CLI (not bash)
configure terminal
feature grpc
grpc use-vrf default
end
```

#### Step 2: Edit the Setup Script

Open `Arcnet_Cisco_gNMI_Setup` and fill in the configuration variables at the top:

```bash
# Azure Configuration
REGION="eastus"
RESOURCE_GROUP="ARCNET"
SUBSCRIPTION_ID="<YOUR_SUBSCRIPTION_ID>"
MACHINE_NAME="<YOUR_SWITCH_HOSTNAME>"
TENANT_ID="<YOUR_TENANT_ID>"

# Log Analytics Workspace
WORKSPACE_ID="<FROM_PREREQUISITES>"
PRIMARY_KEY="<FROM_PREREQUISITES>"
SECONDARY_KEY="<FROM_PREREQUISITES>"

# gNMI Credentials (the switch login credentials)
GNMI_USER="<NX-OS_USERNAME>"
GNMI_PASS="<NX-OS_PASSWORD>"
```

#### Step 3: Run the Script

```bash
# SSH into the switch
ssh admin@<switch-ip>

# Enter bash and become root
run bash
sudo su -

# Paste the entire script contents
```

The script will:
- Install the Arc agent RPM (with NX-OS relocation handling)
- Configure Arc services (HIMDS, ArcProxy, EXTD, GCAD) as init.d daemons
- Download and install the gNMI collector
- Create the collector configuration and environment file
- Install the collector as an init.d service

#### Step 4: Connect to Azure Arc

After the script completes, run the `azcmagent connect` command displayed by the script:

```bash
azcmagent connect \
  --resource-group "ARCNET" \
  --tenant-id "<YOUR_TENANT_ID>" \
  --location "eastus" \
  --subscription-id "<YOUR_SUBSCRIPTION_ID>" \
  --cloud "AzureCloud"
```

This will display a URL and code for Device Code Flow authentication:
1. Open the URL in a browser on another device
2. Enter the code when prompted
3. Sign in with your Azure credentials
4. Return to the switch terminal

#### Step 5: Verify

```bash
# Arc agent
azcmagent show         # Should show "Agent Status: Connected"

# Arc services
for svc in himdsd arcproxyd extd gcad; do /etc/init.d/$svc status; done

# gNMI collector
/etc/init.d/gnmi-collectord status
# Or: ps aux | grep gnmi-collector

# Dry-run test (prints telemetry to stdout instead of sending)
/opt/gnmi-collector/gnmi-collector --config /opt/gnmi-collector/config.yaml --once --dry-run
```

**Telemetry paths collected** (20 paths): Interface counters, interface status, interface Ethernet, interface errors, BGP neighbors, BGP summary, LLDP, transceiver DOM, environment temperature, environment power, environment fan, system resources, system uptime, system version, ARP, MAC addresses, IP routes, route summary, inventory, platform components.

---

### Cisco NX-OS — CLI (Legacy)

**Script**: [`Arcnet_Cisco_Arc_Setup`](Arcnet_Cisco_Arc_Setup)

> ⚠️ **Deprecated**: This path uses CLI text scraping and is being phased out.
> Use the gNMI path above for all new deployments.

**What it installs**: Azure Arc agent (via RPM) + Cisco CLI parser + cron job + Azure logger shell script

#### Step 1: Edit the Setup Script

Open `Arcnet_Cisco_Arc_Setup` and fill in the configuration variables:

```bash
REGION="eastus"
RESOURCE_GROUP="ARCNET"
SUBSCRIPTION_ID="<YOUR_SUBSCRIPTION_ID>"
MACHINE_NAME="<YOUR_SWITCH_HOSTNAME>"
TENANT_ID="<YOUR_TENANT_ID>"
WORKSPACE_ID="<FROM_PREREQUISITES>"
PRIMARY_KEY="<FROM_PREREQUISITES>"
SECONDARY_KEY="<FROM_PREREQUISITES>"
```

#### Step 2: Run the Script

Same as gNMI — SSH in, become root, paste the script.

#### Step 3: Connect to Azure Arc

Same Device Code Flow process as described above.

#### Step 4: Verify

```bash
# Arc agent
azcmagent show

# Cron job
crontab -l | grep cisco-parser-collector

# Manual test
/opt/cisco-parser-collector.sh

# Check logs
tail -f /var/log/cisco-parser-collector.log
```

---

### Dell Enterprise SONiC — gNMI

**Script**: [`Arcnet_Sonic_gNMI_Setup`](Arcnet_Sonic_gNMI_Setup)

**What it installs**: Azure Arc agent (via standard installer) + gNMI telemetry collector + systemd service

**Platform requirements**:
- Dell Enterprise SONiC with `sonic-gnmi` container running
- gNMI server listening on port 8080

#### Step 1: Verify gNMI is Available

```bash
# Check that the sonic-gnmi container is running
docker ps | grep gnmi
```

#### Step 2: Edit the Setup Script

Open `Arcnet_Sonic_gNMI_Setup` and fill in the configuration variables:

```bash
# Azure Configuration
REGION="eastus"
RESOURCE_GROUP="ARCNET"
SUBSCRIPTION_ID="<YOUR_SUBSCRIPTION_ID>"
MACHINE_NAME="<YOUR_SWITCH_HOSTNAME>"
TENANT_ID="<YOUR_TENANT_ID>"

# Log Analytics Workspace
WORKSPACE_ID="<FROM_PREREQUISITES>"
PRIMARY_KEY="<FROM_PREREQUISITES>"
SECONDARY_KEY="<FROM_PREREQUISITES>"

# gNMI Credentials (the SONiC admin credentials)
GNMI_USER="admin"
GNMI_PASS="<SONIC_ADMIN_PASSWORD>"
```

#### Step 3: Run the Script

```bash
# SSH into the switch
ssh admin@<switch-ip>

# Become root
sudo su -

# Paste the entire script contents
```

The script will:
- Install the Arc agent via the standard Linux installer
- Download and install the gNMI collector
- Create a systemd service for the collector
- Validate gNMI connectivity

#### Step 4: Connect to Azure Arc

```bash
azcmagent connect \
  --resource-group "ARCNET" \
  --tenant-id "<YOUR_TENANT_ID>" \
  --location "eastus" \
  --subscription-id "<YOUR_SUBSCRIPTION_ID>" \
  --cloud "AzureCloud"
```

Complete Device Code Flow authentication as described above.

#### Step 5: Verify

```bash
# Arc agent
azcmagent show

# gNMI collector (systemd)
systemctl status gnmi-collector

# Dry-run test
/opt/gnmi-collector/gnmi-collector --config /opt/gnmi-collector/config.yaml --once --dry-run
```

**Telemetry paths collected** (14 paths): Interface counters, interface status, interface Ethernet, BGP neighbors, BGP summary, LLDP, environment (platform inventory: temperature, PSU, fans), system resources, system uptime, ARP, MAC addresses, IP routes.

---

### Dell OS10 — CLI

**Script**: [`Arcnet_Dell_Setup`](Arcnet_Dell_Setup)

**What it installs**: Dell CLI parser + cron job + Azure logger

> **Note**: Dell OS10 does not include Azure Arc onboarding. The setup script
> installs only the telemetry parser and logging pipeline. gNMI is not
> available on Dell OS10 in production mode (requires SmartFabric Director mode).

#### Step 1: Edit the Setup Script

Open `Arcnet_Dell_Setup` and fill in:

```bash
WORKSPACE_ID="<FROM_PREREQUISITES>"
PRIMARY_KEY="<FROM_PREREQUISITES>"
SECONDARY_KEY="<FROM_PREREQUISITES>"
```

#### Step 2: Run the Script

```bash
# SSH into the switch
ssh admin@<switch-ip>

# Become root
sudo su -

# Paste the entire script contents
```

#### Step 3: Verify

```bash
# Check cron job
crontab -l | grep dell-parser

# Manual test
/opt/dell-parser-collector.sh

# Check logs
tail -f /var/log/dell-parser-collector.log
```

---

## Verify Data in Log Analytics

After installation, wait 5–10 minutes for data to appear. Then query your workspace:

1. Go to **Azure Portal** → **Log Analytics workspaces** → your workspace
2. Click **Logs** in the left menu
3. Run a test query:

```kusto
// List all custom tables from switches
search "*"
| where TimeGenerated > ago(1h)
| distinct Type
| where Type endswith "_CL"

// Example: Query interface counters
CiscoInterfaceCounter_CL
| where TimeGenerated > ago(1h)
| take 10

// Example: Check all switch data by hostname
search "*"
| where TimeGenerated > ago(1h)
| where Type endswith "_CL"
| summarize count() by Type, hostname_s
```

### Telemetry Tables

The following custom tables are created in your Log Analytics workspace (Azure automatically appends `_CL` suffix):

| Table Name | Description | Platforms |
|------------|-------------|-----------|
| `CiscoInterfaceCounter_CL` | Interface traffic statistics | Cisco, SONiC |
| `CiscoInterfaceStatus_CL` | Interface admin/oper state | Cisco, SONiC |
| `CiscoInterfaceEthernet_CL` | Ethernet-specific counters | Cisco, SONiC |
| `CiscoInterfaceErrors_CL` | Interface error counters | Cisco |
| `CiscoBgpSummary_CL` | BGP neighbor summary | Cisco, SONiC |
| `CiscoLldpNeighbor_CL` | LLDP neighbor information | Cisco, SONiC |
| `CiscoTransceiver_CL` | SFP/QSFP module details | Cisco |
| `CiscoEnvTemp_CL` | Temperature sensors | Cisco, SONiC |
| `CiscoEnvPower_CL` | Power supply status | Cisco, SONiC |
| `CiscoEnvFan_CL` | Fan status | Cisco, SONiC |
| `CiscoSystemResources_CL` | CPU, memory utilization | Cisco, SONiC |
| `CiscoSystemUptime_CL` | System uptime | Cisco, SONiC |
| `CiscoIpArp_CL` | ARP table entries | Cisco, SONiC |
| `CiscoMacAddress_CL` | MAC address table | Cisco, SONiC |
| `CiscoIpRoute_CL` | Routing table | Cisco, SONiC |
| `CiscoRouteSummary_CL` | Route count summary | Cisco |
| `CiscoInventory_CL` | Hardware inventory | Cisco |
| `CiscoVersion_CL` | Software version info | Cisco |
| `CiscoClassMap_CL` | QoS class maps | Cisco (CLI only) |

---

## Architecture

### Data Flow — gNMI (Cisco + SONiC)

```
Network Switch (Cisco NX-OS or Dell SONiC)
    ├─> gNMI Get/Subscribe (gRPC, structured YANG data)
    ├─> gnmi-collector (transforms + ships in one binary)
    └─> Azure Log Analytics Workspace
```

### Data Flow — Legacy CLI (Cisco + Dell OS10)

```
Network Switch (Cisco NX-OS or Dell OS10)
    ├─> CLI commands (vsh/clish show commands)
    ├─> Parser binary (converts text output to JSON)
    ├─> Azure Logger script (adds metadata, signs requests)
    └─> Azure Log Analytics Workspace
```

### Platform Comparison

| | Cisco NX-OS (gNMI) | Cisco NX-OS (CLI) | SONiC (gNMI) | Dell OS10 (CLI) |
|---|---|---|---|---|
| **Collection** | gNMI Get over gRPC | `vsh -c "show ..."` | gNMI Get over gRPC | `clish -c "show ..."` |
| **Parsing** | YANG models (structured) | Go regex parser | YANG models (structured) | Go regex parser |
| **Shipping** | Built into collector | Bash `curl` script | Built into collector | Bash `curl` script |
| **Scheduling** | init.d daemon (long-running) | Cron every 5 min | systemd service | Cron every 5 min |
| **Port** | 50051 | N/A | 8080 | N/A |
| **Encoding** | JSON | N/A | JSON_IETF | N/A |
| **Arc Agent** | Yes (RPM + init.d) | Yes (RPM + init.d) | Yes (standard installer) | No |
| **Resilience** | YANG model versioned | Breaks on format changes | YANG model versioned | Breaks on format changes |

---

## Automated Onboarding (Copilot Skill)

For quick onboarding, use the **arc-onboarding** Copilot skill. Tell
Copilot something like:

> "Onboard the switch at 10.0.0.1"

The skill will:
1. SSH into the switch and auto-detect the vendor (Cisco NX-OS or SONiC)
2. Resolve Azure parameters from `az CLI` and environment variables
3. Fill in the setup script template with all configuration values
4. Upload and execute the script on the switch
5. **Pause** — display the `azcmagent connect` command for you to run
6. You complete Device Code Flow (DCF) in your browser
7. Tell the skill to continue — it validates Arc connection + collector

### Prerequisites

```powershell
az login
$env:WORKSPACE_ID = "<your-workspace-id>"
$env:PRIMARY_KEY   = "<your-primary-key>"
```

### Usage

```powershell
# Auto-detect vendor
.\.github\skills\arc-onboarding\onboard-switch.ps1 -SwitchIP "10.0.0.1"

# Specify vendor and credentials
.\.github\skills\arc-onboarding\onboard-switch.ps1 -SwitchIP "10.0.0.1" `
  -Vendor cisco -SshUser admin -SshPassword "secret"

# Dry-run: generate filled-in script without executing
.\.github\skills\arc-onboarding\onboard-switch.ps1 -SwitchIP "10.0.0.1" -DryRun
```

> [!NOTE]
> The `azcmagent connect` step requires interactive browser authentication
> (Device Code Flow). The skill pauses and shows you exactly what to run.
> In the future, this will be replaced with service principal auth to make
> the entire flow fully automated.

---

## Service Management

### Cisco NX-OS (init.d)

```bash
# Check status
for svc in himdsd arcproxyd extd gcad; do /etc/init.d/$svc status; done
/etc/init.d/gnmi-collectord status

# Restart all Arc services
for svc in gcad extd arcproxyd himdsd; do /etc/init.d/$svc restart; sleep 5; done

# Restart gNMI collector
/etc/init.d/gnmi-collectord restart

# View logs
tail -f /var/opt/azcmagent/log/himds.log
```

### SONiC (systemd)

```bash
# Check status
systemctl status gnmi-collector

# Restart
systemctl restart gnmi-collector

# View logs
journalctl -u gnmi-collector -f
```

---

## Uninstalling a Collector

If you need to switch between CLI parser and gNMI collector on a **Cisco NX-OS**
switch (or remove a collector entirely), follow the instructions below.

> **Note**: These steps only remove the telemetry collector. Azure Arc agent
> services (himdsd, arcproxyd, extd, gcad) are shared and are **not** removed.

### Cisco NX-OS — Uninstall gNMI Collector

```bash
# Stop and remove the gNMI collector service
/etc/init.d/gnmi-collectord stop 2>/dev/null
rm -f /etc/init.d/gnmi-collectord

# Remove the gNMI collector installation
rm -rf /opt/gnmi-collector

# Remove log and PID files
rm -f /var/log/gnmi-collector.log /var/run/gnmi-collector.pid

# Remove from autostart (if present)
sed -i '/gnmi-collectord/d' /bootflash/.rpmstore/config/etc/init.d/arcnet-autostart 2>/dev/null
```

### Cisco NX-OS — Uninstall CLI Parser

```bash
# Remove the cron job
crontab -l 2>/dev/null | grep -v 'cisco-parser-collector' | crontab -

# Remove parser and helper scripts
rm -rf /opt/cisco-parser
rm -f /opt/cisco-parser-collector.sh
rm -f /opt/cisco-azure-logger-v2.sh
rm -f /opt/azure-signature-generator.sh

# Remove temporary files and logs
rm -rf /tmp/cisco-parser-output /tmp/azure-logger
rm -f /var/log/cisco-parser-collector.log
```

---

## Troubleshooting

### Arc Agent Not Connected

```bash
azcmagent show    # Check status
azcmagent check   # Test connectivity
```

If disconnected:
1. Check network connectivity: `curl -I https://gbl.his.arc.azure.com`
2. Check DNS resolution
3. Verify services are running (see Service Management above)
4. Re-run `azcmagent connect` if needed

### No Data in Log Analytics

```bash
# Test gNMI collector manually
/opt/gnmi-collector/gnmi-collector --config /opt/gnmi-collector/config.yaml --once --dry-run

# Check collector is running
ps aux | grep gnmi-collector

# Check environment variables are set
cat /opt/gnmi-collector/gnmi-collector.env
```

### NX-OS Specific Issues

- **RPM relocation**: NX-OS installs RPM contents to `/bootflash/rpm_test/opt/` instead of `/opt/`. The setup script handles this automatically. If you installed manually, copy files from `/bootflash/rpm_test/opt/` to `/opt/`.
- **Missing azcmagent directories**: `azcmagent connect` fails with "no such file or directory" if `/var/opt/azcmagent/certs/` doesn't exist. Run: `mkdir -p /var/opt/azcmagent/{certs,log,tokens,socks,arcproxy}`
- **wget not available**: NX-OS doesn't include `wget`. Use `curl` instead for any downloads.

### SONiC Specific Issues

- **Minimum subscribe interval**: SONiC enforces a 30-second minimum for gNMI Subscribe mode.
- **JSON_IETF encoding**: SONiC requires `JSON_IETF` encoding (not `JSON`). Ensure the config file uses the correct encoding.

---

## Grafana Dashboard Setup

To visualize telemetry data, use Grafana with the Azure Monitor data source.

### Prerequisites

- Grafana 8.0+
- Azure Monitor data source plugin
- Service principal with "Log Analytics Reader" role

### Quick Setup

```bash
# Create service principal for Grafana
az ad sp create-for-rbac --name "GrafanaSwitchMonitoring" \
  --role "Log Analytics Reader" \
  --scopes "/subscriptions/<SUB_ID>/resourceGroups/ARCNET/providers/Microsoft.OperationalInsights/workspaces/SwitchTelemetry"
```

In Grafana: Configuration → Data Sources → Add "Azure Monitor" → enter tenant ID, client ID, and secret → select your workspace.

### Available Dashboard Panels

- **Device Health**: CPU, memory, temperature, power supply
- **Interface Performance**: Traffic (bps/pps), packet distribution
- **Routing Metrics**: Route counts, BGP neighbor status
- **Error Metrics**: Interface errors, discards
- **ARP & MAC Tracking**: Table size trends
- **LLDP & Inventory**: Neighbor discovery, transceiver monitoring

---

## Adding a New Vendor

See [adding_new_vendor.md](adding_new_vendor.md) for instructions on adding
support for a new switch vendor (e.g., Arista).
