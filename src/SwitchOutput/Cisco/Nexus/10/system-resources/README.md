# Cisco Nexus System Resources Parser

This tool parses the output of the `show system resources` command from Cisco Nexus switches and converts it to structured JSON format for monitoring and analysis.

## Features

- **Dual Input Modes**: Parse from input file or execute commands directly on switch
- **Comprehensive Metrics**: Captures load averages, process counts, CPU usage (overall and per-core), memory statistics, and system status
- **Per-CPU Core Statistics**: Individual metrics for each CPU core (user, kernel, idle percentages)
- **Memory Analysis**: Total, used, free memory along with kernel buffers and cache information
- **Standardized JSON Output**: Uses the project's standardized JSON structure compatible with syslogwriter
- **JSON Lines Format**: Produces JSON objects for streaming/logging compatibility
- **Error Handling**: Robust parsing with proper handling of malformed data
- **CLI Integration**: Uses `vsh` for direct switch communication

## Installation

```bash
# Build the binary
go build -o system_resources_parser system_resources_parser.go
```

## Usage

### Parse from Input File

```bash
# Parse system resources from a text file
./system_resources_parser -input show-system-resources.txt -output output.json

# Parse and output to stdout
./system_resources_parser -input show-system-resources.txt
```

### Get Data Directly from Switch

```bash
# Execute commands on switch using commands.json
./system_resources_parser -commands ../commands.json -output output.json
```

### Command Line Options

- `-input <file>`: Input file containing `show system resources` output
- `-output <file>`: Output file for JSON data (optional, defaults to stdout)
- `-commands <file>`: Commands JSON file (used when no input file is specified)
- `-help`: Show help message

## Input Format

The tool expects standard Cisco Nexus `show system resources` output, which includes:

1. **Load Averages**: 1-minute, 5-minute, and 15-minute load averages
2. **Process Information**: Total and running process counts
3. **Overall CPU States**: User, kernel, and idle percentages across all CPUs
4. **Per-CPU States**: Individual statistics for each CPU core (CPU0, CPU1, etc.)
5. **Memory Usage**: Total, used, and free memory in KB
6. **Kernel Memory**: Vmalloc, buffers, and cache information
7. **Memory Status**: Current memory status indicator (OK, WARNING, etc.)

Example input:
```
CONTOSO-DC1-TOR-01# show system resources
Load average:   1 minute: 0.60   5 minutes: 0.63   15 minutes: 0.73
Processes   :   954 total, 5 running
CPU states  :   11.43% user,   4.52% kernel,   84.04% idle
        CPU0 states  :   28.71% user,   6.93% kernel,   64.35% idle
        CPU1 states  :   0.00% user,   0.00% kernel,   100.00% idle
        ...
Memory usage:   24538812K total,   10765396K used,   13773416K free
Current memory status: OK
```

## Output Format

The parser outputs JSON Lines format with a standardized structure compatible with the syslogwriter library.

### Standardized Structure

```json
{
  "data_type": "cisco_nexus_system_resources",
  "timestamp": "2025-10-21T22:45:00Z",
  "date": "2025-10-21",
  "message": {
    // System resources-specific fields here
  }
}
```

### Required Fields

- `data_type`: Always "cisco_nexus_system_resources"
- `timestamp`: Processing timestamp in ISO 8601 format (e.g., "2025-10-21T22:45:00Z")
- `date`: Processing date in ISO format (YYYY-MM-DD)
- `message`: JSON object containing all system resources data

### Message Fields

The `message` field contains all system resources data:

```json
{
  "load_avg_1min": "0.60",
  "load_avg_5min": "0.63",
  "load_avg_15min": "0.73",
  "processes_total": 954,
  "processes_running": 5,
  "cpu_state_user": "11.43",
  "cpu_state_kernel": "4.52",
  "cpu_state_idle": "84.04",
  "cpu_usage": [
    {
      "cpuid": "0",
      "user": "28.71",
      "kernel": "6.93",
      "idle": "64.35"
    },
    {
      "cpuid": "1",
      "user": "0.00",
      "kernel": "0.00",
      "idle": "100.00"
    }
  ],
  "memory_usage_total": 24538812,
  "memory_usage_used": 10765396,
  "memory_usage_free": 13773416,
  "kernel_vmalloc_total": 0,
  "kernel_vmalloc_free": 0,
  "kernel_buffers": 59712,
  "kernel_cached": 5993960,
  "current_memory_status": "OK"
}
```

