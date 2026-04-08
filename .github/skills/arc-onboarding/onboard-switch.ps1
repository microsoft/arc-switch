<#
.SYNOPSIS
    Onboards a network switch to Azure Arc with gNMI telemetry collection.
.DESCRIPTION
    Automates the full onboarding flow:
      1. Auto-detects vendor (Cisco NX-OS or Dell SONiC)
      2. Resolves Azure parameters (subscription, tenant, workspace)
      3. Fills in the setup script template
      4. Uploads and executes the script on the switch
      5. Validates services are running
      6. Outputs the azcmagent connect command for device-code auth

    The setup script templates are read from:
      Docs/arcnet_onboarding_instructions/Arcnet_Cisco_gNMI_Setup
      Docs/arcnet_onboarding_instructions/Arcnet_Sonic_gNMI_Setup
.PARAMETER SwitchIP
    IP address of the switch to onboard.
.PARAMETER Vendor
    Switch vendor: "cisco" or "sonic". Auto-detected if not specified.
.PARAMETER SshUser
    SSH username. Defaults to "admin".
.PARAMETER SshPassword
    SSH password. Resolved from Key Vault if not specified.
.PARAMETER MachineName
    Azure Arc machine name. Auto-detected from switch hostname if not specified.
.PARAMETER ResourceGroup
    Azure resource group. Defaults to "ARCNET".
.PARAMETER Region
    Azure region. Defaults to "eastus".
.PARAMETER DryRun
    Generate the filled-in script locally without uploading or executing.
.EXAMPLE
    .\onboard-switch.ps1 -SwitchIP "10.0.0.1"
.EXAMPLE
    .\onboard-switch.ps1 -SwitchIP "10.0.0.1" -Vendor cisco -DryRun
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$SwitchIP,

    [Parameter(Mandatory = $false)]
    [ValidateSet("cisco", "sonic")]
    [string]$Vendor,

    [Parameter(Mandatory = $false)]
    [string]$SshUser = "admin",

    [Parameter(Mandatory = $false)]
    [string]$SshPassword,

    [Parameter(Mandatory = $false)]
    [string]$MachineName,

    [Parameter(Mandatory = $false)]
    [string]$ResourceGroup = "ARCNET",

    [Parameter(Mandatory = $false)]
    [string]$Region = "eastus",

    [Parameter(Mandatory = $false)]
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..")).Path

# Import shared helpers
. (Join-Path $PSScriptRoot "..\ssh-helpers.ps1")

# ─────────────────────────────────────────────────────────────────────────────
# Helper: resolve SSH password (extends shared resolve-password.ps1 with fallbacks)
# ─────────────────────────────────────────────────────────────────────────────
function Resolve-SshPassword {
    if ($SshPassword) { return $SshPassword }

    $resolveScript = Join-Path $PSScriptRoot "..\resolve-password.ps1"
    if (Test-Path $resolveScript) {
        try {
            $pw = & $resolveScript -EnvVarName "SWITCH_PASSWORD" 2>$null
            if ($pw) { return $pw }
        } catch {}
    }

    if ($env:SWITCH_PASSWORD) { return $env:SWITCH_PASSWORD }
    if ($env:SONIC_PASSWORD)  { return $env:SONIC_PASSWORD }
    if ($env:GNMI_PASS)       { return $env:GNMI_PASS }

    throw "No SSH password found. Provide -SshPassword, set SWITCH_PASSWORD env var, or ensure az login for Key Vault."
}

# Cisco-specific banner patterns
$ciscoFilters = @('^hostname ', '^BuildVersion:')

# Wrapper that joins output into a single string (for variable assignment)
function Invoke-SwitchSsh {
    param([string]$Command, [int]$Timeout = 30)
    $extra = if ($Vendor -eq "cisco") { $ciscoFilters } else { @() }
    $result = Invoke-SshCommand `
        -User $SshUser `
        -HostName $SwitchIP `
        -Password $script:password `
        -Command $Command `
        -Timeout $Timeout `
        -ExtraFilterPatterns $extra
    return ($result -join "`n")
}

# ═════════════════════════════════════════════════════════════════════════════
# STEP 1: Resolve SSH password
# ═════════════════════════════════════════════════════════════════════════════

Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Azure Arc Switch Onboarding — $SwitchIP" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

