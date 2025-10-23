# Cisco Nexus Environment Power Details Parser

This parser processes the output of the `show environment power detail | json` command from Cisco Nexus switches and converts it to structured JSON format for power monitoring and analysis.

## Features

- **Comprehensive Power Data Parsing**: Extracts all power-related information including:
  - Power supply information (model, capacity, status)
  - Power usage summary (redundancy modes, total capacity, actual draw)
  - Power usage details (reserved power, inlet cord status)
  - Detailed power supply metrics (voltage, current, power in/out)
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **JSON Lines Format**: Each entry produces a separate JSON object for streaming/logging compatibility
- **Integrated with Cisco Parser**: Available as part of the unified cisco-parser binary

## Installation

### Building the Unified Parser

This parser is integrated into the unified cisco-parser binary:

```bash
cd /home/runner/work/arc-switch/arc-switch/src/SwitchOutput/Cisco/Nexus/10/cisco-parser
make
```

## Usage

### Using the Unified Cisco Parser

```bash
# Parse power environment data from a file
./cisco-parser -p environment-power -i show-environment-power-detail.txt -o power-output.json

# List all available parsers
./cisco-parser -list
```

### Using Commands File

The parser looks for a command named `power-supply` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "power-supply",
      "command": "show environment power detail | json"
    }
  ]
}
```

## Output Format

The parser outputs JSON Lines format with a standardized structure. Each power environment snapshot produces a single JSON object:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_environment_power",
  "timestamp": "2025-10-21T10:30:45Z",
  "date": "2025-10-21",
  "message": {
    // Power environment data here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_environment_power"
- `timestamp`: Processing timestamp in ISO 8601 format
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all power environment data

### Message Structure

The `message` field contains comprehensive power data:

Command: `show environment power detail | json`

Example file: [show-environment-power-detail.txt](../show-environment-power-detail.txt)

```json
{
  "voltage": "12 Volts",
  "power_supplies": [
    {
      "ps_number": "1",
      "model": "NXA-PAC-500W-PE",
      "actual_output": "79 W",
      "actual_input": "89 W",
      "total_capacity": "500 W",
      "status": "Ok"
    }
  ],
  "power_summary": {
    "ps_redundancy_mode_configured": "PS-Redundant",
    "ps_redundancy_mode_operational": "PS-Redundant",
    "total_power_capacity": "500.00 W",
    "total_grid_a_power_capacity": "500.00 W",
    "total_grid_b_power_capacity": "500.00 W",
    "total_power_of_all_inputs": "1000.00 W",
    "total_power_output_actual_draw": "158.00 W",
    "total_power_input_actual_draw": "179.00 W",
    "total_power_allocated_budget": "N/A",
    "total_power_available": "N/A"
  },
  "power_usage_details": {
    "power_reserved_for_supervisors": "N/A",
    "power_reserved_for_fabric_sc": "N/A",
    "power_reserved_for_fan_modules": "N/A",
    "total_power_reserved": "N/A",
    "all_inlet_cords_connected": "Yes"
  },
  "power_supply_details": [
    {
      "name": "PS_1",
      "total_capacity": "500 W",
      "voltage": "12V",
      "pin": "89.00W",
      "vin": "207.00V",
      "iin": "0.44A",
      "pout": "79.00W",
      "vout": "12.08V",
      "iout": "6.62A",
      "cord_status": "connected to 220V AC",
      "software_alarm": "No"
    }
  ]
}
```

## Testing

Run the tests to verify the parser:

```bash
cd /home/runner/work/arc-switch/arc-switch/src/SwitchOutput/Cisco/Nexus/10/show-environment-power-details
go test -v
```

## KQL Query Examples

The `data_type` field enables easy filtering in Azure Log Analytics or other systems that use KQL (Kusto Query Language).

### Basic Queries

```kql
// Filter for Cisco Nexus power environment entries only
| where data_type == "cisco_nexus_environment_power"

// Get the most recent power status
| where data_type == "cisco_nexus_environment_power"
| top 1 by timestamp desc

// View power supply status for a specific date
| where data_type == "cisco_nexus_environment_power" and date == "2025-10-21"
```

### Alert Queries - Power Supply Health

```kql
// Alert on power supply failures
| where data_type == "cisco_nexus_environment_power"
| mv-expand power_supply = message.power_supplies
| where power_supply.status != "Ok"
| project timestamp, ps_number = power_supply.ps_number, 
          model = power_supply.model, status = power_supply.status

// Alert on non-redundant power configuration
| where data_type == "cisco_nexus_environment_power"
| where message.power_summary.ps_redundancy_mode_operational != "PS-Redundant"
| project timestamp, configured = message.power_summary.ps_redundancy_mode_configured,
          operational = message.power_summary.ps_redundancy_mode_operational

// Alert on disconnected power cords
| where data_type == "cisco_nexus_environment_power"
| where message.power_usage_details.all_inlet_cords_connected != "Yes"
| project timestamp, inlet_cords_status = message.power_usage_details.all_inlet_cords_connected

