<#
.SYNOPSIS
    Runs a command on a SONiC/Dell OS10 switch over SSH.
.DESCRIPTION
    Uses SSH_ASKPASS to handle password authentication automatically.
    Password is fetched from Azure Key Vault (azurestack-network/Net-Admin),
    falling back to the SONIC_PASSWORD environment variable.
    The switch IP and user are read from SONIC_SSH_HOST / SONIC_SSH_USER env vars,
    falling back to defaults (100.100.47.95 / admin).
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
    .\ssh-command.ps1 -Command "ls -la /home/admin/gnmi-collector"
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$Command,

    [Parameter(Mandatory = $false)]
    [int]$TimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"

# Import shared helpers
. (Join-Path $PSScriptRoot "..\ssh-helpers.ps1")

# Resolve connection details from environment (with defaults)
$switchHost = if ($env:SONIC_SSH_HOST) { $env:SONIC_SSH_HOST } else { "100.100.47.95" }
$switchUser = if ($env:SONIC_SSH_USER) { $env:SONIC_SSH_USER } else { "admin" }

# Resolve password from Key Vault or environment variable
$resolveScript = Join-Path $PSScriptRoot "..\resolve-password.ps1"
$password = & $resolveScript -EnvVarName "SONIC_PASSWORD"

Invoke-SshCommand `
    -User $switchUser `
    -HostName $switchHost `
    -Password $password `
    -Command $Command `
    -Timeout $TimeoutSeconds
