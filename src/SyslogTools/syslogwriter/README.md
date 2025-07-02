# Syslog Writer Library

A Go library for writing JSON entries to Linux syslog systems with standardized common fields.

## Installation

```bash
go mod init your-project
go get github.com/arc-switch/syslogwriter
```

Or if using locally:

```bash
# In your go.mod file, add:
replace github.com/arc-switch/syslogwriter => /workspaces/arc-switch/src/SyslogTools/syslogwriter
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/arc-switch/syslogwriter"
)

func main() {
    // Create writer with default settings
    writer, err := syslogwriter.NewWithDefaults("my-app")
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    // Write a JSON entry
    jsonEntry := `{"data_type":"interface_counters","timestamp":"07/01/2025 10:00:01 AM","date":"07/01/2025","message":{"interface":"eth0","status":"up"}}`
    if err := writer.WriteEntry(jsonEntry); err != nil {
        log.Printf("Failed to write entry: %v", err)
    }
}
```

## API Reference

### Types

#### Writer

Main syslog writer instance.

#### Config

Configuration for the syslog writer:

```go
type Config struct {
    Tag          string  // Syslog tag
    MaxEntrySize int     // Maximum entry size in bytes
    Verbose      bool    // Enable verbose logging
}
```

**Note:** The library now uses hardcoded facility (local0) and priority (LOG_INFO) for simplified operation.

### Functions

#### Creating Writers

```go
// Create with custom parameters
func New(tag string, maxEntrySize int, verbose bool) (*Writer, error)

// Create with defaults (4096 max size, no verbose output)
func NewWithDefaults(tag string) (*Writer, error)
```

#### Writing Entries

```go
// Write single entry (uses global verbose setting from config)
func (w *Writer) WriteEntry(jsonEntry string) error
```

#### Utility Functions

```go
// Close the writer
func (w *Writer) Close() error

// Detect if systemd is available (utility function)
func DetectSystemLogger() bool
```

## Examples

### Basic Usage

```go
package main

import (
    "log"
    "github.com/arc-switch/syslogwriter"
)

func main() {
    writer, err := syslogwriter.NewWithDefaults("my-app")
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    jsonEntry := `{"data_type":"system_event","timestamp":"07/01/2025 10:00:01 AM","date":"07/01/2025","message":{"event":"startup","status":"success"}}`
    
    if err := writer.WriteEntry(jsonEntry); err != nil {
        log.Printf("Error: %v", err)
    }
}
```

### Custom Configuration

```go
package main

import (
    "log"
    "github.com/arc-switch/syslogwriter"
)

func main() {
    // Create writer with custom settings
    writer, err := syslogwriter.New("network-monitor", 8192, true) // verbose enabled
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    // Write entry with verbose output enabled
    jsonEntry := `{"data_type":"interface_status","timestamp":"07/01/2025 10:00:01 AM","date":"07/01/2025","message":{"interface":"eth0","status":"up"}}`
    
    if err := writer.WriteEntry(jsonEntry); err != nil {
        log.Printf("Error: %v", err)
    }
}
```

### Processing Multiple Entries

```go
package main

import (
    "encoding/json"
    "log"
    "github.com/arc-switch/syslogwriter"
)

func main() {
    writer, err := syslogwriter.NewWithDefaults("batch-processor")
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    // Process multiple entries individually
    entries := []string{
        `{"data_type":"network_event","timestamp":"07/01/2025 10:00:01 AM","date":"07/01/2025","message":{"event":"connection_opened","port":80}}`,
        `{"data_type":"network_event","timestamp":"07/01/2025 10:00:02 AM","date":"07/01/2025","message":{"event":"connection_closed","port":80}}`,
    }

    for i, entry := range entries {
        if err := writer.WriteEntry(entry); err != nil {
            log.Printf("Failed to write entry %d: %v", i, err)
        }
    }
}
```

### Integration with MAC Address Parser

Here's how you could integrate this library into the MAC address parser:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"
    "github.com/arc-switch/syslogwriter"
)

type MACEntry struct {
    DataType   string                 `json:"data_type"`
    Timestamp  string                 `json:"timestamp"`
    Date       string                 `json:"date"`
    Message    map[string]interface{} `json:"message"`
}

func main() {
    // Initialize syslog writer
    writer, err := syslogwriter.NewWithDefaults("cisco-mac-parser")
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    // Parse MAC addresses (your existing logic here)
    macEntries := []MACEntry{
        {
            DataType:  "cisco_nexus_mac_table",
            Timestamp: time.Now().Format("01/02/2006 03:04:05 PM"),
            Date:      time.Now().Format("01/02/2006"),
            Message: map[string]interface{}{
                "vlan":        "100",
                "mac_address": "00:50:56:c0:00:08",
                "port":        "Eth1/1",
            },
        },
        // ... more entries
    }

    // Convert to JSON and log each entry
    for _, entry := range macEntries {
        jsonData, err := json.Marshal(entry)
        if err != nil {
            log.Printf("Failed to marshal entry: %v", err)
            continue
        }

        if err := writer.WriteEntry(string(jsonData)); err != nil {
            log.Printf("Failed to log entry: %v", err)
        }
    }

    log.Printf("Processed %d MAC table entries", len(macEntries))
}
```

## Required JSON Fields

All JSON entries must contain these four required fields:

- `data_type`: Data type identifier (can serve as tag identifier)  
- `timestamp`: Full timestamp with date and time
- `date`: Date in MM/DD/YYYY format
- `message`: JSON object containing parser-specific data (structure varies by parser)

### Message Field Structure

The `message` field must be a JSON object and will contain parser-specific data. For example:

```json
{
  "data_type": "interface_counters",
  "timestamp": "2024-01-15T10:30:00Z", 
  "date": "2024-01-15",
  "message": {
    "interface": "eth0",
    "counters": {
      "rx_packets": 12345,
      "tx_packets": 67890
    }
  }
}
```

## Features

- **Simplified API**: Easy-to-use interface with hardcoded facility (local0) and priority (LOG_INFO)
- **Single Entry Writing**: Focused on single JSON entry processing
- **Automatic Validation**: Validates required JSON fields before writing
- **Size Management**: Automatic truncation of oversized entries
- **Verbose Logging**: Optional verbose output for debugging
- **Cross-Platform**: Works on Linux systems with syslog support

## Error Handling

The library provides robust error handling:

- Validates JSON format and required fields
- Returns detailed error messages for debugging
- Truncates oversized entries automatically
- Continues operation on validation failures

## Thread Safety

The library is designed to be thread-safe for concurrent use across goroutines. The simplified design eliminates previous race conditions around statistics tracking.
