---
name: arc-onboarding
description: >
  Use this skill when the user asks to onboard a network switch to Azure Arc.
  This includes running setup scripts, detecting vendor type, resolving Azure
  parameters, and deploying the Arc agent + gNMI collector. Activate whenever
  the user mentions "onboard", "onboard switch", "arc onboard", "add switch
  to arc", or asks to set up a new switch for Azure Arc management.
allowed-tools: shell
---

# Azure Arc Switch Onboarding Skill

This skill automates the onboarding of Cisco NX-OS and Dell SONiC switches
to Azure Arc with gNMI telemetry collection.

## What it does

1. **Detects the vendor** by SSH-ing into the switch and fingerprinting the OS
2. **Resolves Azure parameters** (subscription, tenant, resource group, workspace)
3. **Fills in the setup script** from the template in `Docs/arcnet_onboarding_instructions/`
4. **Uploads and executes** the setup script on the switch
5. **Validates** services are running and gNMI connectivity works
6. **Outputs the `azcmagent connect` command** for the human device-code auth step

## Usage

### Full onboarding

```powershell
.\onboard-switch.ps1 -SwitchIP "10.0.0.1"
```

### With explicit vendor and credentials

```powershell
.\onboard-switch.ps1 -SwitchIP "10.0.0.1" -Vendor cisco -SshUser admin -SshPassword "secret"
```

### Dry-run (generate script without executing)

```powershell
.\onboard-switch.ps1 -SwitchIP "10.0.0.1" -DryRun
```

## Parameters

| Parameter       | Required | Default              | Description |
|-----------------|----------|----------------------|-------------|
| `-SwitchIP`     | Yes      | —                    | IP address of the switch |
| `-Vendor`       | No       | Auto-detect          | `cisco` or `sonic` |
| `-SshUser`      | No       | `admin`              | SSH username |
| `-SshPassword`  | No       | From Key Vault       | SSH password (falls back to `SWITCH_PASSWORD` / `SONIC_PASSWORD` env var) |
| `-MachineName`  | No       | Auto-detect hostname | Azure Arc machine name |
| `-ResourceGroup`| No       | `ARCNET`             | Azure resource group |
| `-Region`       | No       | `eastus`             | Azure region |
| `-DryRun`       | No       | `$false`             | Generate script but don't execute |

## Azure Parameters (resolved automatically)

The skill resolves these from `az CLI` and environment variables:

| Parameter        | Source | Env Var Fallback |
|------------------|--------|------------------|
| Subscription ID  | `az account show` | `SUBSCRIPTION_ID` |
| Tenant ID        | `az account show` | `TENANT_ID` |
| Workspace ID     | — | `WORKSPACE_ID` (required) |
| Primary Key      | — | `PRIMARY_KEY` (required) |
| Secondary Key    | — | `SECONDARY_KEY` |
| gNMI User        | Same as SSH user | `GNMI_USER` |
| gNMI Password    | Same as SSH password | `GNMI_PASS` |

## Prerequisites

- `az login` completed (for subscription/tenant resolution)
- `WORKSPACE_ID` and `PRIMARY_KEY` environment variables set
- SSH access to the target switch
- The switch must have outbound internet for downloading Arc agent and collector

## Onboarding Flow

The onboarding is a **two-phase process** because `azcmagent connect`
requires Device Code Flow (DCF) — interactive browser sign-in that
cannot be automated.

```
  User: "Onboard switch 10.0.0.1"
    │
    │  PHASE 1 — Automated (skill handles this)
    ├─ 1. SSH → detect vendor (Cisco NX-OS / SONiC)
    ├─ 2. SSH → get hostname
    ├─ 3. az CLI → get subscription ID, tenant ID
    ├─ 4. Read template script from Docs/
    ├─ 5. Fill in all config values
    ├─ 6. SCP script to switch
    ├─ 7. SSH → execute script (installs Arc agent + gNMI collector)
    │
    │  PHASE 2 — Manual (user does this)
    ├─ 8. Skill outputs the azcmagent connect command
    ├─ 9. User SSHs into the switch and runs azcmagent connect
    ├─10. User completes DCF in browser (URL + code)
    │
    │  PHASE 3 — Automated (skill resumes)
    ├─11. SSH → verify azcmagent show reports "Connected"
    └─12. SSH → verify gNMI collector service is running
```

> [!NOTE]
> **Phase 2 is a hard stop.** The skill will pause and tell you exactly
> what to run on the switch. Once you've completed DCF and confirmed the
> agent is connected, tell the skill to continue and it will validate
> everything is healthy.
>
> **Future improvement:** Replace DCF with service principal auth
> (`azcmagent connect --service-principal-id ...`) to make the entire
> flow fully automated.

## Important Notes

- **Device Code Flow (DCF)**: `azcmagent connect` requires the user to
  open a URL in their browser and enter a code. The skill cannot automate
  this step. It will display the exact command and wait for the user to
  confirm completion before proceeding with validation.

- **wget not available**: Network switches do not have `wget` installed.
  All setup scripts use `curl` instead. If using the Azure Portal
  generated onboarding script, replace `wget` with `curl -sSL ... -o`.

- **Cisco NX-OS**: Uses `run bash` prefix for shell commands. Setup script
  creates init.d services and systemd shims (NX-OS has no systemd).
  Binary is installed to `/opt/gnmi-collector/`. Config persists across
  reboots via `/bootflash/.rpmstore/`.

- **Dell SONiC**: Standard Debian with systemd. For SONiC, the Arc agent
  can be installed via the standard Azure Portal onboarding script (with
  curl fix), then the gNMI collector is installed separately. The setup
  script handles both steps if run end-to-end.

- **Credentials**: SSH password is resolved from Azure Key Vault
  (`azurestack-network/Net-Admin`) via `resolve-password.ps1`.

- **Safety**: The script does NOT restart existing workloads or modify
  switch forwarding configuration. It only installs monitoring services.
