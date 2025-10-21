# Cisco Nexus Transceiver Parser

This tool parses Cisco Nexus `show interface transceiver details` output and converts it to structured JSON format for optical monitoring and inventory management.

## Features

- **Dual Input Format Support**: Automatically detects and parses both text and JSON input formats
- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Transceiver Data**: Captures all transceiver information including type, manufacturer, serial numbers, and specifications
- **Enhanced QSFP Support**: Full support for QSFP multi-lane transceivers with per-lane DOM monitoring
- **DOM Support**: Parses Digital Optical Monitoring data with thresholds and alarm status
  - Temperature (current, high/low alarms, high/low warnings, status)
  - Voltage (current, high/low alarms, high/low warnings, status)
  - Current (current, high/low alarms, high/low warnings, status)
  - TX Power (current, high/low alarms, high/low warnings, status)
  - RX Power (current, high/low alarms, high/low warnings, status)
  - Transmit fault count
- **Multi-vendor Support**: Handles various transceiver types (SFP, QSFP, copper, optical)
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each transceiver produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of absent transceivers and DOM data
- **CLI Integration**: Uses `vsh` for direct switch communication
- **Automated Health Monitoring**: KQL query examples for trend detection, outlier identification, and predictive maintenance

## Installation

```bash
# Build the binary
go build -o transceiver_parser transceiver_parser.go
```

## Usage

### Parse from Input File

The parser automatically detects the input format (text or JSON) and processes accordingly.

```bash
# Parse transceiver data from a text file
./transceiver_parser -input show-transceiver.txt -output output.json

# Parse transceiver data from JSON format
./transceiver_parser -input show-transceiver-json.txt -output output.json

# Parse and output to stdout
./transceiver_parser -input show-transceiver.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./transceiver_parser -commands ../commands.json -output output.json
```

### Collecting Data from Cisco Nexus Switch

To collect transceiver data in JSON format directly from a Cisco Nexus switch:

```bash
# Text format (traditional)
show interface transceiver details | no

# JSON format (for parser compatibility)
show interface transceiver details | json-pretty

# Save to file for offline parsing
show interface transceiver details | json-pretty > /bootflash/transceiver-data.json
```

### Command Line Options

- `-input <file>`: Input file containing `show interface transceiver details` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool supports two input formats and automatically detects which one is being used:

### Text Format

Standard Cisco Nexus `show interface transceiver details` output:

- Transceiver presence status
- Type and manufacturer details
- Part numbers and serial numbers
- Bitrate and cable/fiber specifications
- Cisco product identifiers
- DOM support status
- Digital Optical Monitoring data (when available)

### JSON Format

JSON output from `show interface transceiver details | json-pretty`:

- All text format fields in structured JSON
- Support for QSFP multi-lane transceivers
- Lane-specific DOM monitoring data
- Automatic parsing of nested TABLE/ROW structures
- Enhanced support for QSFP-40G-CSR4, QSFP-100G-PCC, and other multi-lane transceivers

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each transceiver produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_transceiver",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    // Transceiver-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_transceiver"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-01-20T10:30:45Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all transceiver data

### Message Fields

The `message` field contains all transceiver-specific data:

