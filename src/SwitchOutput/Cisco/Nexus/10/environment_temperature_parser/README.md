# Cisco Nexus Environment Temperature Parser

This parser extracts and formats environment temperature data from Cisco Nexus switch output. It processes the `show environment temperature | json` command output and converts it into structured JSON format for monitoring and analysis.

## Overview

The Environment Temperature Parser is designed to integrate with the cisco-parser system at `src/SwitchOutput/Cisco/Nexus/10/cisco-parser`. It provides real-time temperature monitoring capabilities for network administrators, enabling proactive hardware health management and early detection of thermal issues.

### Key Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Temperature Data**: Captures module, sensor, thresholds, current temperature, and status
- **Multi-sensor Support**: Handles different sensor types (FRONT, BACK, CPU, etc.)
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Each temperature reading produces a separate JSON object for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper validation
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

### Building from Source

```bash
# Navigate to the parser directory
cd src/SwitchOutput/Cisco/Nexus/10/environment_temperature_parser

# Build the binary
go build -o environment_temperature_parser environment_temperature_parser.go

# Run it
./environment_temperature_parser --help
```

## Usage

### Parse from Input File

```bash
# Parse temperature data from a text file
./environment_temperature_parser -input show-environment-temperature.txt -output output.json

# Parse and output to stdout
./environment_temperature_parser -input show-environment-temperature.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./environment_temperature_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show environment temperature` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show environment temperature` output:

```
CONTOSO-DC1-TOR-01# show environment temperature 
Temperature:
--------------------------------------------------------------------
Module   Sensor        MajorThresh   MinorThres   CurTemp     Status
                      (Celsius)     (Celsius)    (Celsius)         
--------------------------------------------------------------------
1        FRONT           80              70          28         Ok             
1        BACK            70              42          26         Ok             
1        CPU             90              80          42         Ok             
1        Homewood        110             90          43         Ok 
```

### JSON Input Format

Command: `show environment temperature | json`

Example file: [show-environment-temperature-json.txt](../show-environment-temperature-json.txt)

```json
{
    "TABLE_tempinfo": {
        "ROW_tempinfo": [
            {
                "tempmod": "1",
                "sensor": "FRONT",
                "majthres": "80",
                "minthres": "70",
                "curtemp": "29",
                "alarmstatus": "Ok"
            },
            {
                "tempmod": "1",
                "sensor": "BACK",
                "majthres": "70",
                "minthres": "50",
                "curtemp": "27",
                "alarmstatus": "Ok"
            },
            {
                "tempmod": "1",
                "sensor": "CPU",
                "majthres": "90",
                "minthres": "80",
                "curtemp": "36",
                "alarmstatus": "Ok"
            },
            {
                "tempmod": "1",
                "sensor": "Heavenly",
                "majthres": "110",
                "minthres": "90",
                "curtemp": "47",
                "alarmstatus": "Ok"
            }
        ]
    }
}
```

Each temperature entry includes:
- **Module**: Module number
- **Sensor**: Sensor name (FRONT, BACK, CPU, etc.)
- **MajorThresh**: Major temperature threshold in Celsius
- **MinorThres**: Minor temperature threshold in Celsius
- **CurTemp**: Current temperature in Celsius
- **Status**: Status (Ok, Alert, etc.)

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library. Each temperature reading produces a JSON object with the following structure:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_environment_temperature",
  "timestamp": "2025-01-20T10:30:45Z",
  "date": "2025-01-20",
  "message": {
    // Temperature-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_environment_temperature"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-01-20T10:30:45Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all temperature data

### Message Fields

The `message` field contains all temperature-specific data:

```json
{
  "module": "1",
  "sensor": "FRONT",
  "major_threshold": "80",
  "minor_threshold": "70",
  "current_temp": "28",
  "status": "Ok"
}
```

#### Field Descriptions

- `module`: Module number as a string
- `sensor`: Sensor identifier (FRONT, BACK, CPU, Homewood, etc.)
- `major_threshold`: Major temperature threshold in Celsius (triggers critical alerts)
- `minor_threshold`: Minor temperature threshold in Celsius (triggers warnings)
- `current_temp`: Current temperature reading in Celsius
- `status`: Current status (Ok, Alert, Critical, etc.)

## Sample Output

```json
{"data_type":"cisco_nexus_environment_temperature","timestamp":"2025-01-20T10:30:45Z","date":"2025-01-20","message":{"module":"1","sensor":"FRONT","major_threshold":"80","minor_threshold":"70","current_temp":"28","status":"Ok"}}
{"data_type":"cisco_nexus_environment_temperature","timestamp":"2025-01-20T10:30:45Z","date":"2025-01-20","message":{"module":"1","sensor":"BACK","major_threshold":"70","minor_threshold":"42","current_temp":"26","status":"Ok"}}
{"data_type":"cisco_nexus_environment_temperature","timestamp":"2025-01-20T10:30:45Z","date":"2025-01-20","message":{"module":"1","sensor":"CPU","major_threshold":"90","minor_threshold":"80","current_temp":"42","status":"Ok"}}
{"data_type":"cisco_nexus_environment_temperature","timestamp":"2025-01-20T10:30:45Z","date":"2025-01-20","message":{"module":"1","sensor":"Homewood","major_threshold":"110","minor_threshold":"90","current_temp":"43","status":"Ok"}}
```

## Building

```bash
go build -o environment_temperature_parser environment_temperature_parser.go
```

## Testing

```bash
go test -v
```

To test with the included sample file:

```bash
go build -o environment_temperature_parser environment_temperature_parser.go
./environment_temperature_parser -input ../show-environment-temperature.txt
```

## Direct Switch Integration

The parser can get data directly from the switch using the commands JSON file. This requires:

1. A `commands.json` file with command definitions
2. Network connectivity to the switch
3. Proper VSH (Virtual Shell) access

When using the `-commands` option, the parser will:
1. Load the commands JSON file
2. Find the command with name "environment-temperature" 
3. Execute the command using VSH
4. Parse the output and generate JSON results

### Commands JSON Format

When using the `-commands` option, provide a JSON file with the following structure:

```json
{
  "commands": [
    {
      "name": "environment-temperature",
      "command": "show environment temperature"
    }
  ]
}
```

## Integration with cisco-parser

This parser is integrated into the unified cisco-parser binary at `src/SwitchOutput/Cisco/Nexus/10/cisco-parser`. Once integrated, it can be invoked using:

```bash
cd ../cisco-parser
make build
./build/cisco-parser -parser environment-temperature -input ../show-environment-temperature.txt -output output.json
```

## KQL Query Examples

The `data_type` field enables easy filtering in Azure Log Analytics, Azure Monitor, or other systems that use KQL (Kusto Query Language). Below are actionable queries for network administrators to monitor and manage temperature health.

### Basic Temperature Monitoring

```kql
// Filter for temperature entries only
| where data_type == "cisco_nexus_environment_temperature"

