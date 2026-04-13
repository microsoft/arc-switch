# Enabling GitHub Copilot CLI for SSH to a Cisco NX-OS Switch

This guide sets up GitHub Copilot CLI so it can SSH into a single Cisco
NX-OS switch on your behalf — run show commands, manage the gnmi-collector
daemon, deploy new builds, and inspect live telemetry. This is tailored for
the `arc-switch` gnmi-collector development workflow.

---

## Quick Start — Hand This to Copilot

> **Copy the prompt below into Copilot CLI.** It creates the SSH config,
> copilot instructions, and launch script. Replace the placeholder values
> with your switch details.

### Prerequisites (Human Must Do First)

```powershell
# 1. OpenSSH client (ships with Windows 10+)
ssh -V

# 2. GitHub Copilot CLI
copilot --version

# 3. VPN connected (switch management IP must be reachable)
Test-NetConnection <SWITCH_IP> -Port 22

# 4. Set credentials as environment variables
$env:SWITCH_USER = "admin"
$env:SWITCH_PASSWORD = "YourSwitchPassword"

# 5. (Optional) Set gnmi-collector credentials for remote deployment
$env:GNMI_USER = "admin"
$env:GNMI_PASS = "YourGnmiPassword"
$env:WORKSPACE_ID = "YourAzureWorkspaceId"
$env:PRIMARY_KEY = "YourAzurePrimaryKey"
```

### The Prompt

Paste this into Copilot CLI and replace the placeholder values:

````text
Set up SSH access to my Cisco NX-OS switch. Create the SSH config,
copilot instructions, and launch script so you can SSH to the device
on my behalf and manage the gnmi-collector.

My device:
- Name: cisco-switch, IP: 100.71.34.149, User: admin, OS: NX-OS

My info:
- OS: Windows 11 / PowerShell 7
- Password method: environment variable SWITCH_PASSWORD
- Repo: C:\repos\networking\arc-switch

