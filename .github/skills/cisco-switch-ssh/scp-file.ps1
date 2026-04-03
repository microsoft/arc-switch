<#
.SYNOPSIS
    Copies a file to or from the Cisco NX-OS switch over SCP.
.DESCRIPTION
    Uses SSH_ASKPASS to handle keyboard-interactive authentication automatically.
    Password is fetched from Azure Key Vault (azurestack-network/Net-Admin),
    falling back to the SWITCH_PASSWORD environment variable.
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

# Resolve connection details
$switchHost = if ($env:SWITCH_SSH_HOST) { $env:SWITCH_SSH_HOST } else { "100.71.34.149" }
$switchUser = if ($env:SWITCH_SSH_USER) { $env:SWITCH_SSH_USER } else { "camilose" }

# Resolve password from Key Vault or environment variable
$resolveScript = Join-Path $PSScriptRoot "..\resolve-password.ps1"
$password = & $resolveScript -EnvVarName "SWITCH_PASSWORD"

# Locate Git's SCP
$gitScp = "C:\Program Files\Git\usr\bin\scp.exe"
if (-not (Test-Path $gitScp)) {
    $gitScp = (Get-Command scp -ErrorAction SilentlyContinue).Source
    if (-not $gitScp) {
        Write-Error "Cannot find scp.exe. Install Git for Windows or OpenSSH."
        exit 1
    }
}

# Build the SSH_ASKPASS helper
$askpassPath = Join-Path $env:TEMP "copilot_switch_askpass_$PID.cmd"
Set-Content -Path $askpassPath -Value "@echo $password" -NoNewline

try {
    $env:SSH_ASKPASS = $askpassPath
    $env:SSH_ASKPASS_REQUIRE = "force"
    $env:DISPLAY = "dummy"

    $scpArgs = @(
        "-O",
        "-o", "StrictHostKeyChecking=no",
        "-o", "UserKnownHostsFile=NUL",
        "-o", "PreferredAuthentications=keyboard-interactive",
        "-o", "PubkeyAuthentication=no",
        "-o", "ConnectTimeout=$TimeoutSeconds",
        "-o", "LogLevel=ERROR"
    )

    $remoteSpec = "${switchUser}@${switchHost}:${RemotePath}"

    if ($Direction -eq "upload") {
        if (-not (Test-Path $LocalPath)) {
            Write-Error "Local file not found: $LocalPath"
            exit 1
        }
        Write-Host "Uploading $LocalPath -> ${switchHost}:${RemotePath}"
        & $gitScp @scpArgs $LocalPath $remoteSpec 2>&1
    }
    else {
        Write-Host "Downloading ${switchHost}:${RemotePath} -> $LocalPath"
        & $gitScp @scpArgs $remoteSpec $LocalPath 2>&1
    }

    if ($LASTEXITCODE -ne 0) {
        Write-Error "SCP failed with exit code $LASTEXITCODE"
        exit $LASTEXITCODE
    }

    Write-Host "Transfer complete."
}
finally {
    Remove-Item -Path $askpassPath -Force -ErrorAction SilentlyContinue
}