```json
{
  "interface_name": "Ethernet1/17",
  "transceiver_present": true,
  "type": "10Gbase-SR",
  "manufacturer": "Siemon",
  "part_number": "S1S10F-V05.0M13",
  "revision": "A",
  "serial_number": "SIM24001B1A",
  "nominal_bitrate": 10300,
  "link_length": "50/125um OM3 fiber is 300 m",
  "cable_type": "",
  "cisco_id": "3",
  "cisco_extended_id": "4",
  "cisco_part_number": "10-2415-03",
  "cisco_product_id": "SFP-10G-SR",
  "cisco_version_id": "V03",
  "dom_supported": true,
  "dom_data": {
    "temperature": {
      "current_value": 34.22,
      "unit": "C",
      "alarm_high": 80.0,
      "alarm_low": -10.0,
      "warning_high": 75.0,
      "warning_low": -5.0,
      "status": "normal"
    },
    "voltage": {
      "current_value": 3.26,
      "unit": "V",
      "alarm_high": 3.6,
      "alarm_low": 3.0,
      "warning_high": 3.5,
      "warning_low": 3.1,
      "status": "normal"
    },
    "current": {
      "current_value": 6.76,
      "unit": "mA",
      "alarm_high": 15.0,
      "alarm_low": 0.0,
      "warning_high": 12.0,
      "warning_low": 0.0,
      "status": "normal"
    },
    "tx_power": {
      "current_value": -1.45,
      "unit": "dBm",
      "alarm_high": 0.99,
      "alarm_low": -8.32,
      "warning_high": 0.0,
      "warning_low": -7.79,
      "status": "normal"
    },
    "rx_power": {
      "current_value": -1.89,
      "unit": "dBm",
      "alarm_high": 0.99,
      "alarm_low": -10.91,
      "warning_high": 0.0,
      "warning_low": -9.91,
      "status": "normal"
    },
    "transmit_fault_count": 0
  }
}
```

#### Core Fields

- `interface_name`: Interface identifier (e.g., "Ethernet1/1")
- `transceiver_present`: Boolean indicating if transceiver is installed

#### Transceiver Details (when present)

- `type`: Transceiver type (e.g., "SFP-H25GB-CU3M", "10Gbase-SR", "QSFP-100G-CR4")
- `manufacturer`: Manufacturer name (e.g., "CISCO-AMPHENOL", "Siemon")
- `part_number`: Manufacturer part number
- `revision`: Hardware revision
- `serial_number`: Unique serial number
- `nominal_bitrate`: Speed in MBit/sec (e.g., 25500, 10300)

#### Physical Specifications

- `link_length`: Cable length or fiber distance specifications
- `cable_type`: Cable type code (e.g., "CA-S", "CA-N")

#### Cisco Identifiers

- `cisco_id`: Cisco ID number
- `cisco_extended_id`: Extended ID number
- `cisco_part_number`: Cisco part number
- `cisco_product_id`: Cisco product identifier
- `cisco_version_id`: Version identifier

#### DOM (Digital Optical Monitoring) Fields

- `dom_supported`: Boolean indicating DOM support
- `dom_data`: Object containing monitoring data (when supported):
  - `temperature`: Temperature monitoring with thresholds
  - `voltage`: Voltage monitoring with thresholds
  - `current`: Current monitoring with thresholds
  - `tx_power`: Transmit power monitoring with thresholds
  - `rx_power`: Receive power monitoring with thresholds
  - `transmit_fault_count`: Number of transmit faults

Each DOM parameter includes:
- `current_value`: Current measurement
- `unit`: Measurement unit (C, V, mA, dBm)
- `alarm_high`: High alarm threshold
- `alarm_low`: Low alarm threshold
- `warning_high`: High warning threshold
- `warning_low`: Low warning threshold
- `status`: Calculated status (normal, high-alarm, low-alarm, high-warning, low-warning)

## KQL Query Examples for Monitoring

These KQL (Kusto Query Language) queries demonstrate how to monitor transceiver health in Azure Data Explorer, Azure Monitor, or Log Analytics. They provide actionable insights for senior network engineers.

### Basic Queries

#### View All Transceiver Data

```kql
// View all transceiver telemetry
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| extend interface_name = tostring(message.interface_name)
| extend transceiver_present = tobool(message.transceiver_present)
| extend type = tostring(message.type)
| extend manufacturer = tostring(message.manufacturer)
| extend serial_number = tostring(message.serial_number)
| project timestamp, interface_name, transceiver_present, type, manufacturer, serial_number
| order by timestamp desc
```

#### Filter Only Active Transceivers with DOM Support

```kql
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.transceiver_present == true
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend type = tostring(message.type)
| project timestamp, interface_name, type
| order by timestamp desc
```

### Temperature Monitoring

#### Current Temperature Status Across All Interfaces