Step 1 — Create these files:
1. ~/.ssh/config — main config with includes (if it doesn't exist)
2. ~/.ssh/includes/common/config — global SSH defaults
3. ~/.ssh/includes/work/config — work defaults (default user)
4. ~/.ssh/includes/work/cisco-lab.conf — host definition for my switch
5. ~/.copilot/instructions.md — copilot instructions with:
   - SSH section listing the alias and how to run commands
   - gnmi-collector section with build, deploy, and management commands
   - NX-OS reference commands for common troubleshooting

Step 2 — After creating the files:
- Test SSH config syntax with: ssh -G cisco-switch
- Test connectivity with: ssh cisco-switch "show version"
  (I'll provide the password when prompted)
````

### What Copilot Will Create

```text
~/.ssh/
├── config                          # Include-based root config
├── includes/
│   ├── common/
│   │   └── config                  # ServerAliveInterval, TCPKeepAlive
│   └── work/
│       ├── config                  # Default user, port 22
│       └── cisco-lab.conf          # Cisco switch host definition
~/.copilot/
└── instructions.md                 # SSH + gnmi-collector context
```

### After Copilot Finishes (Human Steps)

1. **Set your password**: `$env:SWITCH_PASSWORD = "YourPassword"`
2. **Test manually**: `ssh cisco-switch "show version"`
3. **Relaunch with context**: `.\init-copilot-cisco.ps1`
4. **Verify Copilot can SSH**: Ask "SSH to cisco-switch and show the
   running-config for grpc"

---

## Why This Matters

When Copilot CLI can SSH to the switch, it can:

- Build gnmi-collector locally, SCP it to the switch, and restart the
  daemon — all in one shot
- Check collector logs and status without you switching terminals
- Run NX-OS `show` commands and parse output for debugging
- Verify gNMI feature state, BGP peering, interface counters
- Compare live CLI output against gNMI telemetry data

## Architecture Overview

```text
┌──────────────────────────────────────────────────────────────────┐
│  Your Workstation (Windows)                                      │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │ Copilot CLI  │  │ arc-switch   │  │ ~/.ssh/includes/work/  │ │
│  │              │  │ repo         │  │ cisco-lab.conf         │ │
│  │ reads        │  │              │  │                        │ │
│  │ instructions │  │ make build-  │  │ Host cisco-switch      │ │
│  │ + SSH config │  │ linux        │  │   HostName 10.x.x.x   │ │
│  └──────┬───────┘  └──────┬───────┘  └────────────────────────┘ │
│         │                 │                                      │
│         │    ssh cisco-switch "<cmd>"                             │
│         │    scp gnmi-collector cisco-switch:/opt/gnmi-collector/ │
│         ▼                 ▼                                      │
├─────────────────── SSH (port 22) ────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │  Cisco NX-OS Switch                                         │ │
│  │                                                             │ │
│  │  /opt/gnmi-collector/                                       │ │
│  │  ├── gnmi-collector          (binary, runs as daemon)       │ │
│  │  ├── config.yaml             (YANG paths, Azure creds)      │ │
│  │  └── env                     (environment variables)        │ │
│  │                                                             │ │
│  │  gnmi-collector ──gRPC──▶ localhost:50051 (gNMI server)     │ │
│  │       │                                                     │ │
│  │       └──HTTPS──▶ Azure Log Analytics                       │ │
│  └──────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

## Step-by-Step Setup

### Step 1: Create the SSH Config

**`~/.ssh/includes/work/cisco-lab.conf`** (your switch):

```ssh-config
# Cisco NX-OS Lab Switch
# Direct SSH via management IP (requires VPN)

Host cisco-switch
    HostName <SWITCH_IP>
    User admin
    Port 22
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ServerAliveInterval 30
    ServerAliveCountMax 4
    TCPKeepAlive yes
```

> **Note:** `StrictHostKeyChecking no` and `UserKnownHostsFile /dev/null`
> prevent host key prompts that block Copilot's non-interactive SSH.

If you already have `~/.ssh/config` with includes set up (from the SONiC
guide), just drop the `cisco-lab.conf` file into `~/.ssh/includes/work/`
and it will be picked up automatically.

If starting fresh, create the full structure:

**`~/.ssh/config`** (main entry point):

```ssh-config
Include includes/common/config
Include includes/work/*.conf
Include includes/work/config
```

**`~/.ssh/includes/common/config`**:

```ssh-config
Host *
    ServerAliveInterval 30
    ServerAliveCountMax 4
    TCPKeepAlive yes
```

**`~/.ssh/includes/work/config`**:

```ssh-config
Host *
    Port 22
```

### Step 2: Handle Credentials

Copilot CLI cannot type passwords into interactive SSH prompts. You need
one of these methods for passwordless access:

#### Option A: SSH Keys (Recommended)

```powershell
# Generate a key pair
ssh-keygen -t rsa -b 2048 -f $env:USERPROFILE\.ssh\id_cisco_lab -C "cisco-lab"

# Copy to the switch (NX-OS requires RSA, not ed25519)
# On the switch, configure:
#   username admin sshkey file bootflash:id_cisco_lab.pub

# Add IdentityFile to SSH config:
# Host cisco-switch
#     IdentityFile ~/.ssh/id_cisco_lab
```

#### Option B: sshpass via WSL (Quick and Dirty)

```powershell
# From WSL or Git Bash:
sshpass -p "$SWITCH_PASSWORD" ssh cisco-switch "show version"

# Copilot can use this pattern in PowerShell via wsl:
wsl sshpass -p $env:SWITCH_PASSWORD ssh cisco-switch "show version"
```

#### Option C: SSH Agent

```powershell
# Start the agent service (once)
Set-Service ssh-agent -StartupType Automatic
Start-Service ssh-agent

# Add your key
ssh-add $env:USERPROFILE\.ssh\id_cisco_lab
```

### Step 3: Test SSH Manually

```powershell
# Verify alias resolution
ssh -G cisco-switch | Select-String "hostname|user|port"

# Test connectivity
ssh cisco-switch "show version"
```

If this works, Copilot will be able to use it too.

### Step 4: Tell Copilot About Your Setup

Create or update **`~/.copilot/instructions.md`** (global) or
**`<repo>/.github/copilot-instructions.md`** (per-repo).

Add the following sections:

````markdown
## SSH Access — Cisco NX-OS Switch

I have a Cisco NX-OS switch accessible via SSH. Use the alias
`cisco-switch` to connect and run commands.

### SSH Alias

| Alias | Device | IP | User | OS |
|-------|--------|----|------|----|
| `cisco-switch` | Cisco Nexus | <SWITCH_IP> | admin | NX-OS 9.3+ |

### Running Commands

Run NX-OS commands via SSH using this pattern:

```bash
ssh cisco-switch "<command>"
```

Examples:

```bash
ssh cisco-switch "show version"
ssh cisco-switch "show ip bgp summary"
ssh cisco-switch "show interface status"
ssh cisco-switch "show feature | grep grpc"
ssh cisco-switch "show running-config | section grpc"
```

For commands with pipes, wrap the entire remote command in double quotes:

```bash
ssh cisco-switch "show ip arp | head 20"
ssh cisco-switch "show mac address-table | grep Eth1/1"
```

### Important Notes

- Do NOT run configuration commands (`configure terminal`) without
  explicit user approval
- Prefer `show` commands for read-only inspection
- NX-OS uses `|` for pipe filtering: `include`, `grep`, `head`,
  `section`, `begin`
- Some commands require `no-more` to disable pagination:
  `terminal length 0` is set per-session

## gnmi-collector Management

The gnmi-collector daemon runs ON the Cisco switch at
`/opt/gnmi-collector/`. It collects gNMI telemetry and sends it to
Azure Log Analytics.

### Checking Status

```bash
# Check if the collector is running
ssh cisco-switch "/etc/init.d/gnmi-collectord status"

# Check the process directly
ssh cisco-switch "ps aux | grep gnmi-collector"

# View recent logs (last 50 lines)
ssh cisco-switch "tail -50 /var/log/gnmi-collector.log"

# View live logs (use with caution — interactive)
ssh cisco-switch "tail -f /var/log/gnmi-collector.log"

# Check the running config
ssh cisco-switch "cat /opt/gnmi-collector/config.yaml"
```

### Restarting the Collector

```bash
ssh cisco-switch "/etc/init.d/gnmi-collectord restart"
```

### Stopping / Starting

```bash
ssh cisco-switch "/etc/init.d/gnmi-collectord stop"
ssh cisco-switch "/etc/init.d/gnmi-collectord start"
```

### Build and Deploy Workflow

To build a new gnmi-collector binary and deploy it to the switch:

```powershell
# 1. Build for Linux (the switch runs Linux under NX-OS)
cd C:\repos\networking\arc-switch\src\TelemetryClient
# On Windows, use Go directly:
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"
go build -ldflags="-s -w" -o gnmi-collector ./cmd/gnmi-collector/

# 2. Copy the binary to the switch
scp gnmi-collector cisco-switch:/opt/gnmi-collector/gnmi-collector

# 3. Restart the daemon on the switch
ssh cisco-switch "/etc/init.d/gnmi-collectord restart"

# 4. Verify it started
ssh cisco-switch "/etc/init.d/gnmi-collectord status"

# 5. Check logs for errors
ssh cisco-switch "tail -20 /var/log/gnmi-collector.log"

# 6. Clean up local binary
Remove-Item gnmi-collector
```

### Deploying a New Config

```powershell
# Copy updated config to the switch
scp config.yaml cisco-switch:/opt/gnmi-collector/config.yaml

# Restart to pick up new config
ssh cisco-switch "/etc/init.d/gnmi-collectord restart"
```

### Running a One-Shot Collection (Debug)

```bash
# Run a single collection cycle with output to stdout
ssh cisco-switch "/opt/gnmi-collector/gnmi-collector --config /opt/gnmi-collector/config.yaml --once --dry-run"
```

### Dumping Raw gNMI Responses (Debug)

```bash
# Save raw gNMI JSON to a directory for inspection
ssh cisco-switch "/opt/gnmi-collector/gnmi-collector --config /opt/gnmi-collector/config.yaml --once --dump /tmp/gnmi-dump"
ssh cisco-switch "ls -la /tmp/gnmi-dump/"
ssh cisco-switch "cat /tmp/gnmi-dump/<file>.json"
```

## NX-OS Quick Reference

Common NX-OS commands for troubleshooting the gnmi-collector setup:

### gNMI / gRPC Status

```bash
ssh cisco-switch "show feature | grep grpc"
ssh cisco-switch "show running-config | section grpc"
ssh cisco-switch "show grpc internal service-status"
```

### Interfaces

```bash
ssh cisco-switch "show interface status"
ssh cisco-switch "show interface Eth1/1"
ssh cisco-switch "show interface counters"
ssh cisco-switch "show interface counters errors"
```

### BGP

```bash
ssh cisco-switch "show ip bgp summary"
ssh cisco-switch "show bgp all summary"
ssh cisco-switch "show ip bgp neighbors"
```

### ARP / MAC

```bash
ssh cisco-switch "show ip arp"
ssh cisco-switch "show mac address-table"
```

### LLDP

```bash
ssh cisco-switch "show lldp neighbors"
ssh cisco-switch "show lldp neighbors detail"
```

### System Health

```bash
ssh cisco-switch "show system resources"
ssh cisco-switch "show system uptime"
ssh cisco-switch "show environment temperature"
ssh cisco-switch "show environment power"
ssh cisco-switch "show inventory"
```

### Transceiver / Optics

```bash
ssh cisco-switch "show interface transceiver"
ssh cisco-switch "show interface transceiver details"
```

### Linux Shell (Bash on NX-OS)

NX-OS has a Linux shell underneath. Access it for gnmi-collector
management:

```bash
# Run bash commands directly via run bash
ssh cisco-switch "run bash ls -la /opt/gnmi-collector/"
ssh cisco-switch "run bash tail -20 /var/log/gnmi-collector.log"
ssh cisco-switch "run bash ps aux | grep gnmi"

# Check disk space
ssh cisco-switch "run bash df -h"

# Check environment variables
ssh cisco-switch "run bash cat /opt/gnmi-collector/env"
```

Note: Some NX-OS versions require `run bash` prefix to execute Linux
commands from the NX-OS CLI prompt.
````

### Step 5: Create the Launch Script

Save as `init-copilot-cisco.ps1` in the repo root or your home
directory:

```powershell
# init-copilot-cisco.ps1
# Launch Copilot CLI with SSH config and arc-switch repo context.

$sshPath = "$env:USERPROFILE\.ssh"
$repoPath = "C:\repos\networking\arc-switch"

# Ensure credentials are set
if (-not $env:SWITCH_PASSWORD) {
    Write-Warning "SWITCH_PASSWORD not set. SSH may fail."
    Write-Host 'Set it with: $env:SWITCH_PASSWORD = "YourPassword"'
}

copilot `
    --add-dir $sshPath `
    --add-dir "$repoPath\src\TelemetryClient" `
    --add-dir "$repoPath\tools\gnmi-collector"
```

### Step 6: Verify End-to-End

Start Copilot CLI and test these interactions:

```text
You: SSH to cisco-switch and show the version
Copilot: *runs ssh cisco-switch "show version" and parses output*

You: Check if the gnmi-collector is running on the switch
Copilot: *runs ssh cisco-switch "/etc/init.d/gnmi-collectord status"*

You: Show me the last 20 lines of the collector log
Copilot: *runs ssh cisco-switch "tail -20 /var/log/gnmi-collector.log"*

You: Build the gnmi-collector, deploy it to the switch, and restart
Copilot: *runs make build-linux, scp, ssh restart, checks status*
```

---

## Common Workflows

### Deploy a New gnmi-collector Build

Ask Copilot:

```text
Build the gnmi-collector for Linux, SCP it to cisco-switch, restart
the daemon, and show me the logs to confirm it started.
```

Copilot will:

1. `cd src/TelemetryClient && go build` (cross-compiled for Linux)
2. `scp gnmi-collector cisco-switch:/opt/gnmi-collector/`
3. `ssh cisco-switch "/etc/init.d/gnmi-collectord restart"`
4. `ssh cisco-switch "tail -20 /var/log/gnmi-collector.log"`

### Debug a Telemetry Path

Ask Copilot:

```text
Run gnmi-collector in one-shot dry-run mode on the switch and show
me the output for interface-counters.
```

### Compare CLI vs gNMI Data

Ask Copilot:

```text
SSH to the switch, run "show ip bgp summary", then run gnmi-collector
in dry-run mode for the bgp-peers path. Compare the two outputs.
```

### Update the Collector Config

Ask Copilot:

```text
Update the sample_interval for interface-counters to 30s in
config.cisco.yaml, copy it to the switch, and restart the
collector.
```

---

## Troubleshooting

### SSH Connection Refused

```powershell
# Verify network connectivity
Test-NetConnection <SWITCH_IP> -Port 22

# Check VPN is connected
# Check the switch has SSH enabled:
ssh cisco-switch "show feature | grep ssh"
```

### SSH Hangs on Password Prompt

Copilot cannot enter passwords interactively. You must set up
key-based authentication (Step 2, Option A) or use sshpass.

### gnmi-collector Won't Start

```bash
# Check if binary exists and is executable
ssh cisco-switch "run bash ls -la /opt/gnmi-collector/gnmi-collector"

# Check if config exists
ssh cisco-switch "run bash cat /opt/gnmi-collector/config.yaml"

# Check environment variables
ssh cisco-switch "run bash cat /opt/gnmi-collector/env"

# Run manually to see startup errors
ssh cisco-switch "run bash /opt/gnmi-collector/gnmi-collector --config /opt/gnmi-collector/config.yaml --once --dry-run 2>&1"
```

### gRPC Not Enabled on Switch

```bash
# Check gRPC feature
ssh cisco-switch "show feature | grep grpc"

# If disabled, enable it (requires config mode — ask user first):
# configure terminal
# feature grpc
# grpc use-vrf default
```

### Permission Denied on SCP

NX-OS may restrict file writes. Use the bash shell path:

```powershell
# SCP to a writable location first
scp gnmi-collector cisco-switch:/bootflash/gnmi-collector

# Then move it into place via SSH
ssh cisco-switch "run bash sudo cp /bootflash/gnmi-collector /opt/gnmi-collector/gnmi-collector"
ssh cisco-switch "run bash sudo chmod +x /opt/gnmi-collector/gnmi-collector"
```

---

## Minimum Viable Setup (TL;DR)

If you just want Copilot to SSH to one Cisco switch with minimal files:

### 1. SSH Config (`~/.ssh/config`)

```ssh-config
Host cisco-switch
    HostName <SWITCH_IP>
    User admin
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
```

### 2. Copilot Instructions (`~/.copilot/instructions.md`)

````markdown
## SSH

I have a Cisco NX-OS switch at `cisco-switch` (<SWITCH_IP>, user: admin).
Use `ssh cisco-switch "<command>"` to run NX-OS commands.

The gnmi-collector daemon runs at `/opt/gnmi-collector/` on the switch.
Manage it with `/etc/init.d/gnmi-collectord {start|stop|restart|status}`.
Logs are at `/var/log/gnmi-collector.log`.

For Linux shell commands on NX-OS, prefix with `run bash`.
````

### 3. Launch

```powershell
copilot --add-dir ~/.ssh
```

### 4. Use

```text
You: SSH to cisco-switch and check the gnmi-collector status
Copilot: *runs ssh cisco-switch "/etc/init.d/gnmi-collectord status"*
```