// View all current temperature readings
| where data_type == "cisco_nexus_environment_temperature"
| project timestamp, module=tostring(message.module), sensor=tostring(message.sensor), 
          current_temp=toint(message.current_temp), status=tostring(message.status)
| order by timestamp desc

// Get the latest temperature for each sensor
| where data_type == "cisco_nexus_environment_temperature"
| summarize arg_max(timestamp, *) by tostring(message.module), tostring(message.sensor)
| project timestamp, module=tostring(message.module), sensor=tostring(message.sensor),
          current_temp=toint(message.current_temp), major_threshold=toint(message.major_threshold),
          minor_threshold=toint(message.minor_threshold), status=tostring(message.status)
```

### Temperature Alerting and Anomaly Detection

```kql
// Alert: Temperatures approaching minor threshold (within 5°C)
| where data_type == "cisco_nexus_environment_temperature"
| extend current = toint(message.current_temp), minor = toint(message.minor_threshold)
| where current >= (minor - 5)
| project timestamp, module=tostring(message.module), sensor=tostring(message.sensor),
          current_temp=current, minor_threshold=minor, 
          degrees_below_threshold=(minor - current), status=tostring(message.status)
| order by degrees_below_threshold asc

// Alert: Temperatures exceeding minor threshold
| where data_type == "cisco_nexus_environment_temperature"
| extend current = toint(message.current_temp), minor = toint(message.minor_threshold)
| where current >= minor
| project timestamp, module=tostring(message.module), sensor=tostring(message.sensor),
          current_temp=current, minor_threshold=minor,
          degrees_over_threshold=(current - minor), status=tostring(message.status)
| order by degrees_over_threshold desc

// Critical Alert: Temperatures approaching major threshold (within 5°C)
| where data_type == "cisco_nexus_environment_temperature"
| extend current = toint(message.current_temp), major = toint(message.major_threshold)
| where current >= (major - 5)
| project timestamp, module=tostring(message.module), sensor=tostring(message.sensor),
          current_temp=current, major_threshold=major,
          degrees_below_critical=(major - current), status=tostring(message.status)
| order by degrees_below_critical asc

// Critical Alert: Temperatures exceeding major threshold
| where data_type == "cisco_nexus_environment_temperature"
| extend current = toint(message.current_temp), major = toint(message.major_threshold)
| where current >= major
| project timestamp, module=tostring(message.module), sensor=tostring(message.sensor),
          current_temp=current, major_threshold=major,
          degrees_over_critical=(current - major), status=tostring(message.status)

// Alert: Non-OK status sensors
| where data_type == "cisco_nexus_environment_temperature"
| where tostring(message.status) != "Ok"
| project timestamp, module=tostring(message.module), sensor=tostring(message.sensor),
          current_temp=toint(message.current_temp), status=tostring(message.status)
| order by timestamp desc
```

### Temperature Trending and Analysis

```kql
// Temperature trend over time for all sensors
| where data_type == "cisco_nexus_environment_temperature"
| extend current_temp = toint(message.current_temp)
| summarize avg(current_temp), max(current_temp), min(current_temp) by bin(timestamp, 1h), 
            sensor=tostring(message.sensor)
| render timechart

// Temperature trend for CPU sensor
| where data_type == "cisco_nexus_environment_temperature"
| where tostring(message.sensor) == "CPU"
| extend current_temp = toint(message.current_temp)
| project timestamp, current_temp
| render timechart