```kql
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend temp = todouble(message.dom_data.temperature.current_value)
| extend temp_status = tostring(message.dom_data.temperature.status)
| extend temp_alarm_high = todouble(message.dom_data.temperature.alarm_high)
| extend temp_alarm_low = todouble(message.dom_data.temperature.alarm_low)
| project timestamp, interface_name, temp, temp_status, temp_alarm_high, temp_alarm_low
| order by timestamp desc, temp desc
```

#### Identify Temperature Alarms and Warnings

```kql
// Find transceivers with temperature issues
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend temp = todouble(message.dom_data.temperature.current_value)
| extend temp_status = tostring(message.dom_data.temperature.status)
| where temp_status != "normal"
| project timestamp, interface_name, temp, temp_status
| order by timestamp desc
```

#### Temperature Trend Analysis (Last 24 Hours)

```kql
// Track temperature trends over time
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where timestamp > ago(24h)
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend temp = todouble(message.dom_data.temperature.current_value)
| summarize 
    avg_temp = avg(temp),
    min_temp = min(temp),
    max_temp = max(temp),
    current_temp = arg_max(timestamp, temp)
    by interface_name
| order by max_temp desc
```

#### Temperature Outlier Detection

```kql
// Detect temperature outliers using statistical methods
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where timestamp > ago(7d)
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend temp = todouble(message.dom_data.temperature.current_value)
| summarize 
    avg_temp = avg(temp),
    stdev_temp = stdev(temp),
    samples = count()
    by interface_name
| extend upper_threshold = avg_temp + (2 * stdev_temp)
| extend lower_threshold = avg_temp - (2 * stdev_temp)
| join kind=inner (
    SwitchLogs
    | where data_type == "cisco_nexus_transceiver"
    | where timestamp > ago(1h)
    | where message.dom_supported == true
    | extend interface_name = tostring(message.interface_name)
    | extend temp = todouble(message.dom_data.temperature.current_value)
    | summarize current_temp = avg(temp) by interface_name
) on interface_name
| where current_temp > upper_threshold or current_temp < lower_threshold
| project interface_name, current_temp, avg_temp, upper_threshold, lower_threshold
| order by abs(current_temp - avg_temp) desc
```

### Voltage Monitoring

#### Voltage Status and Alerts

```kql
// Monitor voltage levels across all transceivers
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend voltage = todouble(message.dom_data.voltage.current_value)
| extend voltage_status = tostring(message.dom_data.voltage.status)
| extend voltage_alarm_high = todouble(message.dom_data.voltage.alarm_high)
| extend voltage_alarm_low = todouble(message.dom_data.voltage.alarm_low)
| where voltage_status != "normal"
| project timestamp, interface_name, voltage, voltage_status, voltage_alarm_high, voltage_alarm_low
| order by timestamp desc
```

#### Voltage Trend Detection

```kql
// Detect voltage drift over time
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where timestamp > ago(24h)
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend voltage = todouble(message.dom_data.voltage.current_value)
| summarize 
    voltage_values = make_list(voltage),
    timestamps = make_list(timestamp)
    by interface_name
| extend voltage_slope = series_fit_line_dynamic(voltage_values).slope
| where abs(voltage_slope) > 0.01  // Threshold for significant drift
| project interface_name, voltage_slope, voltage_values
| order by abs(voltage_slope) desc
```

### Optical Power Monitoring

#### TX and RX Power Analysis

```kql
// Monitor transmit and receive power levels
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend tx_power = todouble(message.dom_data.tx_power.current_value)
| extend rx_power = todouble(message.dom_data.rx_power.current_value)
| extend tx_status = tostring(message.dom_data.tx_power.status)
| extend rx_status = tostring(message.dom_data.rx_power.status)
| project timestamp, interface_name, tx_power, tx_status, rx_power, rx_status
| order by timestamp desc
```

#### Optical Power Budget Analysis

```kql
// Calculate optical power budget (TX - RX) to assess link quality
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend tx_power = todouble(message.dom_data.tx_power.current_value)
| extend rx_power = todouble(message.dom_data.rx_power.current_value)
| extend power_budget = tx_power - rx_power
| summarize 
    avg_power_budget = avg(power_budget),
    latest_tx = arg_max(timestamp, tx_power),
    latest_rx = arg_max(timestamp, rx_power)
    by interface_name
| where avg_power_budget > 10  // Flag links with high loss
| project interface_name, avg_power_budget, latest_tx, latest_rx
| order by avg_power_budget desc
```