// Alert on power supply software alarms
| where data_type == "cisco_nexus_environment_power"
| mv-expand ps_detail = message.power_supply_details
| where ps_detail.software_alarm != "No" and ps_detail.software_alarm != ""
| project timestamp, ps_name = ps_detail.name, alarm_status = ps_detail.software_alarm
```

### Capacity and Load Monitoring

```kql
// Calculate power usage percentage
| where data_type == "cisco_nexus_environment_power"
| extend total_capacity_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_capacity))
| extend actual_draw_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| extend usage_percentage = (actual_draw_numeric / total_capacity_numeric) * 100
| project timestamp, total_capacity_numeric, actual_draw_numeric, usage_percentage
| order by timestamp desc

// Alert on high power usage (>80% of capacity)
| where data_type == "cisco_nexus_environment_power"
| extend total_capacity_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_capacity))
| extend actual_draw_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| extend usage_percentage = (actual_draw_numeric / total_capacity_numeric) * 100
| where usage_percentage > 80
| project timestamp, usage_percentage, total_capacity = message.power_summary.total_power_capacity,
          actual_draw = message.power_summary.total_power_output_actual_draw

// Alert on low available power capacity
| where data_type == "cisco_nexus_environment_power"
| extend total_capacity_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_capacity))
| extend actual_draw_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| extend available_power = total_capacity_numeric - actual_draw_numeric
| where available_power < 100  // Less than 100W available
| project timestamp, available_power, total_capacity_numeric, actual_draw_numeric
```

### Trend Analysis Queries

```kql
// Trend power consumption over time
| where data_type == "cisco_nexus_environment_power"
| extend actual_draw_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| summarize avg_power_draw = avg(actual_draw_numeric), 
            max_power_draw = max(actual_draw_numeric),
            min_power_draw = min(actual_draw_numeric)
  by bin(timestamp, 1h)
| order by timestamp asc
| render timechart

// Trend power input vs output efficiency over time
| where data_type == "cisco_nexus_environment_power"
| extend power_input = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_input_actual_draw))
| extend power_output = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| extend efficiency = (power_output / power_input) * 100
| summarize avg_efficiency = avg(efficiency) by bin(timestamp, 1h)
| order by timestamp asc
| render timechart

// Track power supply count over time
| where data_type == "cisco_nexus_environment_power"
| extend ps_count = array_length(message.power_supplies)
| extend ps_ok_count = array_length(message.power_supplies[?status == "Ok"])
| summarize avg_ps_count = avg(ps_count), 
            avg_ps_ok_count = avg(ps_ok_count)
  by bin(timestamp, 1d)
| order by timestamp asc
```

### Power Supply Performance Analysis

```kql
// Analyze individual power supply performance
| where data_type == "cisco_nexus_environment_power"
| mv-expand ps = message.power_supplies
| extend ps_output_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", tostring(ps.actual_output)))
| extend ps_capacity_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", tostring(ps.total_capacity)))
| extend ps_load_percentage = (ps_output_numeric / ps_capacity_numeric) * 100
| project timestamp, ps_number = ps.ps_number, ps_model = ps.model,
          ps_load_percentage, ps_status = ps.status
| order by timestamp desc

// Identify power supply load imbalance
| where data_type == "cisco_nexus_environment_power"
| mv-expand ps = message.power_supplies
| extend ps_output_numeric = todouble(replace(@"([0-9.]+)\s*W", @"\1", tostring(ps.actual_output)))
| summarize ps_loads = make_list(ps_output_numeric) by timestamp
| extend max_load = array_max(ps_loads)
| extend min_load = array_min(ps_loads)
| extend load_difference = max_load - min_load
| where load_difference > 20  // Alert if difference > 20W
| project timestamp, max_load, min_load, load_difference

// Compare power supply voltage and current metrics
| where data_type == "cisco_nexus_environment_power"
| mv-expand ps_detail = message.power_supply_details
| extend vin_numeric = todouble(replace(@"([0-9.]+)V", @"\1", tostring(ps_detail.vin)))
| extend vout_numeric = todouble(replace(@"([0-9.]+)V", @"\1", tostring(ps_detail.vout)))
| extend iin_numeric = todouble(replace(@"([0-9.]+)A", @"\1", tostring(ps_detail.iin)))
| extend iout_numeric = todouble(replace(@"([0-9.]+)A", @"\1", tostring(ps_detail.iout)))
| project timestamp, ps_name = ps_detail.name, vin_numeric, vout_numeric, 
          iin_numeric, iout_numeric, cord_status = ps_detail.cord_status
```

### Anomaly Detection Queries

```kql
// Detect sudden power consumption changes
| where data_type == "cisco_nexus_environment_power"
| extend power_draw = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| serialize timestamp_sort = timestamp
| extend prev_power_draw = prev(power_draw)
| extend power_change = power_draw - prev_power_draw
| extend power_change_percentage = abs((power_draw - prev_power_draw) / prev_power_draw) * 100
| where power_change_percentage > 20  // Alert on >20% change
| project timestamp, power_draw, prev_power_draw, power_change, power_change_percentage