// Identify sensors with highest average temperature in last 24 hours
| where data_type == "cisco_nexus_environment_temperature"
| where timestamp > ago(24h)
| extend current_temp = toint(message.current_temp)
| summarize avg_temp=avg(current_temp), max_temp=max(current_temp) by 
            module=tostring(message.module), sensor=tostring(message.sensor)
| order by avg_temp desc

// Temperature rate of change (detect rapid increases)
| where data_type == "cisco_nexus_environment_temperature"
| extend current_temp = toint(message.current_temp)
| partition by tostring(message.module), tostring(message.sensor)
(
    order by timestamp asc
    | extend prev_temp = prev(current_temp, 1), prev_time = prev(timestamp, 1)
    | extend temp_change = current_temp - prev_temp
    | extend time_diff_minutes = datetime_diff('minute', timestamp, prev_time)
    | where isnotnull(temp_change)
    | extend rate_of_change = temp_change / time_diff_minutes
    | where abs(rate_of_change) > 0.5  // Alert if temperature changes more than 0.5°C per minute
    | project timestamp, module=tostring(message.module), sensor=tostring(message.sensor),
              current_temp, prev_temp, temp_change, rate_of_change
)
| order by abs(rate_of_change) desc
```

### Comparative Analysis and Reporting

```kql
// Compare current temperature to threshold percentages
| where data_type == "cisco_nexus_environment_temperature"
| extend current = toint(message.current_temp), 
         minor = toint(message.minor_threshold),
         major = toint(message.major_threshold)
| extend minor_pct = round(100.0 * current / minor, 1),
         major_pct = round(100.0 * current / major, 1)
| summarize arg_max(timestamp, *) by tostring(message.module), tostring(message.sensor)
| project module=tostring(message.module), sensor=tostring(message.sensor),
          current_temp=current, minor_threshold=minor, major_threshold=major,
          pct_of_minor=minor_pct, pct_of_major=major_pct, status=tostring(message.status)
| order by pct_of_major desc

// Temperature distribution by sensor type
| where data_type == "cisco_nexus_environment_temperature"
| extend current_temp = toint(message.current_temp)
| summarize count(), avg(current_temp), min(current_temp), max(current_temp) 
            by sensor=tostring(message.sensor)
| order by avg_current_temp desc

// Daily temperature summary report
| where data_type == "cisco_nexus_environment_temperature"
| where date == "2025-01-20"  // Replace with desired date
| extend current_temp = toint(message.current_temp)
| summarize entries=count(), 
            avg_temp=round(avg(current_temp), 1),
            max_temp=max(current_temp),
            min_temp=min(current_temp),
            alerts=countif(tostring(message.status) != "Ok")
            by module=tostring(message.module), sensor=tostring(message.sensor)
| order by max_temp desc

// Identify outliers using percentiles
| where data_type == "cisco_nexus_environment_temperature"
| extend current_temp = toint(message.current_temp)
| summarize p50=percentile(current_temp, 50), 
            p95=percentile(current_temp, 95),
            p99=percentile(current_temp, 99)
            by sensor=tostring(message.sensor)
| order by p99 desc
```

### Actionable Alerting Rules

These queries can be used to create automated alerts in Azure Monitor:

**Rule 1: Minor Threshold Breach**
```kql
| where data_type == "cisco_nexus_environment_temperature"
| extend current = toint(message.current_temp), minor = toint(message.minor_threshold)
| where current >= minor
| summarize count() by module=tostring(message.module), sensor=tostring(message.sensor)
```
Alert when: Result count > 0
Severity: Warning

**Rule 2: Major Threshold Breach**
```kql
| where data_type == "cisco_nexus_environment_temperature"
| extend current = toint(message.current_temp), major = toint(message.major_threshold)
| where current >= major
| summarize count() by module=tostring(message.module), sensor=tostring(message.sensor)
```
Alert when: Result count > 0
Severity: Critical

**Rule 3: Sustained High Temperature**
```kql
| where data_type == "cisco_nexus_environment_temperature"
| where timestamp > ago(30m)
| extend current = toint(message.current_temp), minor = toint(message.minor_threshold)
| where current >= (minor - 5)
| summarize count() by module=tostring(message.module), sensor=tostring(message.sensor)
| where count_ >= 3  // At least 3 readings in last 30 minutes
```
Alert when: Result count > 0
Severity: Warning

## Error Handling

The parser handles various error conditions:
- Invalid input files
- Malformed temperature table entries
- VSH execution failures
- JSON encoding errors

## Compatibility

Tested with Cisco Nexus switches running NX-OS. The parser should work with various Nexus models (including Nexus 9000, 7000, 5000, 3000 series) that support the standard `show environment temperature` command format.

## Related Parsers

This parser is part of the Cisco Nexus parser suite:
- `environment_power_parser`: Power supply monitoring
- `mac_address_parser`: MAC address table parsing
- `interface_counters_parser`: Interface statistics
- `ip_arp_parser`: ARP table parsing
- And more...

## License

See the repository LICENSE file for details.
