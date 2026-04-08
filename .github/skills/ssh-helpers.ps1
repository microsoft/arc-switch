<#
.SYNOPSIS
    Shared SSH/SCP helper functions for Copilot skills.
.DESCRIPTION
    Provides reusable functions for SSH command execution and SCP file transfer
    with ASKPASS-based authentication. Used by cisco-switch-ssh, sonic-switch-ssh,
    and arc-onboarding skills to avoid code duplication.

    Import with:  . (Join-Path $PSScriptRoot "..\ssh-helpers.ps1")
#>

# ─────────────────────────────────────────────────────────────────────────────
# Binary resolution
# ─────────────────────────────────────────────────────────────────────────────

function Find-SshBinary {
    $path = "C:\Program Files\Git\usr\bin\ssh.exe"
    if (Test-Path $path) { return $path }
    $found = (Get-Command ssh -ErrorAction SilentlyContinue).Source
    if ($found) { return $found }
    throw "Cannot find ssh.exe. Install Git for Windows or OpenSSH."
}

function Find-ScpBinary {
    $path = "C:\Program Files\Git\usr\bin\scp.exe"
    if (Test-Path $path) { return $path }
    $found = (Get-Command scp -ErrorAction SilentlyContinue).Source
    if ($found) { return $found }
    throw "Cannot find scp.exe. Install Git for Windows or OpenSSH."
}

# ─────────────────────────────────────────────────────────────────────────────
# ASKPASS helpers
# ─────────────────────────────────────────────────────────────────────────────

function New-AskPassFile {
    param([Parameter(Mandatory)][string]$Password, [string]$Label = "ssh")
    $path = Join-Path $env:TEMP "copilot_${Label}_askpass_$PID.cmd"
    Set-Content -Path $path -Value "@echo $Password" -NoNewline
    $env:SSH_ASKPASS = $path
    $env:SSH_ASKPASS_REQUIRE = "force"
    $env:DISPLAY = "dummy"
    return $path
}

function Remove-AskPassFile {
    param([string]$Path)
    if ($Path) { Remove-Item -Path $Path -Force -ErrorAction SilentlyContinue }
}

# ─────────────────────────────────────────────────────────────────────────────
# SSH output filtering (common banner noise across all vendors)
# ─────────────────────────────────────────────────────────────────────────────

# Patterns present on all switch types
$script:CommonNoisePatterns = @(
    '^\*\* WARNING',
    'post-quantum',
    'store now, decrypt later',
    'server may need to be upgraded',
    'NOTICE',
    'Unauthorized access',
    'subject to monitoring',
    'Permanently added'
)

function Remove-SshNoise {
    <#
    .SYNOPSIS
        Filters common SSH banner noise from output lines.
    .PARAMETER Output
        Raw SSH output (string array or error records from 2>&1).
    .PARAMETER ExtraPatterns
        Additional vendor-specific regex patterns to filter out.
    #>
    param(
        [Parameter(ValueFromPipeline)]$Output,
        [string[]]$ExtraPatterns = @()
    )
    begin { $allPatterns = $script:CommonNoisePatterns + $ExtraPatterns }
    process {
        foreach ($item in $Output) {
            $line = $item.ToString()
            $skip = $false
            foreach ($p in $allPatterns) {
                if ($line -match $p) { $skip = $true; break }
            }
            if (-not $skip) { $line }
        }
    }
}

# ─────────────────────────────────────────────────────────────────────────────
# SSH command execution
# ─────────────────────────────────────────────────────────────────────────────