## Integration with Cisco Parser

This parser is integrated into the unified cisco-parser binary. To use it through the unified parser:

```bash
# Navigate to the cisco-parser directory
cd ../cisco-parser

# Build the unified parser
make build

# Use the system-resources parser
./build/cisco-parser -parser system-resources -input ../show-system-resources.txt -output output.json
```

## Testing

Run the test suite to verify the parser:

```bash
go test -v
```

The tests validate:
- Parsing of all system resources metrics
- JSON serialization/deserialization
- UnifiedParser interface implementation
- Individual CPU core parsing
- Memory statistics extraction

## KQL Query Examples

The `data_type` field enables easy filtering in Azure Log Analytics or other systems that use KQL (Kusto Query Language).

### Memory Trending

Monitor memory usage trends over time:

```kql
// Memory usage percentage over time
| where data_type == "cisco_nexus_system_resources"
| extend memory_usage_percent = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| project timestamp, memory_usage_percent, message.current_memory_status
| render timechart

// Memory usage by component
| where data_type == "cisco_nexus_system_resources"
| extend used_gb = todouble(message.memory_usage_used) / 1024 / 1024
| extend free_gb = todouble(message.memory_usage_free) / 1024 / 1024
| extend cached_gb = todouble(message.kernel_cached) / 1024 / 1024
| extend buffers_gb = todouble(message.kernel_buffers) / 1024 / 1024
| project timestamp, used_gb, free_gb, cached_gb, buffers_gb
| render timechart

// Average memory usage by device
| where data_type == "cisco_nexus_system_resources"
| extend memory_usage_percent = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| summarize avg_memory_usage = avg(memory_usage_percent) by bin(timestamp, 1h)
| render timechart
```

### CPU Trending - Overall and Per-Core

Track CPU utilization across all cores and individually:

```kql
// Overall CPU utilization over time
| where data_type == "cisco_nexus_system_resources"
| extend cpu_user = todouble(message.cpu_state_user)
| extend cpu_kernel = todouble(message.cpu_state_kernel)
| extend cpu_idle = todouble(message.cpu_state_idle)
| project timestamp, cpu_user, cpu_kernel, cpu_idle
| render timechart

// Total CPU usage (100 - idle)
| where data_type == "cisco_nexus_system_resources"
| extend total_cpu_usage = 100 - todouble(message.cpu_state_idle)
| project timestamp, total_cpu_usage
| render timechart

// Per-CPU core usage
| where data_type == "cisco_nexus_system_resources"
| mv-expand cpu_core = message.cpu_usage
| extend cpuid = tostring(cpu_core.cpuid)
| extend user = todouble(cpu_core.user)
| extend kernel = todouble(cpu_core.kernel)
| extend idle = todouble(cpu_core.idle)
| extend cpu_usage = user + kernel
| project timestamp, cpuid, cpu_usage
| render timechart

// Average CPU usage by core
| where data_type == "cisco_nexus_system_resources"
| mv-expand cpu_core = message.cpu_usage
| extend cpuid = tostring(cpu_core.cpuid)
| extend cpu_usage = todouble(cpu_core.user) + todouble(cpu_core.kernel)
| summarize avg_cpu_usage = avg(cpu_usage) by cpuid, bin(timestamp, 5m)
| render timechart
```

### Idle, Kernel, User CPU Analysis by Core

Detailed breakdown of CPU time allocation:

