# Project SNMP Poll - Go Best Practices

This document outlines the Go best practices specific to the SNMP Poll project.

## Code Organization

- Separate SNMP data collection from data processing
- Use interfaces to mock SNMP client for testing
- Structure packages as:
  - `cmd/snmp_poll` - Main application entry point
  - `pkg/snmppoll` - Core functionality
  - `pkg/config` - Configuration handling
  - `pkg/mibparser` - MIB file parsing

## Error Handling

Examples:

```go
// Good
if err := client.Connect(); err != nil {
    return nil, fmt.Errorf("failed to connect to device %s: %w", device.IP, err)
}

// Avoid
if err := client.Connect(); err != nil {
    return nil, err // Less context for debugging
}
```

## Logging

- Use structured logging
- Include device details in log entries
- Log errors with context
- Use appropriate log levels

```go
// Preferred approach
log.WithFields(log.Fields{
    "device": device.Name,
    "ip":     device.IP,
}).Info("Starting device poll")
```

## JSON Handling

- Define clear struct types with JSON tags
- Validate JSON input
- Handle unmarshaling errors gracefully

## Testing

- Create mock SNMP responses
- Test with various device configurations
- Test error handling paths
- Use table-driven tests

Example:

```go
func TestPollDevice(t *testing.T) {
    tests := []struct {
        name        string
        device      Device
        oids        []string
        expectError bool
        expected    SNMPResult
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Configuration Management

- Use environment variables for sensitive data
- Support both file and CLI configuration
- Validate all configuration values
- Provide helpful error messages for config issues

## Concurrency

- Use worker pools for polling multiple devices
- Implement proper timeout handling
- Use context for cancellation

## Performance

- Batch SNMP requests when possible
- Pre-allocate result slices
- Cache MIB data when appropriate

This guide should be consulted when contributing to the SNMP Poll project.