function Invoke-SshCommand {
    <#
    .SYNOPSIS
        Executes a command on a remote host over SSH with ASKPASS authentication.
    .PARAMETER User
        SSH username.
    .PARAMETER HostName
        SSH hostname or IP.
    .PARAMETER Password
        SSH password.
    .PARAMETER Command
        Command string to execute on the remote host.
    .PARAMETER AuthMethods
        SSH PreferredAuthentications value. Default: "keyboard-interactive,password".
    .PARAMETER Timeout
        Connection timeout in seconds. Default: 30.
    .PARAMETER ExtraFilterPatterns
        Additional regex patterns to filter from output (vendor-specific banners).
    #>
    param(
        [Parameter(Mandatory)][string]$User,
        [Parameter(Mandatory)][string]$HostName,
        [Parameter(Mandatory)][string]$Password,
        [Parameter(Mandatory)][string]$Command,
        [string]$AuthMethods = "keyboard-interactive,password",
        [int]$Timeout = 30,
        [string[]]$ExtraFilterPatterns = @()
    )

    $sshExe = Find-SshBinary
    $askpass = New-AskPassFile -Password $Password -Label "ssh"

    try {
        $sshArgs = @(
            "-o", "StrictHostKeyChecking=no",
            "-o", "UserKnownHostsFile=NUL",
            "-o", "PreferredAuthentications=$AuthMethods",
            "-o", "PubkeyAuthentication=no",
            "-o", "IdentitiesOnly=yes",
            "-o", "ConnectTimeout=$Timeout",
            "-o", "LogLevel=ERROR",
            "$User@$HostName",
            $Command
        )

        $output = & $sshExe @sshArgs 2>&1
        $exitCode = $LASTEXITCODE

        $filtered = $output | Remove-SshNoise -ExtraPatterns $ExtraFilterPatterns
        $filtered

        if ($exitCode -ne 0) {
            Write-Error "SSH command failed with exit code $exitCode"
            exit $exitCode
        }
    }
    finally {
        Remove-AskPassFile -Path $askpass
    }
}

# ─────────────────────────────────────────────────────────────────────────────
# SCP file transfer
# ─────────────────────────────────────────────────────────────────────────────

function Send-ScpFile {
    <#
    .SYNOPSIS
        Transfers a file to/from a remote host over SCP with ASKPASS authentication.
    .PARAMETER User
        SSH username.
    .PARAMETER HostName
        SSH hostname or IP.
    .PARAMETER Password
        SSH password.
    .PARAMETER LocalPath
        Local file path.
    .PARAMETER RemotePath
        Remote file path on the host.
    .PARAMETER Direction
        "upload" (local→remote) or "download" (remote→local).
    .PARAMETER AuthMethods
        SSH PreferredAuthentications value. Default: "keyboard-interactive,password".
    .PARAMETER Timeout
        Connection timeout in seconds. Default: 120.
    #>
    param(
        [Parameter(Mandatory)][string]$User,
        [Parameter(Mandatory)][string]$HostName,
        [Parameter(Mandatory)][string]$Password,
        [Parameter(Mandatory)][string]$LocalPath,
        [Parameter(Mandatory)][string]$RemotePath,
        [Parameter(Mandatory)][ValidateSet("upload", "download")][string]$Direction,
        [string]$AuthMethods = "keyboard-interactive,password",
        [int]$Timeout = 120
    )

    $scpExe = Find-ScpBinary
    $askpass = New-AskPassFile -Password $Password -Label "scp"

    try {
        $scpArgs = @(
            "-O",
            "-o", "StrictHostKeyChecking=no",
            "-o", "UserKnownHostsFile=NUL",
            "-o", "PreferredAuthentications=$AuthMethods",
            "-o", "PubkeyAuthentication=no",
            "-o", "IdentitiesOnly=yes",
            "-o", "ConnectTimeout=$Timeout",
            "-o", "LogLevel=ERROR"
        )

        $remoteSpec = "${User}@${HostName}:${RemotePath}"

        if ($Direction -eq "upload") {
            if (-not (Test-Path $LocalPath)) {
                throw "Local file not found: $LocalPath"
            }
            Write-Host "Uploading $LocalPath -> ${HostName}:${RemotePath}"
            & $scpExe @scpArgs $LocalPath $remoteSpec 2>&1
        }
        else {
            Write-Host "Downloading ${HostName}:${RemotePath} -> $LocalPath"
            & $scpExe @scpArgs $remoteSpec $LocalPath 2>&1
        }

        if ($LASTEXITCODE -ne 0) {
            throw "SCP failed with exit code $LASTEXITCODE"
        }

        Write-Host "Transfer complete."
    }
    finally {
        Remove-AskPassFile -Path $askpass
    }
}
