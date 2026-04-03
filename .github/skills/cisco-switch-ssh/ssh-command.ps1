<#
.SYNOPSIS
    Runs a command on the Cisco NX-OS switch over SSH using keyboard-interactive auth.
.DESCRIPTION
    Uses SSH_ASKPASS to handle keyboard-interactive authentication automatically.
    Password is fetched from Azure Key Vault (azurestack-network/Net-Admin),
    falling back to the SWITCH_PASSWORD environment variable.
    The switch IP and user are read from SWITCH_SSH_HOST / SWITCH_SSH_USER env vars,
    falling back to defaults from the SSH config (100.71.34.149 / camilose).
.PARAMETER Command
    The NX-OS CLI command to run. For Linux shell commands, prefix with "run bash".
.PARAMETER BashCommand
    A Linux shell command to run inside bash on the switch. Automatically wrapped
    with "run bash" so you don't have to.
.PARAMETER TimeoutSeconds
    SSH connection timeout in seconds. Default: 30.
.EXAMPLE
    .\ssh-command.ps1 -Command "show version"
.EXAMPLE
    .\ssh-command.ps1 -BashCommand "ls -la /usr/temp/gnmi-collector"
.EXAMPLE
    .\ssh-command.ps1 -BashCommand "cat /usr/temp/gnmi-collector --version 2>&1 || /usr/temp/gnmi-collector --version 2>&1"
#>
param(
    [Parameter(Mandatory = $false)]
    [string]$Command,

    [Parameter(Mandatory = $false)]
    [string]$BashCommand,

    [Parameter(Mandatory = $false)]
    [int]$TimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"

# Validate parameters
if (-not $Command -and -not $BashCommand) {
    Write-Error "You must provide either -Command or -BashCommand."
    exit 1
}
if ($Command -and $BashCommand) {
    Write-Error "Provide only one of -Command or -BashCommand, not both."
    exit 1
}

# Resolve the actual command to send
$remoteCmd = if ($BashCommand) { "run bash $BashCommand" } else { $Command }

# Resolve connection details from environment (with defaults)
$switchHost = if ($env:SWITCH_SSH_HOST) { $env:SWITCH_SSH_HOST } else { "100.71.34.149" }
$switchUser = if ($env:SWITCH_SSH_USER) { $env:SWITCH_SSH_USER } else { "camilose" }

# Resolve password from Key Vault or environment variable
$resolveScript = Join-Path $PSScriptRoot "..\resolve-password.ps1"
$password = & $resolveScript -EnvVarName "SWITCH_PASSWORD"

# Locate Git's SSH (supports SSH_ASKPASS reliably on Windows)
$gitSsh = "C:\Program Files\Git\usr\bin\ssh.exe"
if (-not (Test-Path $gitSsh)) {
    $gitSsh = (Get-Command ssh -ErrorAction SilentlyContinue).Source
    if (-not $gitSsh) {
        Write-Error "Cannot find ssh.exe. Install Git for Windows or OpenSSH."
        exit 1
    }
}

# Build the SSH_ASKPASS helper (temporary .cmd that echoes the password)
$askpassPath = Join-Path $env:TEMP "copilot_switch_askpass_$PID.cmd"
Set-Content -Path $askpassPath -Value "@echo $password" -NoNewline

try {
    $env:SSH_ASKPASS = $askpassPath
    $env:SSH_ASKPASS_REQUIRE = "force"
    $env:DISPLAY = "dummy"

    $sshArgs = @(
        "-o", "StrictHostKeyChecking=no",
        "-o", "UserKnownHostsFile=NUL",
        "-o", "PreferredAuthentications=keyboard-interactive",
        "-o", "PubkeyAuthentication=no",
        "-o", "ConnectTimeout=$TimeoutSeconds",
        "-o", "LogLevel=ERROR",
        "$switchUser@$switchHost",
        $remoteCmd
    )

    $output = & $gitSsh @sshArgs 2>&1

    # Filter out banner noise
    $filtered = $output | Where-Object {
        $line = $_.ToString()
        $line -notmatch '^\*\* WARNING' -and
        $line -notmatch 'post-quantum' -and
        $line -notmatch 'store now, decrypt later' -and
        $line -notmatch 'server may need to be upgraded' -and
        $line -notmatch 'NOTICE' -and
        $line -notmatch '^hostname ' -and
        $line -notmatch '^BuildVersion:' -and
        $line -notmatch 'Unauthorized access' -and
        $line -notmatch 'subject to monitoring' -and
        $line -notmatch 'Permanently added'
    }

    $filtered | ForEach-Object { $_.ToString() }

    if ($LASTEXITCODE -ne 0) {
        Write-Error "SSH command failed with exit code $LASTEXITCODE"
        exit $LASTEXITCODE
    }
}
finally {
    # Clean up askpass helper
    Remove-Item -Path $askpassPath -Force -ErrorAction SilentlyContinue
}
