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
    jsonEntry := `{"data_type":"test_entry","timestamp":"06/24/2025 10:00:01 AM","date":"06/24/2025","message":"Hello World"}`
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
    Priority     syslog.Priority  // Combined facility and priority
    Tag          string           // Syslog tag
    Facility     string           // Facility name (for reference)
    UseSystemd   bool            // Whether systemd is detected
    MaxEntrySize int             // Maximum entry size in bytes
}
```

#### Statistics

Processing statistics:

```go
type Statistics struct {
    TotalEntries     int
    SuccessEntries   int
    FailedEntries    int
    TruncatedEntries int
    StartTime        time.Time
}
```

### Functions

#### Creating Writers

```go
// Create with custom configuration
func New(config *Config) (*Writer, error)

// Create with defaults (local0, info priority, 4096 max size)
func NewWithDefaults(tag string) (*Writer, error)
```

#### Writing Entries

```go
// Write single entry
func (w *Writer) WriteEntry(jsonEntry string) error

// Write single entry with verbose output
func (w *Writer) WriteEntryWithVerbose(jsonEntry string, verbose bool) error

// Write multiple entries
func (w *Writer) WriteEntries(jsonEntries []string) error

// Write from io.Reader (file, stdin, etc.)
func (w *Writer) WriteFromReader(reader io.Reader) error
```

#### Utility Functions

```go
// Validate entry without writing
func (w *Writer) ValidateEntry(jsonEntry string) error

// Get processing statistics
func (w *Writer) GetStatistics() *Statistics

// Reset statistics
func (w *Writer) ResetStatistics()

// Close the writer
func (w *Writer) Close() error
```

#### Helper Functions

```go
// Parse priority string to syslog.Priority
func ParseSyslogPriority(priority string) (syslog.Priority, error)

// Parse facility string to syslog.Priority
func ParseSyslogFacility(facility string) (syslog.Priority, error)

// Detect if systemd is available
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

    jsonEntry := `{"data_type":"system_event","timestamp":"06/24/2025 10:00:01 AM","date":"06/24/2025","event":"startup","status":"success"}`
    
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
    "log/syslog"
    "github.com/arc-switch/syslogwriter"
)

func main() {
    // Parse facility and priority
    facility, _ := syslogwriter.ParseSyslogFacility("local1")
    priority, _ := syslogwriter.ParseSyslogPriority("warning")
    
    config := &syslogwriter.Config{
        Priority:     facility | priority,
        Tag:          "network-monitor",
        Facility:     "local1",
        UseSystemd:   syslogwriter.DetectSystemLogger(),
        MaxEntrySize: 8192,
    }

    writer, err := syslogwriter.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    // Write multiple entries
    entries := []string{
        `{"data_type":"interface_status","timestamp":"06/24/2025 10:00:01 AM","date":"06/24/2025","interface":"eth0","status":"up"}`,
        `{"data_type":"interface_status","timestamp":"06/24/2025 10:00:02 AM","date":"06/24/2025","interface":"eth1","status":"down"}`,
    }

    if err := writer.WriteEntries(entries); err != nil {
        log.Printf("Some entries failed: %v", err)
    }

    // Print statistics
    stats := writer.GetStatistics()
    log.Printf("Processed %d entries, %d successful, %d failed", 
        stats.TotalEntries, stats.SuccessEntries, stats.FailedEntries)
}
```

### Processing from File

```go
package main

import (
    "log"
    "os"
    "github.com/arc-switch/syslogwriter"
)

func main() {
    writer, err := syslogwriter.NewWithDefaults("file-processor")
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()

    file, err := os.Open("data.json")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    if err := writer.WriteFromReaderWithVerbose(file, true); err != nil {
        log.Printf("Processing completed with errors: %v", err)
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
    DataType   string `json:"data_type"`
    Timestamp  string `json:"timestamp"`
    Date       string `json:"date"`
    VLAN       string `json:"vlan"`
    MACAddress string `json:"mac_address"`
    Port       string `json:"port"`
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
            DataType:   "cisco_nexus_mac_table",
            Timestamp:  time.Now().Format("01/02/2006 03:04:05 PM"),
            Date:       time.Now().Format("01/02/2006"),
            VLAN:       "100",
            MACAddress: "00:50:56:c0:00:08",
            Port:       "Eth1/1",
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

    // Print statistics
    stats := writer.GetStatistics()
    fmt.Printf("Logged %d MAC table entries\n", stats.SuccessEntries)
}
```

## Required JSON Fields

All JSON entries must contain:

- `data_type`: Data type identifier (can serve as tag identifier)
- `timestamp`: Full timestamp with date and time  
- `date`: Date in MM/DD/YYYY format

## Error Handling

The library provides robust error handling:

- Validates JSON format and required fields
- Continues processing on individual entry failures
- Tracks detailed statistics
- Truncates oversized entries automatically

## Thread Safety

The library is designed to be thread-safe for concurrent use across goroutines, except for the `Statistics` counters, which are not protected by mutexes or atomic operations and may result in data races if accessed concurrently. If you require accurate statistics in concurrent scenarios, ensure to add synchronization (e.g., mutex or atomic operations) around statistics updates.