```kql
// User vs Kernel CPU time by core
| where data_type == "cisco_nexus_system_resources"
| mv-expand cpu_core = message.cpu_usage
| extend cpuid = tostring(cpu_core.cpuid)
| extend user = todouble(cpu_core.user)
| extend kernel = todouble(cpu_core.kernel)
| project timestamp, cpuid, user, kernel
| render timechart

// Idle CPU percentage by core
| where data_type == "cisco_nexus_system_resources"
| mv-expand cpu_core = message.cpu_usage
| extend cpuid = tostring(cpu_core.cpuid)
| extend idle = todouble(cpu_core.idle)
| project timestamp, cpuid, idle
| render timechart

// Identify cores with low idle time (high utilization)
| where data_type == "cisco_nexus_system_resources"
| mv-expand cpu_core = message.cpu_usage
| extend cpuid = tostring(cpu_core.cpuid)
| extend idle = todouble(cpu_core.idle)
| where idle < 20  // Less than 20% idle means >80% utilization
| summarize count() by cpuid, bin(timestamp, 1h)
| order by count_ desc

// CPU time distribution (stacked view)
| where data_type == "cisco_nexus_system_resources"
| mv-expand cpu_core = message.cpu_usage
| extend cpuid = tostring(cpu_core.cpuid)
| extend user = todouble(cpu_core.user)
| extend kernel = todouble(cpu_core.kernel)
| extend idle = todouble(cpu_core.idle)
| summarize avg_user=avg(user), avg_kernel=avg(kernel), avg_idle=avg(idle) by cpuid
```

### Alert on Status Changes

Monitor for memory status changes from normal:

```kql
// Alert when memory status is not OK
| where data_type == "cisco_nexus_system_resources"
| where message.current_memory_status != "OK"
| project timestamp, message.current_memory_status, message.memory_usage_used, message.memory_usage_total
| order by timestamp desc

// Memory status changes over time
| where data_type == "cisco_nexus_system_resources"
| summarize count() by message.current_memory_status, bin(timestamp, 1h)
| render timechart

// Alert on high memory usage (>85%)
| where data_type == "cisco_nexus_system_resources"
| extend memory_usage_percent = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| where memory_usage_percent > 85
| project timestamp, memory_usage_percent, message.current_memory_status
| order by timestamp desc

// Alert on high CPU usage (>80% for any core)
| where data_type == "cisco_nexus_system_resources"
| mv-expand cpu_core = message.cpu_usage
| extend cpuid = tostring(cpu_core.cpuid)
| extend cpu_usage = 100 - todouble(cpu_core.idle)
| where cpu_usage > 80
| project timestamp, cpuid, cpu_usage
| order by timestamp desc
```

### Trend Metrics Over Time

Analyze trends and patterns:

```kql
// Load average trends
| where data_type == "cisco_nexus_system_resources"
| extend load_1min = todouble(message.load_avg_1min)
| extend load_5min = todouble(message.load_avg_5min)
| extend load_15min = todouble(message.load_avg_15min)
| project timestamp, load_1min, load_5min, load_15min
| render timechart

// Process count trends
| where data_type == "cisco_nexus_system_resources"
| project timestamp, 
          processes_total = message.processes_total,
          processes_running = message.processes_running
| render timechart

// Memory trend with multiple components
| where data_type == "cisco_nexus_system_resources"
| extend used_percent = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| extend free_percent = (todouble(message.memory_usage_free) / todouble(message.memory_usage_total)) * 100
| extend cached_percent = (todouble(message.kernel_cached) / todouble(message.memory_usage_total)) * 100
| project timestamp, used_percent, free_percent, cached_percent
| render timechart

// Hourly average metrics
| where data_type == "cisco_nexus_system_resources"
| extend cpu_usage = 100 - todouble(message.cpu_state_idle)
| extend memory_usage = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| extend load_avg = todouble(message.load_avg_1min)
| summarize 
    avg_cpu = avg(cpu_usage),
    avg_memory = avg(memory_usage),
    avg_load = avg(load_avg)
    by bin(timestamp, 1h)
| render timechart
```

### Identify Outlier Values

Detect anomalous behavior:

