# Azure Arc for Cisco Nexus Switches

## Overview

This project enables Azure Arc management and monitoring for Cisco Nexus switches running NX-OS. By onboarding Cisco switches to Azure Arc, you can:

- Centralized Management: Manage network devices alongside other Azure resources
- Monitoring & Telemetry: Collect and analyze switch telemetry in Azure Log Analytics
- Dashboards & Alerts: Create custom dashboards and alerts in Azure or Grafana
- Compliance & Governance: Apply Azure policies and track configuration changes
- Hybrid Infrastructure: Unified view of cloud and on-premises network devices

## Architecture

The solution consists of several components:

1. Azure Arc Agent - Connects the switch to Azure and provides identity
2. Azure Arc Services - Himdsd, Arcproxyd, Extd, GCAD and GAS manages security, policies and extensions
3. Parsers - Extracts structured data from Cisco show commands
4. Azure Logger - Sends parsed telemetry to Log Analytics workspace
5. Cron-based Collector - Runs every minute to gather and send metrics

### Data Flow

```
Cisco Switch (NX-OS)
    ├─> vsh commands (show interface, show inventory, etc.)
    ├─> Cisco Parser (converts text to JSON)
    ├─> Azure Logger (adds metadata, signs requests)
    └─> Azure Log Analytics Workspace
```

## Prerequisites

### Azure Requirements

- Azure Subscription with appropriate permissions
- Resource Group for Arc-enabled devices
- Log Analytics Workspace for telemetry storage
- Network connectivity from switch to Azure endpoints

### Switch Requirements

- Cisco Nexus Switch running NX-OS
- Bash access to the switch operating system
- Internet connectivity to download binaries and connect to Azure
- Root/sudo access for installation
- Minimum RAM and Storage as per Azure Local documentation

### Network Requirements

The switch must have outbound connectivity to:
- Azure Arc services (`*.his.arc.azure.com`)
- Azure Log Analytics (`*.ods.opinsights.azure.com`)
- GitHub releases (for parser download)

## Installation Guide

### Step 1: Prepare Azure Resources

#### 1.1 Create Log Analytics Workspace

Navigate to Azure Portal and create a new Log Analytics workspace:

```bash
# Using Azure CLI
az monitor log-analytics workspace create \
  --resource-group "ARCNET" \
  --workspace-name "CiscoSwitchLogs" \
  --location "eastus"
```

Via Azure Portal:
1. Go to Azure Portal → Create a resource
2. Search for "Log Analytics Workspace"
3. Fill in the details:
   - Subscription: Your subscription
   - Resource Group: Create new or use existing (e.g., "ARCNET")
   - Name: CiscoSwitchLogs
   - Region: Select your region (e.g., East US)
4. Click "Review + Create"

#### 1.2 Get Log Analytics Credentials

You'll need three values from your workspace:

Get Workspace ID:
```bash
az monitor log-analytics workspace show \
  --resource-group "ARCNET" \
  --workspace-name "CiscoSwitchLogs" \
  --query "customerId" -o tsv
```

Get Primary and Secondary Keys:
```bash
az monitor log-analytics workspace get-shared-keys \
  --resource-group "ARCNET" \
  --workspace-name "CiscoSwitchLogs"
```

### Step 2: Configure the Installation Script

1. Download the `ArcNet_Cisco_Arc_Setup.sh` script which is a part of this folder
2. Edit the configuration section at the top:

```bash
# Azure Configuration
REGION="eastus"                                    # Your Azure region
RESOURCE_GROUP="ARCNET"                           # Your resource group name
SUBSCRIPTION_ID="<YOUR_SUBSCRIPTION_ID>"          # From Azure Portal
MACHINE_NAME="<SWITCH_HOSTNAME>"                  # Unique identifier for switch
TENANT_ID="<YOUR_TENANT_ID>"                      # From Azure Portal

# Log Analytics Workspace Configuration
WORKSPACE_ID="<FROM_STEP_1.2>"                   # Workspace ID from Step 1.2
PRIMARY_KEY="<FROM_STEP_1.2>"                    # Primary key from Step 1.2
SECONDARY_KEY="<FROM_STEP_1.2>"                  # Secondary key from Step 1.2
```

### Step 3: Execute the Setup Script on the Switch

#### 3.1 Access the Switch

SSH into your Cisco Nexus switch:

```bash
ssh admin@<switch-ip-address>
```

#### 3.2 Enter Bash and Become Root

```bash
# Enter bash shell
run bash

# Become root
sudo su -
```

#### 3.3 Run the Setup Script

