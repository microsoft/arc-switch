<#
.SYNOPSIS
    Runs a command on a SONiC/Dell OS10 switch over SSH.
.DESCRIPTION
    Uses SSH_ASKPASS to handle password authentication automatically.
    Password is fetched from Azure Key Vault (azurestack-network/Net-Admin),
    falling back to the SONIC_PASSWORD environment variable.
    The switch IP and user are read from SONIC_SSH_HOST / SONIC_SSH_USER env vars,
    falling back to defaults (100.100.47.95 / camilose).
.PARAMETER Command
    The command to run on the switch. Runs directly in the Linux shell.
.PARAMETER TimeoutSeconds
    SSH connection timeout in seconds. Default: 30.
.EXAMPLE
    .\ssh-command.ps1 -Command "show version"
.EXAMPLE
    .\ssh-command.ps1 -Command "docker ps | grep gnmi"
.EXAMPLE
    .\ssh-command.ps1 -Command "ps aux | grep gnmi"
.EXAMPLE
    .\ssh-command.ps1 -Command "ls -la /home/camilose/gnmi-collector"
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$Command,

    [Parameter(Mandatory = $false)]
    [int]$TimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"

# Resolve connection details from environment (with defaults)
$switchHost = if ($env:SONIC_SSH_HOST) { $env:SONIC_SSH_HOST } else { "100.100.47.95" }
$switchUser = if ($env:SONIC_SSH_USER) { $env:SONIC_SSH_USER } else { "camilose" }

# Resolve password from Key Vault or environment variable
$resolveScript = Join-Path $PSScriptRoot "..\resolve-password.ps1"
$password = & $resolveScript -EnvVarName "SONIC_PASSWORD"

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
$askpassPath = Join-Path $env:TEMP "copilot_sonic_askpass_$PID.cmd"
Set-Content -Path $askpassPath -Value "@echo $password" -NoNewline

try {
    $env:SSH_ASKPASS = $askpassPath
    $env:SSH_ASKPASS_REQUIRE = "force"
    $env:DISPLAY = "dummy"

    $sshArgs = @(
        "-o", "StrictHostKeyChecking=no",
        "-o", "UserKnownHostsFile=NUL",
        "-o", "PreferredAuthentications=keyboard-interactive,password",
        "-o", "PubkeyAuthentication=no",
        "-o", "ConnectTimeout=$TimeoutSeconds",
        "-o", "LogLevel=ERROR",
        "$switchUser@$switchHost",
        $Command
    )

    $output = & $gitSsh @sshArgs 2>&1

    # Filter out SSH banner noise
    $filtered = $output | Where-Object {
        $line = $_.ToString()
        $line -notmatch '^\*\* WARNING' -and
        $line -notmatch 'post-quantum' -and
        $line -notmatch 'store now, decrypt later' -and
        $line -notmatch 'server may need to be upgraded' -and
        $line -notmatch 'NOTICE' -and
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
