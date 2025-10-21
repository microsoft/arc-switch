# Cisco Nexus System Uptime Parser

This tool parses the output of the `show system uptime` command from Cisco Nexus switches and converts it to structured JSON format for system availability monitoring and analysis.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Multiple Format Support**: Handles both text and JSON-pretty output formats
- **Comprehensive Uptime Data**: Captures system start time, system uptime, and kernel uptime with breakdown by days, hours, minutes, and seconds
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **Error Handling**: Robust parsing with proper error handling
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

### Option 1: Build from Source

```bash
# Navigate to the parser directory
cd src/SwitchOutput/Cisco/Nexus/10/system-uptime

# Build the binary
go build -o system_uptime_parser system_uptime_parser.go
```

### Option 2: Use via Unified Parser

The system uptime parser is integrated into the unified cisco-parser. See the parent cisco-parser directory for installation instructions.

## Usage

### Parse from Input File

```bash
# Parse system uptime data from a text file
./system_uptime_parser -input ../show-system-uptime.txt -output output.json

# Parse and output to stdout
./system_uptime_parser -input ../show-system-uptime.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./system_uptime_parser -commands ../commands.json -output output.json
```

### Using the Unified Parser

```bash
# From the cisco-parser directory
./cisco-parser -parser system-uptime -input ../show-system-uptime.txt -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show system uptime` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool supports two input formats:

### Text Format

```text
CONTOSO-DC1-TOR-01# show system uptime
System start time:          Fri Oct 17 13:50:28 2025
System uptime:              2 days, 22 hours, 1 minutes, 43 seconds
Kernel uptime:              2 days, 22 hours, 3 minutes, 51 seconds
```

### JSON Format

```json
{
    "sys_st_time": "Fri Oct 17 13:50:28 2025",
    "sys_up_days": "2",
    "sys_up_hrs": "22",
    "sys_up_mins": "1",
    "sys_up_secs": "51",
    "kn_up_days": "2",
    "kn_up_hrs": "22",
    "kn_up_mins": "3",
    "kn_up_secs": "58"
}
```

## Output Format

