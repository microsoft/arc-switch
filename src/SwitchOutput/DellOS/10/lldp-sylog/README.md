# Switch LLDP

```mermaid
flowchart TD
    A[Start Golang script: lldp_syslog] --> B[Script issues clish command]
    B --> C[Parse data]
    C --> C1[Separate data into array of entries]
    C1 --> C2[Issue call to logger command]
    C2 --> D[Arc agent polls syslog file]
    D --> E[Data present in Azure logs for Arc]
    E --> F[KQL query runs against the data]
```

## Example Data

[show lldp neighbor details](./show-lldp-neighbors-detail.txt)

![LLDP Table](../../../../images/lldp-table.png)

## KQL

[KQL file](./lldp-kql-query.kql)

```sql
Syslog 
| where ProcessName contains "LLDPNeighbor"
| where EventTime between ( datetime(2025-5-13 17:09:39) .. datetime(2025-5-13 17:09:40) )
| extend syslogItems = parse_json(SyslogMessage)
| project 
    EventTime = EventTime,
    Switch = Computer,
    S_Port = tostring(syslogItems.local_port_id),
    D_Device = tostring(syslogItems.remote_system_name),
    D_Port = tostring(syslogItems.remote_port_id),
    D_Desc = tostring(syslogItems.remote_port_description),
    D_MTU = tostring(syslogItems.remote_mtu)
```

## Example lldp neighbor details

[LLDP Example Data](./show-lldp-neighbors-detail.txt)

## Example LLDP JSON

```JSON
{
    "hostname": "",
    "local_port_id": "ethernet1/1/52",
    "remote_system_name": "s46r23b-Rack01-TOR-2",
    "remote_port_id": "ethernet1/1/52",
    "remote_chassis_id": "0c:29:ef:c3:0b:20",
    "remote_port_description": "ethernet1/1/52",
    "remote_mtu": "9216",
    "timestamp": "2025-05-13T17:09:39.0000000Z"
}
```

## Debian Package Creation

This directory includes two build scripts for creating Debian packages:

### build_interface_syslog_deb.sh (Recommended)

Creates a Debian package named `interface-syslog` that installs the LLDP service to `/opt/microsoft/interface-syslog` and configures systemd services.

**Usage:**
```bash
# Build package only
./build_interface_syslog_deb.sh

# Build and install package  
./build_interface_syslog_deb.sh --install
```

**Package details:**
- Package name: `interface-syslog`
- Installation path: `/opt/microsoft/interface-syslog`
- Service name: `interface-syslog.service`
- Timer name: `interface-syslog.timer`
- Runs as root user
- Executes every 1 minute via systemd timer

### build_lldp_syslog_deb.sh (Legacy)

Original build script that creates a `lldpsyslog` package installing to `/opt/microsoft/lldpsyslog`.

**Installation:**
After building the package, install with:
```bash
sudo dpkg -i interface-syslog_1.0_amd64.deb
```

**Uninstallation:**
```bash
sudo apt remove interface-syslog
```

**Package Contents:**
- Binary: `/opt/microsoft/interface-syslog`
- Service: `/etc/systemd/system/interface-syslog.service` 
- Timer: `/etc/systemd/system/interface-syslog.timer`
- Post-install script: Enables and starts the timer
- Post-removal script: Stops, disables and cleans up services
