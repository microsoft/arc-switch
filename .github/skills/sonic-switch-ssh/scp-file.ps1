<#
.SYNOPSIS
    Copies a file to or from a SONiC/Dell OS10 switch over SCP.
.DESCRIPTION
    Uses SSH_ASKPASS to handle password authentication automatically.
    Password is fetched from Azure Key Vault (azurestack-network/Net-Admin),
    falling back to the SONIC_PASSWORD environment variable.
.PARAMETER LocalPath
    The local file path (source for upload, destination for download).
.PARAMETER RemotePath
    The remote file path on the switch.
.PARAMETER Direction
    "upload" to copy local -> switch, "download" to copy switch -> local.
.PARAMETER TimeoutSeconds
    SCP connection timeout in seconds. Default: 120.
.EXAMPLE
    .\scp-file.ps1 -LocalPath ".\gnmi-collector" -RemotePath "/usr/temp/gnmi-collector" -Direction upload
.EXAMPLE
    .\scp-file.ps1 -LocalPath ".\gnmi-collector" -RemotePath "/usr/temp/gnmi-collector" -Direction download
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$LocalPath,

    [Parameter(Mandatory = $true)]
    [string]$RemotePath,

    [Parameter(Mandatory = $true)]
    [ValidateSet("upload", "download")]
    [string]$Direction,

    [Parameter(Mandatory = $false)]
    [int]$TimeoutSeconds = 120
)

$ErrorActionPreference = "Stop"

# Import shared helpers
. (Join-Path $PSScriptRoot "..\ssh-helpers.ps1")

# Resolve connection details
$switchHost = if ($env:SONIC_SSH_HOST) { $env:SONIC_SSH_HOST } else { "100.100.47.95" }
$switchUser = if ($env:SONIC_SSH_USER) { $env:SONIC_SSH_USER } else { "admin" }

# Resolve password from Key Vault or environment variable
$resolveScript = Join-Path $PSScriptRoot "..\resolve-password.ps1"
$password = & $resolveScript -EnvVarName "SONIC_PASSWORD"

Send-ScpFile `
    -User $switchUser `
    -HostName $switchHost `
    -Password $password `
    -LocalPath $LocalPath `
    -RemotePath $RemotePath `
    -Direction $Direction `
    -Timeout $TimeoutSeconds