// Identify outlier power values using standard deviation
| where data_type == "cisco_nexus_environment_power"
| extend power_draw = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| summarize avg_power = avg(power_draw), 
            stdev_power = stdev(power_draw)
| extend lower_threshold = avg_power - (2 * stdev_power)
| extend upper_threshold = avg_power + (2 * stdev_power)
| join kind=inner (
    | where data_type == "cisco_nexus_environment_power"
    | extend power_draw = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
  ) on $left.avg_power == $right.power_draw
| where power_draw < lower_threshold or power_draw > upper_threshold
| project timestamp, power_draw, avg_power, stdev_power, lower_threshold, upper_threshold

// Detect voltage anomalies in power supplies
| where data_type == "cisco_nexus_environment_power"
| mv-expand ps_detail = message.power_supply_details
| extend vout_numeric = todouble(replace(@"([0-9.]+)V", @"\1", tostring(ps_detail.vout)))
| where vout_numeric < 11.5 or vout_numeric > 12.5  // Alert if output voltage is outside normal range
| project timestamp, ps_name = ps_detail.name, vout_numeric, 
          expected_voltage = ps_detail.voltage, cord_status = ps_detail.cord_status
```

### Grid and Redundancy Monitoring

```kql
// Monitor Grid-A and Grid-B capacity balance
| where data_type == "cisco_nexus_environment_power"
| extend grid_a_capacity = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_grid_a_power_capacity))
| extend grid_b_capacity = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_grid_b_power_capacity))
| project timestamp, grid_a_capacity, grid_b_capacity,
          grid_balance = abs(grid_a_capacity - grid_b_capacity)
| where grid_balance > 0  // Alert if grids are not balanced
| order by timestamp desc

// Alert on redundancy mode mismatch
| where data_type == "cisco_nexus_environment_power"
| where message.power_summary.ps_redundancy_mode_configured != message.power_summary.ps_redundancy_mode_operational
| project timestamp, 
          configured_mode = message.power_summary.ps_redundancy_mode_configured,
          operational_mode = message.power_summary.ps_redundancy_mode_operational
```

### Reporting and Dashboard Queries

```kql
// Power supply inventory and health report
| where data_type == "cisco_nexus_environment_power"
| mv-expand ps = message.power_supplies
| summarize count() by ps_model = tostring(ps.model), ps_status = tostring(ps.status)
| order by ps_model asc

// Daily power consumption summary
| where data_type == "cisco_nexus_environment_power"
| extend power_output = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| extend power_input = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_input_actual_draw))
| summarize avg_output = avg(power_output),
            max_output = max(power_output),
            min_output = min(power_output),
            avg_input = avg(power_input),
            avg_efficiency = avg((power_output / power_input) * 100)
  by date
| order by date desc

// Power capacity utilization dashboard
| where data_type == "cisco_nexus_environment_power"
| extend total_capacity = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_capacity))
| extend actual_draw = todouble(replace(@"([0-9.]+)\s*W", @"\1", message.power_summary.total_power_output_actual_draw))
| extend available_power = total_capacity - actual_draw
| extend utilization_percentage = (actual_draw / total_capacity) * 100
| project timestamp, total_capacity, actual_draw, available_power, utilization_percentage
| order by timestamp desc
| take 100
```

## Integration with Cisco Parser

This parser is automatically integrated into the unified `cisco-parser` binary. The parser is registered in the main binary with the identifier `environment-power`.

To use it with the unified parser:

```bash
./cisco-parser -p environment-power -i show-environment-power-detail.txt -o output.json
```

## Parsed Data Elements

### Power Supply Table
- PS Number
- Model
- Actual Output (Watts)
- Actual Input (Watts)
- Total Capacity (Watts)
- Status

### Power Usage Summary
- PS Redundancy Mode (Configured)
- PS Redundancy Mode (Operational)
- Total Power Capacity
- Total Grid-A Power Capacity
- Total Grid-B Power Capacity
- Total Power of All Inputs (cumulative)
- Total Power Output (actual draw)
- Total Power Input (actual draw)
- Total Power Allocated (budget)
- Total Power Available

### Power Usage Details
- Power Reserved for Supervisors
- Power Reserved for Fabric, SC Modules
- Power Reserved for Fan Modules
- Total Power Reserved
- All Inlet Cords Connected Status

### Power Supply Details (Per PS)
- Name (PS_1, PS_2, etc.)
- Total Capacity
- Voltage
- Pin (Power Input)
- Vin (Voltage Input)
- Iin (Current Input)
- Pout (Power Output)
- Vout (Voltage Output)
- Iout (Current Output)
- Cord Connection Status
- Software Alarm Status

## Error Handling

The parser includes robust error handling for:
- Missing or malformed input data
- Incomplete power supply information
- N/A values in power metrics
- Various output formats from different Nexus switch models

## License

This parser is part of the arc-switch project and follows the same license terms.
