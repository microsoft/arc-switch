---
name: sonic-switch-ssh
description: >
  Use this skill when you need to interact with a SONiC/Dell OS10 test switch.
  This includes running CLI commands, Linux shell commands, deploying
  gnmi-collector builds, checking file versions, restarting services, or
  verifying any changes on a SONiC/Dell OS10 device. Activate whenever the
  user mentions "sonic", "sonic switch", "dell switch", "deploy to sonic",
  "test on sonic", "gnmi-collector on sonic", or needs to verify behavior
  on a SONiC/Dell device.
allowed-tools: shell
---

# SONiC Test Switch SSH Skill

This skill lets you execute commands on a SONiC (Dell Enterprise SONiC)
lab switch and transfer files to/from it. The switch is used to test
gnmi-collector and other telemetry tools developed in this repository.

**Switch model:** DellEMC S5248f (x86_64)
**OS Version:** SONiC-OS-4.5.1-Enterprise_Premium

## Connection Details

| Setting  | Env Variable        | Default           |
|----------|---------------------|-------------------|
| Host     | `SONIC_SSH_HOST`    | `100.100.47.95`   |
| User     | `SONIC_SSH_USER`    | `admin`        |
| Password | `SONIC_PASSWORD`    | *(from Key Vault)* |

The switch uses **keyboard-interactive** authentication. The password is
automatically fetched from **Azure Key Vault** (`azurestack-network/Net-Admin`).
Falls back to the `SONIC_PASSWORD` environment variable if Key Vault is
unavailable. Requires `az login` for Key Vault access.

## Available Scripts

### `ssh-command.ps1` — Run commands on the switch

SONiC SSH drops directly into a **Linux bash shell**. All commands run
natively — no CLI wrapper needed.

**Run a command:**

```powershell
.\ssh-command.ps1 -Command "show version"
.\ssh-command.ps1 -Command "docker ps | grep gnmi"
.\ssh-command.ps1 -Command "ps aux | grep gnmi"
.\ssh-command.ps1 -Command "ls -la /home/admin/"
```

### `scp-file.ps1` — Transfer files to/from the switch

**Upload a new build of gnmi-collector:**

```powershell
.\scp-file.ps1 -LocalPath ".\gnmi-collector" -RemotePath "/home/admin/gnmi-collector" -Direction upload
```

**Download a file from the switch:**

```powershell
.\scp-file.ps1 -LocalPath ".\remote-file" -RemotePath "/home/admin/some-file" -Direction download
```

## Common Workflows

### Deploy and test gnmi-collector

1. Build the new binary (cross-compile for Linux/amd64):

   ```powershell
   $env:GOOS = "linux"; $env:GOARCH = "amd64"
   go build -o gnmi-collector ./cmd/gnmi-collector
   ```

2. Upload to the switch:

   ```powershell
   .\scp-file.ps1 -LocalPath ".\gnmi-collector" -RemotePath "/home/admin/gnmi-collector" -Direction upload
   ```

3. Make it executable and verify:

   ```powershell
   .\ssh-command.ps1 -BashCommand "chmod +x /home/admin/gnmi-collector; /home/admin/gnmi-collector --version"
   ```

4. Stop the old process and start the new one:

   ```powershell
   .\ssh-command.ps1 -BashCommand "pkill gnmi-collector; sleep 1; /home/admin/gnmi-collector &"
   ```

### Check switch status

```powershell
.\ssh-command.ps1 -Command "show version"
.\ssh-command.ps1 -Command "show system"
.\ssh-command.ps1 -BashCommand "ps aux | grep gnmi"
.\ssh-command.ps1 -BashCommand "df -h"
```

### Check gNMI server prerequisites

```powershell
.\ssh-command.ps1 -BashCommand "netstat -tlnp 2>/dev/null | grep -E '8080|50051|8081'"
.\ssh-command.ps1 -BashCommand "ls -la /etc/sonic/telemetry/ 2>/dev/null"
.\ssh-command.ps1 -Command "show running-configuration"
```

## Important Notes

- The switch runs **Dell OS10** which is SONiC-based. SSH drops into a
  CLI shell (not bash). Use `-BashCommand` for Linux commands.
- The gnmi-collector binary should be deployed to `/home/admin/`.
- Use `GNMI_USER` / `GNMI_PASS` env vars for gNMI authentication
  (separate from SSH credentials).
- Always run scripts from the skill's directory or provide the full
  path to the script.
- Scripts depend on shared modules in `.github/skills/`:
  `resolve-password.ps1` (Key Vault) and `ssh-helpers.ps1` (SSH/SCP).
