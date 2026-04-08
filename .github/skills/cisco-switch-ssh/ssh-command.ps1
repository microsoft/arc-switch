<#
.SYNOPSIS
    Runs a command on the Cisco NX-OS switch over SSH using keyboard-interactive auth.
.DESCRIPTION
    Uses SSH_ASKPASS to handle keyboard-interactive authentication automatically.
    Password is fetched from Azure Key Vault (azurestack-network/Net-Admin),
    falling back to the SWITCH_PASSWORD environment variable.
    The switch IP and user are read from SWITCH_SSH_HOST / SWITCH_SSH_USER env vars,
    falling back to defaults from the SSH config (100.71.34.149 / admin).
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

# Import shared helpers
. (Join-Path $PSScriptRoot "..\ssh-helpers.ps1")

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
$switchUser = if ($env:SWITCH_SSH_USER) { $env:SWITCH_SSH_USER } else { "admin" }

# Resolve password from Key Vault or environment variable
$resolveScript = Join-Path $PSScriptRoot "..\resolve-password.ps1"
$password = & $resolveScript -EnvVarName "SWITCH_PASSWORD"

# NX-OS specific banners to filter
$ciscoFilters = @('^hostname ', '^BuildVersion:')

Invoke-SshCommand `
    -User $switchUser `
    -HostName $switchHost `
    -Password $password `
    -Command $remoteCmd `
    -AuthMethods "keyboard-interactive" `
    -Timeout $TimeoutSeconds `
    -ExtraFilterPatterns $ciscoFilters