Copy the entire contents of the cisco-arc-setup.sh script and paste it directly into the bash shell on the switch (where you are logged in as root).

The script will execute and perform the following:
- Install and configure Arc agent binaries
- Set up systemd shim services
- Configure environment variables
- Create init scripts for all services
- Download and install the Cisco parser
- Set up Azure logging infrastructure
- Configure cron job for telemetry collection

Expected output:
```
Stopping services...
Starting services...
HIMDS is running (PID 12345)
ArcProxy is running (PID 12346)
EXTD is running (PID 12347)
GCAD is running (PID 12348)
Setup complete. Check service status:
...
Azure Arc Setup Script Completed Successfully!

NEXT STEPS:
1. Connect the Arc agent to Azure (see README.md for instructions)"
2. Verify Arc agent connection with: azcmagent show"
3. Check service status with: /etc/init.d/himdsd status"
4. View logs in /var/log/arc/"
```

### Step 4: Connect Arc Agent to Azure

After the setup script completes, you need to connect the agent to Azure.

#### 4.1 Get the Connection Script from Azure Portal

1. Go to Azure Portal → Azure Arc → Machines
2. Click + Add/Create → Add a machine
3. Select Add a single server
4. Choose your subscription and resource group
5. Select region and operating system: Linux
6. Under "Connectivity method", choose Public endpoint
7. Deselect "Enable Azure SQL extension deployment"
8. Click Generate script

The portal will generate a script similar to this:

```bash
export subscriptionId="<YOUR_SUBSCRIPTION_ID>";
export resourceGroup="ARCNET";
export tenantId="<YOUR_TENANT_ID>";
export location="eastus";
export authType="token";
export correlationId="<GENERATED_CORRELATION_ID>";
export cloud="AzureCloud";

LINUX_INSTALL_SCRIPT="/tmp/install_linux_azcmagent.sh"
if [ -f "$LINUX_INSTALL_SCRIPT" ]; then rm -f "$LINUX_INSTALL_SCRIPT"; fi;
output=$(wget https://gbl.his.arc.azure.com/azcmagent-linux -O "$LINUX_INSTALL_SCRIPT" 2>&1);
if [ $? != 0 ]; then 
    wget -qO- --method=PUT --body-data="{\"subscriptionId\":\"$subscriptionId\",\"resourceGroup\":\"$resourceGroup\",\"tenantId\":\"$tenantId\",\"location\":\"$location\",\"correlationId\":\"$correlationId\",\"authType\":\"$authType\",\"operation\":\"onboarding\",\"messageType\":\"DownloadScriptFailed\",\"message\":\"$output\"}" "https://gbl.his.arc.azure.com/log" &> /dev/null || true;
fi;
echo "$output";
bash "$LINUX_INSTALL_SCRIPT";
sleep 5;
sudo azcmagent connect --resource-group "$resourceGroup" --tenant-id "$tenantId" --location "$location" --subscription-id "$subscriptionId" --cloud "$cloud" --tags 'ArcSQLServerExtensionDeployment=Disabled' --correlation-id "$correlationId";
```

#### 4.2 Run the Connection Script

On the switch (still as root in bash):

```bash
# Copy the entire script from Azure Portal and paste it
# The script will download and run the Arc agent installer
```

Note: The script may prompt for authentication. Follow the device code authentication flow:
1. The script will display a URL and code
2. Open the URL in a browser on another device
3. Enter the code when prompted
4. Sign in with your Azure credentials
5. Return to the switch terminal

#### 4.3 Verify Connection

Once connected, verify the agent status:

```bash
# Check if agent is connected
azcmagent show

# Check connection status
azcmagent check

# List agent configuration
azcmagent config list
```

Expected output from azcmagent show:
```
Resource Name     : <MACHINE_NAME>
Resource Group    : ARCNET
Tenant ID         : <YOUR_TENANT_ID>
Subscription ID   : <YOUR_SUBSCRIPTION_ID>
Cloud             : AzureCloud
Location          : eastus
Agent Version     : 1.x.x
Agent Status      : Connected
```

### Step 4.4 Cleanup Steps If needed after lab completion

## Step 4.4.1
```bash
service himdsd stop
service arcproxyd stop
service extd stop
service gcad stop
killall himds 2>/dev/null
killall arcproxy 2>/dev/null
sudo rpm -e azcmagent
```

## Step 4.4.2

Also remove all files related to ArcNet from /opt folder
rm -f cisco-azure-logger-v2.sh
rm -f cisco-parser
.......

Then delete the resource from the portal

