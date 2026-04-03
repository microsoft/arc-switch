<#
.SYNOPSIS
    Resolves the switch password from Azure Key Vault or environment variable.
.DESCRIPTION
    Tries to fetch the password from Azure Key Vault first (vault: azurestack-network,
    secret: Net-Admin). Falls back to the specified environment variable if Key Vault
    is unavailable (e.g., az CLI not installed or not logged in).

    The resolved password is cached in the environment variable for the duration of
    the session, so Key Vault is only queried once.
.PARAMETER EnvVarName
    The environment variable name to check/cache the password in.
.OUTPUTS
    Returns the password string.
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$EnvVarName
)

$ErrorActionPreference = "Stop"

$vaultName = "azurestack-network"
$secretName = "Net-Admin"

# Return cached value if the env var is already set
$cached = [Environment]::GetEnvironmentVariable($EnvVarName)
if ($cached) {
    return $cached
}

# Try Azure Key Vault
try {
    $azCmd = Get-Command az -ErrorAction SilentlyContinue
    if ($azCmd) {
        $secret = az keyvault secret show --vault-name $vaultName --name $secretName --query "value" -o tsv 2>$null
        if ($LASTEXITCODE -eq 0 -and $secret) {
            # Cache in the environment variable for the rest of the session
            [Environment]::SetEnvironmentVariable($EnvVarName, $secret, "Process")
            return $secret
        }
    }
}
catch {
    # Key Vault unavailable — fall through to error
}

Write-Error @"
Password not found. Either:
  1. Login to Azure CLI ('az login') to fetch from Key Vault ($vaultName/$secretName), or
  2. Set the $EnvVarName environment variable manually.
"@
exit 1