#### Detect Failing Optical Links

```kql
// Identify optical links approaching failure
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend rx_power = todouble(message.dom_data.rx_power.current_value)
| extend rx_alarm_low = todouble(message.dom_data.rx_power.alarm_low)
| extend rx_status = tostring(message.dom_data.rx_power.status)
| where rx_status in ("low-warning", "low-alarm") or rx_power < (rx_alarm_low + 2)
| project timestamp, interface_name, rx_power, rx_alarm_low, rx_status
| order by rx_power asc
```

### Current Monitoring

#### Current Consumption Analysis

```kql
// Monitor current consumption across transceivers
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend current = todouble(message.dom_data.current.current_value)
| extend current_status = tostring(message.dom_data.current.status)
| summarize 
    avg_current = avg(current),
    max_current = max(current)
    by interface_name
| order by max_current desc
```

### Transmit Fault Detection

#### Identify Transceivers with Transmit Faults

```kql
// Find transceivers with transmit faults
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend fault_count = toint(message.dom_data.transmit_fault_count)
| where fault_count > 0
| summarize 
    total_faults = sum(fault_count),
    latest_faults = arg_max(timestamp, fault_count)
    by interface_name
| project interface_name, total_faults, latest_faults
| order by total_faults desc
```

### Multi-Parameter Health Score

#### Calculate Transceiver Health Score

```kql
// Comprehensive health score based on all DOM parameters
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend temp_status = tostring(message.dom_data.temperature.status)
| extend voltage_status = tostring(message.dom_data.voltage.status)
| extend current_status = tostring(message.dom_data.current.status)
| extend tx_status = tostring(message.dom_data.tx_power.status)
| extend rx_status = tostring(message.dom_data.rx_power.status)
| extend fault_count = toint(message.dom_data.transmit_fault_count)
| extend health_score = 
    iff(temp_status == "normal", 20, iff(temp_status in ("high-warning", "low-warning"), 10, 0)) +
    iff(voltage_status == "normal", 20, iff(voltage_status in ("high-warning", "low-warning"), 10, 0)) +
    iff(current_status == "normal", 20, iff(current_status in ("high-warning", "low-warning"), 10, 0)) +
    iff(tx_status == "normal", 20, iff(tx_status in ("high-warning", "low-warning"), 10, 0)) +
    iff(rx_status == "normal", 20, iff(rx_status in ("high-warning", "low-warning"), 10, 0)) -
    (fault_count * 5)
| summarize avg_health_score = avg(health_score) by interface_name
| where avg_health_score < 80  // Flag interfaces with health issues
| order by avg_health_score asc
```

### Alert Conditions for Network Operations

#### Critical Alert: Multiple DOM Parameters Out of Range

```kql
// Generate alerts for transceivers with multiple issues
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where timestamp > ago(5m)  // Recent data only
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend temp_status = tostring(message.dom_data.temperature.status)
| extend voltage_status = tostring(message.dom_data.voltage.status)
| extend rx_status = tostring(message.dom_data.rx_power.status)
| extend issue_count = 
    iff(temp_status != "normal", 1, 0) +
    iff(voltage_status != "normal", 1, 0) +
    iff(rx_status != "normal", 1, 0)
| where issue_count >= 2
| project timestamp, interface_name, temp_status, voltage_status, rx_status, issue_count
| order by issue_count desc, timestamp desc
```

#### Predictive Maintenance Alert

```kql
// Identify transceivers approaching failure based on trends
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where timestamp > ago(7d)
| where message.dom_supported == true
| extend interface_name = tostring(message.interface_name)
| extend rx_power = todouble(message.dom_data.rx_power.current_value)
| summarize 
    power_values = make_list(rx_power),
    timestamps = make_list(timestamp)
    by interface_name
| extend power_trend = series_fit_line_dynamic(power_values).slope
| where power_trend < -0.1  // Rapidly declining RX power
| project interface_name, power_trend, current_rx = power_values[-1]
| order by power_trend asc
```

