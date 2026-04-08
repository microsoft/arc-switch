---
name: cisco-switch-ssh
description: >
  Use this skill when you need to interact with the Cisco NX-OS test switch.
  This includes running NX-OS CLI commands, Linux shell commands, deploying
  gnmi-collector builds, checking file versions, restarting services, or
  verifying any changes on the switch. Activate whenever the user mentions
  "switch", "deploy to switch", "test on switch", "gnmi-collector on switch",
  or needs to verify behavior on the Cisco device.
allowed-tools: shell
---

# Cisco NX-OS Test Switch SSH Skill

This skill lets you execute commands on the Cisco NX-OS lab switch and
transfer files to/from it. The switch is used to test gnmi-collector
and other telemetry tools developed in this repository.

## Connection Details

| Setting  | Env Variable       | Default           |
|----------|--------------------|-------------------|
| Host     | `SWITCH_SSH_HOST`  | `100.71.34.149`   |
| User     | `SWITCH_SSH_USER`  | `admin`        |
| Password | `SWITCH_PASSWORD`  | *(from Key Vault)* |

The switch uses **keyboard-interactive** authentication. The password is
automatically fetched from **Azure Key Vault** (`azurestack-network/Net-Admin`).
Falls back to the `SWITCH_PASSWORD` environment variable if Key Vault is
unavailable. Requires `az login` for Key Vault access.

## Available Scripts

### `ssh-command.ps1` — Run commands on the switch

**Run an NX-OS CLI command:**

```powershell
.\ssh-command.ps1 -Command "show version"
.\ssh-command.ps1 -Command "show interface brief"
```

**Run a Linux shell command on the switch:**

```powershell
.\ssh-command.ps1 -BashCommand "ls -la /usr/temp/gnmi-collector"
.\ssh-command.ps1 -BashCommand "/usr/temp/gnmi-collector --version"
.\ssh-command.ps1 -BashCommand "ps aux | grep gnmi"
```

### `scp-file.ps1` — Transfer files to/from the switch

**Upload a new build of gnmi-collector:**

```powershell
.\scp-file.ps1 -LocalPath ".\gnmi-collector" -RemotePath "/usr/temp/gnmi-collector" -Direction upload
```

**Download a file from the switch:**

```powershell
.\scp-file.ps1 -LocalPath ".\remote-file" -RemotePath "/usr/temp/some-file" -Direction download
```

## Common Workflows

### Deploy and test gnmi-collector

1. Build the new binary (cross-compile for Linux/arm64 if needed):

   ```powershell
   $env:GOOS = "linux"; $env:GOARCH = "arm64"
   go build -o gnmi-collector ./cmd/gnmi-collector
   ```

2. Upload to the switch:

   ```powershell
   .\scp-file.ps1 -LocalPath ".\gnmi-collector" -RemotePath "/usr/temp/gnmi-collector" -Direction upload
   ```

3. Make it executable and verify:

   ```powershell
   .\ssh-command.ps1 -BashCommand "chmod +x /usr/temp/gnmi-collector && /usr/temp/gnmi-collector --version"
   ```

4. Stop the old process and start the new one:

   ```powershell
   .\ssh-command.ps1 -BashCommand "pkill gnmi-collector; sleep 1; /usr/temp/gnmi-collector &"
   ```

### Check switch status

```powershell
.\ssh-command.ps1 -Command "show version"
.\ssh-command.ps1 -Command "show interface brief"
.\ssh-command.ps1 -BashCommand "ps aux | grep gnmi"
.\ssh-command.ps1 -BashCommand "df -h /usr/temp"
```

## Important Notes

- The switch runs **NX-OS**. For Linux commands, always use `-BashCommand`
  (or prefix with `run bash` if using `-Command`).
- The gnmi-collector binary lives at `/usr/temp/gnmi-collector`.
- Use `GNMI_USER` / `GNMI_PASS` env vars for gNMI authentication
  (separate from SSH credentials).
- Always run scripts from the skill's directory or provide the full
  path to the script.
- Scripts depend on shared modules in `.github/skills/`:
  `resolve-password.ps1` (Key Vault) and `ssh-helpers.ps1` (SSH/SCP).