### Step 5: Verify Telemetry Collection

#### 5.1 Check Collector is Running

```bash
# Check if cron job is configured
crontab -l | grep cisco-parser-collector

# Check collector logs
tail -f /var/log/cisco-parser-collector.log
```

#### 5.2 Test Manual Data Collection

Test the complete flow for one command:

```bash
# Step 1: Run a show command
vsh -c "show class-map" > /tmp/test-output.txt

# Step 2: Parse the output
/opt/cisco-parser -p class-map -i /tmp/test-output.txt -o /tmp/test-parsed.json

# Step 3: View parsed JSON
cat /tmp/test-parsed.json

# Step 4: Send to Azure
/opt/cisco-azure-logger-v2.sh send CiscoClassMapTest /tmp/test-parsed.json

# Step 5: Clean up
rm /tmp/test-output.txt /tmp/test-parsed.json
```

#### 5.3 Verify Data in Log Analytics

Wait 5-10 minutes for data to appear in Azure, then query your workspace:

1. Go to Azure Portal → Log Analytics workspaces → CiscoSwitchLogs
2. Click on Logs in the left menu
3. Run a test query:

```kusto
// Check for any custom tables
search "*" 
| where TimeGenerated > ago(1h)
| distinct Type

// Query specific table (after "_CL" suffix is added automatically)
CiscoClassMap_CL
| take 10

// Check all switch data
union CiscoClassMap_CL, CiscoInterfaceCounter_CL, CiscoInventory_CL
| where TimeGenerated > ago(1h)
| summarize count() by Type, hostname_s
```

## Telemetry Tables

The following custom tables will be created in your Log Analytics workspace:

| Table Name | Description | Update Frequency |
|------------|-------------|------------------|
| `CiscoBgpSummary_CL` | BGP neighbor summary | Every minute |
| `CiscoClassMap_CL` | QoS class maps configuration | Every minute |
| `CiscoEnvPower_CL` | Power supply status | Every minute |
| `CiscoEnvTemp_CL` | Temperature sensors | Every minute |
| `CiscoInterfaceCounter_CL` | Interface traffic statistics | Every minute |
| `CiscoInterfaceErrors_CL` | Interface error counters | Every minute |
| `CiscoInterfaceStatus_CL` | Interface status (up/down, speed, duplex) | Every minute |
| `CiscoInventory_CL` | Hardware inventory | Every minute |
| `CiscoIpArp_CL` | ARP table entries | Every minute |
| `CiscoIpRoute_CL` | Routing table | Every minute |
| `CiscoLldpNeighbor_CL` | LLDP neighbor information | Every minute |
| `CiscoMacAddress_CL` | MAC address table | Every minute |
| `CiscoSystemResources_CL` | CPU, memory utilization | Every minute |
| `CiscoSystemUptime_CL` | System uptime | Every minute |
| `CiscoTransceiver_CL` | SFP/QSFP module details | Every minute |
| `CiscoVersion_CL` | NX-OS version, hardware info, uptime | Every minute |

Note: Azure automatically appends _CL suffix to custom log table names.

## Service Management

### Check Service Status

```bash
# Check individual services
/etc/init.d/himdsd status      # HIMDS (metadata service)
/etc/init.d/arcproxyd status   # Arc Proxy (extension handler)
/etc/init.d/extd status        # Extension service
/etc/init.d/gcad status        # Guest configuration service

# Check all services at once
for svc in himdsd arcproxyd extd gcad; do
    echo "=== $svc ==="
    /etc/init.d/$svc status
done
```

### Start/Stop Services

```bash
# Start a service
/etc/init.d/himdsd start

# Stop a service
/etc/init.d/himdsd stop

# Restart a service
/etc/init.d/himdsd restart

# Restart all Arc services
for svc in gcad extd arcproxyd himdsd; do
    /etc/init.d/$svc restart
    sleep 5
done
```

### View Service Logs

```bash
# Arc agent logs
tail -f /var/opt/azcmagent/log/himds.log
tail -f /var/opt/azcmagent/log/arcproxy.log

# Guest configuration logs
tail -f /var/lib/GuestConfig/gc_agent_logs/gc_agent.log

```

## Diagnostics & Troubleshooting

### Common Issues

#### Issue: Arc Agent Not Connected

Symptoms:
```bash
azcmagent show
# Error: The machine is not connected to Azure
```

Solution:
1. Check network connectivity to Azure endpoints
2. Verify DNS resolution
3. Re-run the connection script from Azure Portal
4. Check himds service is running: `/etc/init.d/himdsd status`