### Dashboard Aggregations

#### Summary Dashboard

```kql
// Create summary statistics for dashboard
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where timestamp > ago(5m)
| extend transceiver_present = tobool(message.transceiver_present)
| extend dom_supported = tobool(message.dom_supported)
| summarize 
    total_ports = dcount(tostring(message.interface_name)),
    transceivers_present = countif(transceiver_present),
    dom_supported_count = countif(dom_supported),
    transceivers_missing = countif(not(transceiver_present))
| extend utilization_percent = round(transceivers_present * 100.0 / total_ports, 2)
```

#### Per-Type Health Summary

```kql
// Health summary by transceiver type
SwitchLogs
| where data_type == "cisco_nexus_transceiver"
| where message.dom_supported == true
| extend type = tostring(message.type)
| extend temp_status = tostring(message.dom_data.temperature.status)
| extend voltage_status = tostring(message.dom_data.voltage.status)
| extend rx_status = tostring(message.dom_data.rx_power.status)
| summarize 
    count = count(),
    temp_issues = countif(temp_status != "normal"),
    voltage_issues = countif(voltage_status != "normal"),
    rx_issues = countif(rx_status != "normal")
    by type
| extend health_percent = round((count - temp_issues - voltage_issues - rx_issues) * 100.0 / count, 2)
| order by health_percent asc
```

## Transceiver Types

The parser handles various transceiver types:

- **Copper DAC**: SFP-H25GB-CU3M, SFP-H25GB-CU1M, QSFP-100G-CR4
- **Optical**: 10Gbase-SR, 10Gbase-LR, 40Gbase-SR4
- **Form Factors**: SFP, SFP+, SFP28, QSFP, QSFP28
- **Speeds**: 1G, 10G, 25G, 40G, 100G

## Data Handling

- **Absent Transceivers**: Interfaces without transceivers have `transceiver_present: false`
- **DOM Detection**: Automatically detects DOM support from output
- **Status Calculation**: Compares current values against thresholds to determine alarm/warning status
- **Empty Fields**: Missing or optional fields are omitted or set to empty strings
- **Numeric Parsing**: Bitrate values are parsed as integers, measurements as floats

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "transceiver"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "transceiver",
      "command": "show interface transceiver details"
    }
  ]
}
```

## Examples

### Sample Text Input

```text
Ethernet1/17
    transceiver is present
    type is 10Gbase-SR
    name is Siemon
    part number is S1S10F-V05.0M13
    revision is A
    serial number is SIM24001B1A
    nominal bitrate is 10300 MBit/sec
    cisco id is 3
    cisco extended id number is 4

           SFP Detail Diagnostics Information (internal calibration)
  ----------------------------------------------------------------------------
                Current              Alarms                  Warnings
                Measurement     High        Low         High          Low
  ----------------------------------------------------------------------------
  Temperature   34.22 C        80.00 C    -10.00 C     75.00 C       -5.00 C
  Voltage        3.26 V         3.60 V      3.00 V      3.50 V        3.10 V
  Current        6.76 mA       15.00 mA     0.00 mA    12.00 mA       0.00 mA
  Tx Power      -1.45 dBm       0.99 dBm   -8.32 dBm    0.00 dBm     -7.79 dBm
  Rx Power      -1.89 dBm       0.99 dBm  -10.91 dBm    0.00 dBm     -9.91 dBm
  Transmit Fault Count = 0