Write-Host "[1/8] Resolving SSH credentials..." -ForegroundColor Yellow
$script:password = Resolve-SshPassword
Write-Host "  SSH user: $SshUser, password: ****$($script:password.Substring([Math]::Max(0, $script:password.Length - 3)))"

# ═════════════════════════════════════════════════════════════════════════════
# STEP 2: Detect vendor (if not specified)
# ═════════════════════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "[2/8] Detecting switch vendor..." -ForegroundColor Yellow

if (-not $Vendor) {
    # Try SONiC first (simpler — just run uname + check for sonic)
    $osInfo = Invoke-SwitchSsh -Command "cat /etc/os-release 2>/dev/null || echo UNKNOWN"

    if ($osInfo -match "sonic|SONiC|Enterprise_SONiC") {
        $Vendor = "sonic"
    }
    elseif ($osInfo -match "NX-OS|Nexus|Cisco") {
        $Vendor = "cisco"
    }
    else {
        # Try NX-OS detection via "run bash" prefix
        $nxosCheck = Invoke-SwitchSsh -Command "run bash cat /etc/os-release 2>/dev/null || echo UNKNOWN"
        if ($nxosCheck -match "NX-OS|Nexus|Cisco|wrlinux") {
            $Vendor = "cisco"
        }
        else {
            throw "Could not auto-detect vendor. Use -Vendor cisco or -Vendor sonic."
        }
    }
}

Write-Host "  Vendor: $Vendor" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════════════════
# STEP 3: Get switch hostname
# ═════════════════════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "[3/8] Getting switch hostname..." -ForegroundColor Yellow

if (-not $MachineName) {
    if ($Vendor -eq "cisco") {
        $hostname = Invoke-SwitchSsh -Command "run bash hostname"
    }
    else {
        $hostname = Invoke-SwitchSsh -Command "hostname"
    }
    $MachineName = ($hostname -split "`n" | Select-Object -Last 1).Trim()
    if (-not $MachineName -or $MachineName -eq "UNKNOWN") {
        $MachineName = "switch-$SwitchIP" -replace '\.', '-'
    }
}

Write-Host "  Machine name: $MachineName" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════════════════
# STEP 4: Resolve Azure parameters
# ═════════════════════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "[4/8] Resolving Azure parameters..." -ForegroundColor Yellow

# Subscription ID
$subscriptionId = $env:SUBSCRIPTION_ID
if (-not $subscriptionId) {
    try {
        $acct = az account show --query "{sub:id,tenant:tenantId}" -o json 2>$null | ConvertFrom-Json
        $subscriptionId = $acct.sub
    } catch {}
}
if (-not $subscriptionId) { throw "Cannot resolve Subscription ID. Set SUBSCRIPTION_ID env var or run 'az login'." }

# Tenant ID
$tenantId = $env:TENANT_ID
if (-not $tenantId) {
    try {
        if (-not $acct) { $acct = az account show --query "{sub:id,tenant:tenantId}" -o json 2>$null | ConvertFrom-Json }
        $tenantId = $acct.tenant
    } catch {}
}
if (-not $tenantId) { throw "Cannot resolve Tenant ID. Set TENANT_ID env var or run 'az login'." }

# Workspace credentials (required from env vars)
$workspaceId = $env:WORKSPACE_ID
$primaryKey  = $env:PRIMARY_KEY
$secondaryKey = if ($env:SECONDARY_KEY) { $env:SECONDARY_KEY } else { "" }

if (-not $workspaceId) { throw "WORKSPACE_ID environment variable is required." }
if (-not $primaryKey)  { throw "PRIMARY_KEY environment variable is required." }

# gNMI credentials (default to SSH credentials)
$gnmiUser = if ($env:GNMI_USER) { $env:GNMI_USER } else { $SshUser }
$gnmiPass = if ($env:GNMI_PASS) { $env:GNMI_PASS } else { $script:password }

Write-Host "  Subscription: $($subscriptionId.Substring(0, 8))..." -ForegroundColor Green
Write-Host "  Tenant:       $($tenantId.Substring(0, 8))..." -ForegroundColor Green
Write-Host "  Workspace:    $($workspaceId.Substring(0, 8))..." -ForegroundColor Green
Write-Host "  Resource Group: $ResourceGroup" -ForegroundColor Green
Write-Host "  Region:       $Region" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════════════════
# STEP 5: Generate the setup script
# ═════════════════════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "[5/8] Generating setup script..." -ForegroundColor Yellow