The parser outputs JSON with a standardized structure compatible with the syslogwriter library:

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_system_uptime",
  "timestamp": "2025-10-21T22:30:45Z",
  "date": "2025-10-21",
  "message": {
    "system_start_time": "Fri Oct 17 13:50:28 2025",
    "system_uptime_days": "2",
    "system_uptime_hours": "22",
    "system_uptime_minutes": "1",
    "system_uptime_seconds": "43",
    "system_uptime_total": "2 days, 22 hours, 1 minutes, 43 seconds",
    "kernel_uptime_days": "2",
    "kernel_uptime_hours": "22",
    "kernel_uptime_minutes": "3",
    "kernel_uptime_seconds": "51",
    "kernel_uptime_total": "2 days, 22 hours, 3 minutes, 51 seconds"
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_system_uptime"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-10-21T22:30:45Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all system uptime data

### Message Fields

The `message` field contains all system uptime-specific data:

- `system_start_time`: Date and time when the system was started
- `system_uptime_days`: Number of days the system has been up
- `system_uptime_hours`: Number of hours in the system uptime
- `system_uptime_minutes`: Number of minutes in the system uptime
- `system_uptime_seconds`: Number of seconds in the system uptime
- `system_uptime_total`: Human-readable total system uptime
- `kernel_uptime_days`: Number of days the kernel has been up
- `kernel_uptime_hours`: Number of hours in the kernel uptime
- `kernel_uptime_minutes`: Number of minutes in the kernel uptime
- `kernel_uptime_seconds`: Number of seconds in the kernel uptime
- `kernel_uptime_total`: Human-readable total kernel uptime

## KQL Query Examples

The `data_type` field enables easy filtering in Azure Log Analytics or other systems that use KQL (Kusto Query Language).

### Basic Filtering

```kql
// Filter for Cisco Nexus system uptime entries only
| where data_type == "cisco_nexus_system_uptime"
```

### Uptime Trending Over Time

```kql
// Track system uptime trends over time
| where data_type == "cisco_nexus_system_uptime"
| extend uptime_days = toint(message.system_uptime_days)
| extend uptime_hours = toint(message.system_uptime_hours)
| extend total_uptime_hours = uptime_days * 24 + uptime_hours
| project timestamp, system_name = extract(@"(\w+-\w+-\w+-\w+)", 1, message.system_start_time), total_uptime_hours
| render timechart
```

### Scorecard: Total Days Up

```kql
// Scorecard showing total days the system has been running
| where data_type == "cisco_nexus_system_uptime"
| extend uptime_days = toint(message.system_uptime_days)
| summarize MaxUptimeDays = max(uptime_days), AvgUptimeDays = avg(uptime_days)
| project MaxUptimeDays, AvgUptimeDays
```

### System vs Kernel Uptime Comparison

```kql
// Compare system uptime vs kernel uptime to detect mismatches
| where data_type == "cisco_nexus_system_uptime"
| extend system_days = toint(message.system_uptime_days)
| extend system_hours = toint(message.system_uptime_hours)
| extend kernel_days = toint(message.kernel_uptime_days)
| extend kernel_hours = toint(message.kernel_uptime_hours)
| extend system_total_hours = system_days * 24 + system_hours
| extend kernel_total_hours = kernel_days * 24 + kernel_hours
| extend uptime_difference_hours = kernel_total_hours - system_total_hours
| project timestamp, date, system_total_hours, kernel_total_hours, uptime_difference_hours
| where uptime_difference_hours != 0
```

### Identify Recent Restarts

```kql
// Find systems that have been restarted recently (uptime < 1 day)
| where data_type == "cisco_nexus_system_uptime"
| extend uptime_days = toint(message.system_uptime_days)
| where uptime_days < 1
| project timestamp, date, message.system_start_time, message.system_uptime_total
| order by timestamp desc
```

### Identify Outlier Values

```kql
// Find systems with unusually high or low uptime (statistical outliers)
| where data_type == "cisco_nexus_system_uptime"
| extend uptime_days = toint(message.system_uptime_days)
| extend uptime_hours = toint(message.system_uptime_hours)
| extend total_uptime_hours = uptime_days * 24 + uptime_hours
| summarize avg_uptime = avg(total_uptime_hours), stdev_uptime = stdev(total_uptime_hours) by bin(timestamp, 1h)
| extend lower_bound = avg_uptime - (2 * stdev_uptime)
| extend upper_bound = avg_uptime + (2 * stdev_uptime)
| join kind=inner (
    | where data_type == "cisco_nexus_system_uptime"
    | extend uptime_days = toint(message.system_uptime_days)
    | extend uptime_hours = toint(message.system_uptime_hours)
    | extend total_uptime_hours = uptime_days * 24 + uptime_hours
) on $left.timestamp == $right.timestamp
| where total_uptime_hours < lower_bound or total_uptime_hours > upper_bound
| project timestamp, date, message.system_uptime_total, total_uptime_hours, avg_uptime, lower_bound, upper_bound
```

### Uptime Distribution

```kql
// Analyze uptime distribution across different time ranges
| where data_type == "cisco_nexus_system_uptime"
| extend uptime_days = toint(message.system_uptime_days)
| extend uptime_category = case(
    uptime_days < 1, "Less than 1 day",
    uptime_days < 7, "1-7 days",
    uptime_days < 30, "1-4 weeks",
    uptime_days < 90, "1-3 months",
    uptime_days < 365, "3-12 months",
    "Over 1 year"
)
| summarize count() by uptime_category
| order by count_ desc
```

### System Availability Monitoring

```kql
// Calculate system availability percentage (assuming daily data collection)
| where data_type == "cisco_nexus_system_uptime"
| extend uptime_days = toint(message.system_uptime_days)
| extend uptime_hours = toint(message.system_uptime_hours)
| extend uptime_minutes = toint(message.system_uptime_minutes)
| extend total_uptime_minutes = (uptime_days * 24 * 60) + (uptime_hours * 60) + uptime_minutes
| summarize total_minutes = sum(total_uptime_minutes) by date
| extend availability_percentage = (total_minutes / (24.0 * 60)) * 100
| project date, availability_percentage
| render timechart
```

### Alert on Unexpected Restarts

```kql
// Alert when system uptime decreases (indicating a restart)
| where data_type == "cisco_nexus_system_uptime"
| extend uptime_days = toint(message.system_uptime_days)
| extend uptime_hours = toint(message.system_uptime_hours)
| extend total_uptime_hours = uptime_days * 24 + uptime_hours
| serialize
| extend prev_uptime = prev(total_uptime_hours)
| where total_uptime_hours < prev_uptime
| project timestamp, date, message.system_start_time, message.system_uptime_total, restart_detected = "Yes"
```

## Integration with Commands.json

When using the `-commands` option, the tool looks for an entry with `"name": "system-uptime"` in the commands.json file:

```json
{
  "commands": [
    {
      "name": "system-uptime",
      "command": "show system uptime"
    }
  ]
}
```

## Testing

Run the test suite to verify the parser functionality:

```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestParseSystemUptimeText
```

The test suite covers:
- Text format parsing
- JSON format parsing
- Invalid input handling
- JSON serialization/deserialization
- UnifiedParser interface implementation
- Commands JSON file parsing

## Error Handling

The tool includes comprehensive error handling for:

- Invalid input files
- Malformed system uptime output
- Missing or incomplete data
- Missing commands.json entries
- Network connectivity issues (when using direct switch communication)

## Requirements

- Go 1.21 or later
- For direct switch access: `vsh` command-line tool available on Cisco Nexus switches

## Use Cases

This parser is useful for:

1. **System Availability Monitoring**: Track how long systems have been running without restarts
2. **Restart Detection**: Identify when systems have been restarted
3. **Trend Analysis**: Analyze uptime patterns over time
4. **Capacity Planning**: Understand system stability and plan maintenance windows
5. **Compliance Reporting**: Generate reports on system uptime for SLA compliance
6. **Anomaly Detection**: Identify systems with unusual uptime patterns
7. **Kernel vs System Comparison**: Detect potential issues when kernel and system uptimes diverge significantly

## Integration Notes

### Cisco Parser Integration

This parser is integrated into the unified cisco-parser system, which provides a single binary for parsing multiple Cisco Nexus command outputs. The integration includes:

- Registration in the cisco-parser's parser registry
- Support for the `-parser system-uptime` flag
- Unified error handling and output formatting
- Consistent JSON structure across all parsers

### Syslog Integration

The standardized JSON output is designed to work seamlessly with syslog forwarding systems:

- Each uptime check produces a single JSON object
- The `data_type` field enables easy filtering and routing
- ISO 8601 timestamps ensure proper time-series analysis
- The `message` field encapsulates all uptime-specific data

## Troubleshooting

### Parser Returns Error: "could not parse system uptime data"

This error occurs when the input doesn't match expected formats. Verify:
- The input file contains output from `show system uptime` command
- The output is from a Cisco Nexus switch
- The file is not corrupted or truncated

### JSON Format Not Recognized

If the parser doesn't recognize JSON format:
- Ensure the JSON is valid (use a JSON validator)
- Check that the JSON contains the expected keys (sys_st_time, sys_up_days, etc.)
- Verify there are no extra characters or formatting issues

### Command Not Found in Commands.json

When using `-commands` flag:
- Verify the commands.json file exists and is readable
- Ensure the JSON contains an entry with `"name": "system-uptime"`
- Check the JSON syntax is valid