```

### Sample JSON Input (QSFP with DOM)

```json
{
    "TABLE_interface": {
        "ROW_interface": [
            {
                "interface": "Ethernet1/36/1",
                "sfp": "present",
                "type": "QSFP-40G-CSR4",
                "name": "FS",
                "partnum": "QSFP-CSR4-40G",
                "rev": "C",
                "serialnum": "C2407418236",
                "nom_bitrate": "10300",
                "len_cu": "200",
                "len_50_OM3": "300",
                "ciscoid": "13",
                "ciscoid_1": "16",
                "TABLE_lane": {
                    "ROW_lane": {
                        "lane_number": "1",
                        "temperature": "34.48",
                        "temp_alrm_hi": "75.00",
                        "temp_alrm_lo": "-5.00",
                        "temp_warn_hi": "70.00",
                        "temp_warn_lo": "0.00",
                        "voltage": "3.28",
                        "volt_alrm_hi": "3.63",
                        "volt_alrm_lo": "2.97",
                        "volt_warn_hi": "3.46",
                        "volt_warn_lo": "3.13",
                        "current": "6.49",
                        "current_alrm_hi": "15.00",
                        "current_alrm_lo": "0.50",
                        "current_warn_hi": "12.00",
                        "current_warn_lo": "2.00",
                        "tx_pwr": "-1.78",
                        "tx_pwr_alrm_hi": "2.99",
                        "tx_pwr_alrm_lo": "-9.50",
                        "tx_pwr_warn_hi": "1.99",
                        "tx_pwr_warn_lo": "-7.52",
                        "rx_pwr": "-1.24",
                        "rx_pwr_alrm_hi": "3.40",
                        "rx_pwr_alrm_lo": "-14.10",
                        "rx_pwr_warn_hi": "2.39",
                        "rx_pwr_warn_lo": "-11.10",
                        "xmit_faults": "0"
                    }
                }
            }
        ]
    }
}
```

### Sample Output

```json
{
  "data_type": "cisco_nexus_transceiver",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    "interface_name": "Ethernet1/17",
    "transceiver_present": true,
    "type": "10Gbase-SR",
    "manufacturer": "Siemon",
    "part_number": "S1S10F-V05.0M13",
    "revision": "A",
    "serial_number": "SIM24001B1A",
    "nominal_bitrate": 10300,
    "cisco_id": "3",
    "cisco_extended_id": "4",
    "dom_supported": true,
    "dom_data": {
      "temperature": {
        "current_value": 34.22,
        "unit": "C",
        "alarm_high": 80.0,
        "alarm_low": -10.0,
        "warning_high": 75.0,
        "warning_low": -5.0,
        "status": "normal"
      },
      "voltage": {
        "current_value": 3.26,
        "unit": "V",
        "alarm_high": 3.6,
        "alarm_low": 3.0,
        "warning_high": 3.5,
        "warning_low": 3.1,
        "status": "normal"
      },
      "current": {
        "current_value": 6.76,
        "unit": "mA",
        "alarm_high": 15.0,
        "alarm_low": 0.0,
        "warning_high": 12.0,
        "warning_low": 0.0,
        "status": "normal"
      },
      "tx_power": {
        "current_value": -1.45,
        "unit": "dBm",
        "alarm_high": 0.99,
        "alarm_low": -8.32,
        "warning_high": 0.0,
        "warning_low": -7.79,
        "status": "normal"
      },
      "rx_power": {
        "current_value": -1.89,
        "unit": "dBm",
        "alarm_high": 0.99,
        "alarm_low": -10.91,
        "warning_high": 0.0,
        "warning_low": -9.91,
        "status": "normal"
      },
      "transmit_fault_count": 0
    }
  }
}
```

## Validation

To validate that the parser output conforms to the standardized JSON structure, use the project's validation script:

```bash
# Validate parser output
./transceiver_parser -input show-transceiver.txt | /workspaces/arc-switch2/validate-parser-output.sh

# Or validate a saved output file
./transceiver_parser -input show-transceiver.txt -output transceiver-results.json
/workspaces/arc-switch2/validate-parser-output.sh transceiver-results.json
```

The validation script checks for:

- Presence of all required fields (`data_type`, `timestamp`, `date`, `message`)
- Correct timestamp format (ISO 8601)
- Correct date format (YYYY-MM-DD)
- Valid JSON structure

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed transceiver output
- DOM data parsing with varying formats
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)
- Absent transceivers and missing DOM support

## Requirements

- Go 1.21 or later
- For direct switch access: `vsh` command-line tool available on Cisco Nexus switches