$docsDir = Join-Path $repoRoot "Docs\arcnet_onboarding_instructions"
if ($Vendor -eq "cisco") {
    $templatePath = Join-Path $docsDir "Arcnet_Cisco_gNMI_Setup"
} else {
    $templatePath = Join-Path $docsDir "Arcnet_Sonic_gNMI_Setup"
}

if (-not (Test-Path $templatePath)) {
    throw "Setup script template not found: $templatePath"
}

$script = Get-Content $templatePath -Raw

# Replace all placeholder values
$replacements = @{
    '<YOUR_AZURE_REGION>'              = $Region
    '<YOUR_RESOURCE_GROUP>'            = $ResourceGroup
    '<YOUR_SUBSCRIPTION_ID>'           = $subscriptionId
    '<YOUR_SWITCH_HOSTNAME>'           = $MachineName
    '<YOUR_TENANT_ID>'                 = $tenantId
    '<LOG_ANALYTICS_WORKSPACE_ID>'     = $workspaceId
    '<LOG_ANALYTICS_PRIMARY_KEY>'      = $primaryKey
    '<LOG_ANALYTICS_SECONDARY_KEY>'    = $secondaryKey
    '<NX-OS_USERNAME>'                 = $gnmiUser
    '<NX-OS_PASSWORD>'                 = $gnmiPass
    '<SONIC_ADMIN_USERNAME>'           = $gnmiUser
    '<SONIC_ADMIN_PASSWORD>'           = $gnmiPass
}

foreach ($key in $replacements.Keys) {
    $script = $script.Replace($key, $replacements[$key])
}

# Verify no unfilled placeholders remain
$remaining = [regex]::Matches($script, '<[A-Z_]+>') | ForEach-Object { $_.Value } | Sort-Object -Unique
if ($remaining.Count -gt 0) {
    Write-Warning "Unfilled placeholders remain: $($remaining -join ', ')"
}

# Write to temp file
$tempScript = Join-Path $env:TEMP "arcnet-setup-$Vendor-$($SwitchIP -replace '\.', '-').sh"
Set-Content -Path $tempScript -Value $script -NoNewline -Encoding utf8
Write-Host "  Generated: $tempScript" -ForegroundColor Green
Write-Host "  Vendor:    $Vendor ($((Get-Item $tempScript).Length / 1KB |ForEach-Object { '{0:N0} KB' -f $_ }))" -ForegroundColor Green

# ═════════════════════════════════════════════════════════════════════════════
# DRY-RUN EXIT POINT
# ═════════════════════════════════════════════════════════════════════════════

if ($DryRun) {
    Write-Host ""
    Write-Host "═══ DRY-RUN MODE ═══" -ForegroundColor Magenta
    Write-Host "Setup script generated but NOT executed." -ForegroundColor Magenta
    Write-Host "Script saved to: $tempScript" -ForegroundColor Magenta
    Write-Host ""
    Write-Host "To execute manually:" -ForegroundColor Yellow
    Write-Host "  1. SCP to switch:  scp $tempScript ${SshUser}@${SwitchIP}:/tmp/arcnet-setup.sh"
    Write-Host "  2. SSH and run:    ssh ${SshUser}@${SwitchIP}"
    if ($Vendor -eq "cisco") {
        Write-Host "     run bash"
        Write-Host "     sudo su -"
    } else {
        Write-Host "     sudo su -"
    }
    Write-Host "     bash /tmp/arcnet-setup.sh"
    return
}

# ═════════════════════════════════════════════════════════════════════════════
# STEP 6: Upload and execute the setup script
# ═════════════════════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "[6/8] Uploading and executing setup script..." -ForegroundColor Yellow

if ($Vendor -eq "cisco") {
    $remotePath = "/tmp/arcnet-setup.sh"
} else {
    $remotePath = "/tmp/arcnet-setup.sh"
}

# Upload
Write-Host "  Uploading to ${SwitchIP}:${remotePath}..."
Send-ScpFile `
    -User $SshUser `
    -HostName $SwitchIP `
    -Password $script:password `
    -LocalPath $tempScript `
    -RemotePath $remotePath `
    -Direction "upload"