#### Issue: Services Keep Stopping

Symptoms:
Services show as stopped when checked

Solution:
```bash
# Check for port conflicts
netstat -tulpn | grep -E '40342|40343|40344'

# Kill any conflicting processes
fuser -k 40342/tcp
fuser -k 40343/tcp
fuser -k 40344/tcp

# Restart services
/etc/init.d/himdsd restart
sleep 5
/etc/init.d/arcproxyd restart
```

#### Issue: No Data in Log Analytics

Symptoms:
Queries return no results after 15+ minutes

Solution:
```bash
# Test the logger manually
/opt/cisco-azure-logger-v2.sh test

# Check collector is running
ps aux | grep cisco-parser-collector

# Check cron job
crontab -l

# Review collector logs for errors
tail -100 /var/log/cisco-parser-collector.log

# Verify workspace credentials
grep -E "WORKSPACE_ID|PRIMARY_KEY" /opt/cisco-azure-logger-v2.sh
```

### Diagnostic Commands

```bash
# Complete health check
echo "=== Arc Agent Status ==="
azcmagent show
azcmagent check

echo "=== Service Status ==="
for svc in himdsd arcproxyd extd gcad; do
    /etc/init.d/$svc status
done

echo "=== Network Connectivity ==="
curl -I https://gbl.his.arc.azure.com
curl -I https://${WORKSPACE_ID}.ods.opinsights.azure.com

echo "=== Disk Space ==="
df -h

echo "=== Recent Collector Activity ==="
tail -20 /var/log/cisco-parser-collector.log

echo "=== Cron Status ==="
crontab -l | grep cisco
```

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
# Enable debug logging for Arc agent
azcmagent config set guestconfiguration.enabledebuglogging true

# Restart services to apply
/etc/init.d/extd restart
/etc/init.d/gcad restart

# View debug logs
tail -f /var/lib/GuestConfig/gc_agent_logs/gc_agent.log
```

## Grafana Dashboard Setup

### Overview

To visualize the Cisco switch telemetry data collected in Azure Log Analytics, you can use Grafana with the Azure Monitor data source. This provides real-time dashboards with comprehensive metrics for device health, interface performance, routing, and error tracking.

### Prerequisites for Grafana

- Grafana instance (version 8.0 or higher)
- Azure Monitor data source plugin installed
- Azure service principal or managed identity with read access to Log Analytics workspace

### Step 1: Configure Azure Monitor Data Source in Grafana

1. Log in to your Grafana instance
2. Navigate to Configuration → Data Sources
3. Click "Add data source"
4. Search for and select "Azure Monitor"
5. Configure the data source:

   Connection Details:
   - Name: Give it a descriptive name (e.g., "CiscoSwitchLogs")
   - Authentication: Choose your method (Managed Identity, App Registration, or Azure CLI)

   For App Registration (Service Principal):
   - Directory (Tenant) ID: Your Azure AD tenant ID
   - Application (Client) ID: Your service principal client ID
   - Client Secret: Your service principal secret

   Azure Monitor Details:
   - Subscription: Select your Azure subscription
   - Default Workspace: Select your Log Analytics workspace (CiscoSwitchLogs)

6. Click "Save & Test" to verify the connection

### Step 2: Create Service Principal (if using App Registration)

If you don't have a service principal, create one:

```bash
# Create service principal
az ad sp create-for-rbac --name "GrafanaCiscoMonitoring" --role "Log Analytics Reader" \
  --scopes /subscriptions//resourceGroups/ARCNET/providers/Microsoft.OperationalInsights/workspaces/CiscoSwitchLogs

# Output will include:
# - appId (Client ID)
# - password (Client Secret)
# - tenant (Tenant ID)
```

Grant the service principal access to the workspace:

```bash
az role assignment create \
  --assignee  \
  --role "Log Analytics Reader" \
  --scope /subscriptions//resourceGroups/ARCNET/providers/Microsoft.OperationalInsights/workspaces/CiscoSwitchLogs
```

### Step 3: Dashboard Details

Create new dashboards using Kusto query and format them.

Here are some existing panels we have:

- Device Health: CPU, Memory, Temperature, Power Supply monitoring
- Interface Performance: Traffic (bps/pps), Packet distribution
- Routing Metrics: Route counts, BGP neighbor status and routes learned
- Error Metrics: Interface errors, discards, error rate distribution
- ARP & MAC Tracking: Table size trends
- LLDP & Inventory: Neighbor discovery, transceiver monitoring
- System Information: NX-OS version, hardware details, kernel uptime, last reset info