```kql
// Find memory usage outliers using percentiles
| where data_type == "cisco_nexus_system_resources"
| extend memory_usage_percent = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| summarize 
    p50 = percentile(memory_usage_percent, 50),
    p95 = percentile(memory_usage_percent, 95),
    p99 = percentile(memory_usage_percent, 99),
    max_usage = max(memory_usage_percent)
| project p50, p95, p99, max_usage

// Find CPU usage outliers
| where data_type == "cisco_nexus_system_resources"
| extend total_cpu = 100 - todouble(message.cpu_state_idle)
| summarize 
    p50 = percentile(total_cpu, 50),
    p95 = percentile(total_cpu, 95),
    p99 = percentile(total_cpu, 99),
    max_cpu = max(total_cpu)
| project p50, p95, p99, max_cpu

// Identify unusual load average spikes
| where data_type == "cisco_nexus_system_resources"
| extend load_1min = todouble(message.load_avg_1min)
| where load_1min > 2.0  // Adjust threshold based on your system
| project timestamp, load_1min, message.processes_running
| order by load_1min desc

// Find per-CPU core outliers
| where data_type == "cisco_nexus_system_resources"
| mv-expand cpu_core = message.cpu_usage
| extend cpuid = tostring(cpu_core.cpuid)
| extend cpu_usage = 100 - todouble(cpu_core.idle)
| summarize 
    avg_usage = avg(cpu_usage),
    max_usage = max(cpu_usage),
    min_usage = min(cpu_usage)
    by cpuid
| extend usage_variance = max_usage - min_usage
| order by usage_variance desc

// Detect sudden changes in resource usage
| where data_type == "cisco_nexus_system_resources"
| extend memory_percent = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| extend cpu_usage = 100 - todouble(message.cpu_state_idle)
| order by timestamp asc
| extend prev_memory = prev(memory_percent)
| extend prev_cpu = prev(cpu_usage)
| extend memory_change = abs(memory_percent - prev_memory)
| extend cpu_change = abs(cpu_usage - prev_cpu)
| where memory_change > 10 or cpu_change > 20  // Thresholds for significant change
| project timestamp, memory_change, cpu_change, memory_percent, cpu_usage
| order by timestamp desc
```

### Comparative Analysis

Compare metrics across time periods:

```kql
// Compare current vs previous hour
| where data_type == "cisco_nexus_system_resources"
| extend memory_percent = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| extend cpu_usage = 100 - todouble(message.cpu_state_idle)
| summarize 
    avg_memory = avg(memory_percent),
    avg_cpu = avg(cpu_usage),
    avg_load = avg(todouble(message.load_avg_1min))
    by bin(timestamp, 1h)
| extend prev_memory = prev(avg_memory)
| extend prev_cpu = prev(avg_cpu)
| extend memory_diff = avg_memory - prev_memory
| extend cpu_diff = avg_cpu - prev_cpu
| project timestamp, avg_memory, memory_diff, avg_cpu, cpu_diff, avg_load

// Week-over-week comparison
| where data_type == "cisco_nexus_system_resources"
| extend week = startofweek(timestamp)
| extend memory_percent = (todouble(message.memory_usage_used) / todouble(message.memory_usage_total)) * 100
| summarize avg_memory_usage = avg(memory_percent) by week
| order by week desc
| take 4  // Last 4 weeks
```

## Commands JSON Format

When using the `-commands` option, add the system-resources command to your JSON file:

```json
{
  "commands": [
    {
      "name": "system-resources",
      "command": "show system resources"
    }
  ]
}
```

## Building

```bash
go build -o system_resources_parser system_resources_parser.go
```

## Testing with Sample File

To test with the included sample file:

```bash
go build -o system_resources_parser system_resources_parser.go
./system_resources_parser -input ../show-system-resources.txt
```

## Validation

To validate the output format, you can use the validation script if available in the repository:

```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
VAL_PARSER=$(find "$REPO_ROOT" -name "validate-parser-output.sh")

# Generate output and validate it
./system_resources_parser -input ../show-system-resources.txt | $VAL_PARSER

# Or validate a saved file
./system_resources_parser -input ../show-system-resources.txt -output output.json
$VAL_PARSER output.json
```

## Troubleshooting

### Common Issues

1. **No data parsed**: Verify the input file format matches the expected Cisco Nexus output
2. **Missing CPU cores**: Check that all CPU core entries are present in the input
3. **Incorrect memory values**: Ensure memory values are in KB format

### Debug Mode

For debugging, you can examine the raw JSON output:

```bash
./system_resources_parser -input ../show-system-resources.txt | jq '.'
```

## Contributing

When contributing to this parser:

1. Ensure all tests pass: `go test -v`
2. Follow the existing code style
3. Update tests for any new features
4. Update this README with any new KQL queries or usage examples