Write-Host "  Upload complete." -ForegroundColor Green

# Execute
Write-Host "  Executing setup script (this may take a few minutes)..."
if ($Vendor -eq "cisco") {
    # Cisco: need to copy from bootflash to /tmp, then run as root via bash
    $execOutput = Invoke-SwitchSsh -Command "run bash sudo bash -c 'cp /bootflash$remotePath $remotePath 2>/dev/null; chmod +x $remotePath; bash $remotePath'" -Timeout 300
} else {
    $execOutput = Invoke-SwitchSsh -Command "sudo bash $remotePath" -Timeout 300
}

Write-Host $execOutput

# ═════════════════════════════════════════════════════════════════════════════
# STEP 7: Validate gNMI collector
# ═════════════════════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "[7/8] Validating gNMI collector..." -ForegroundColor Yellow

if ($Vendor -eq "cisco") {
    $statusOutput = Invoke-SwitchSsh -Command "run bash /etc/init.d/gnmi-collectord status 2>&1"
} else {
    $statusOutput = Invoke-SwitchSsh -Command "systemctl status gnmi-collector --no-pager 2>&1"
}
Write-Host "  gNMI collector: $statusOutput"

# Check if azcmagent exists
if ($Vendor -eq "cisco") {
    $arcCheck = Invoke-SwitchSsh -Command "run bash azcmagent version 2>&1 || echo NOT_INSTALLED"
} else {
    $arcCheck = Invoke-SwitchSsh -Command "azcmagent version 2>&1 || echo NOT_INSTALLED"
}
Write-Host "  Arc agent: $arcCheck"

# Cleanup temp file
Remove-Item -Path $tempScript -Force -ErrorAction SilentlyContinue

# ═════════════════════════════════════════════════════════════════════════════
# STEP 8: Device Code Flow instructions (MANUAL STEP)
# ═════════════════════════════════════════════════════════════════════════════

Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Phase 1 Complete — Arc agent + gNMI collector installed" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""
Write-Host "╔═══════════════════════════════════════════════════════════╗" -ForegroundColor Yellow
Write-Host "║  ACTION REQUIRED: Device Code Flow (DCF)                  ║" -ForegroundColor Yellow
Write-Host "║                                                           ║" -ForegroundColor Yellow
Write-Host "║  azcmagent connect requires interactive browser sign-in.  ║" -ForegroundColor Yellow
Write-Host "║  This step cannot be automated.                           ║" -ForegroundColor Yellow
Write-Host "╚═══════════════════════════════════════════════════════════╝" -ForegroundColor Yellow
Write-Host ""
Write-Host "  1. SSH into the switch:" -ForegroundColor White

if ($Vendor -eq "cisco") {
    Write-Host "     ssh ${SshUser}@${SwitchIP}" -ForegroundColor White
    Write-Host "     run bash" -ForegroundColor White
    Write-Host "     sudo su -" -ForegroundColor White
} else {
    Write-Host "     ssh ${SshUser}@${SwitchIP}" -ForegroundColor White
    Write-Host "     sudo su -" -ForegroundColor White
}

Write-Host ""
Write-Host "  2. Run this command:" -ForegroundColor White
Write-Host ""
Write-Host "     azcmagent connect \" -ForegroundColor Green
Write-Host "       --resource-group `"$ResourceGroup`" \" -ForegroundColor Green
Write-Host "       --tenant-id `"$tenantId`" \" -ForegroundColor Green
Write-Host "       --location `"$Region`" \" -ForegroundColor Green
Write-Host "       --subscription-id `"$subscriptionId`"" -ForegroundColor Green
Write-Host ""
Write-Host "  3. The command will display a URL and a code." -ForegroundColor White
Write-Host "     Open the URL in your browser, enter the code," -ForegroundColor White
Write-Host "     and sign in with your Azure credentials." -ForegroundColor White
Write-Host ""
Write-Host "  4. After DCF completes, verify on the switch:" -ForegroundColor White
Write-Host "     azcmagent show" -ForegroundColor Green
Write-Host "     (Status should show 'Connected')" -ForegroundColor DarkGray
Write-Host ""
Write-Host "Once DCF is complete, tell me to continue and I will" -ForegroundColor Cyan
Write-Host "validate the full setup (Arc connection + gNMI collector)." -ForegroundColor Cyan
Write-Host ""
