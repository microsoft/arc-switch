# SNMP Poll Tool

A Go utility to retrieve SNMP MIB data from network devices using a predefined set of MIB entries stored in JSON format.

## Features

- Polls network devices using SNMP GET operations
- Uses predefined OID configurations from a JSON file
- Supports a directory of MIB files for OID resolution
- Configurable polling interval
- Outputs structured JSON data

## Usage

```shell
./snmp_poll -config /path/to/config -mibs /path/to/mibs -interval 60
```

## Configuration

### Directory Structure

```text
├── config/
│   ├── devices.json   # Device configurations
│   └── oids.json      # OID definitions
├── mibs/              # MIB files directory
└── snmp_poll          # Executable
```

### Devices Configuration (devices.json)

```json
{
  "devices": [
    {
      "name": "Switch1",
      "ip": "192.168.1.10",
      "port": 161,
      "version": "v2c",
      "community": "public",
      "timeout": 5
    }
  ]
}
```

### OIDs Configuration (oids.json)

```json
{
  "oids": [
    {
      "name": "sysName",
      "oid": "1.3.6.1.2.1.1.5.0",
      "description": "System name"
    },
    {
      "name": "sysUpTime",
      "oid": "1.3.6.1.2.1.1.3.0",
      "description": "System uptime"
    }
  ]
}
```

## Build

```shell
go build -o snmp_poll snmp_poll.go
```

## Dependencies

- github.com/gosnmp/gosnmp - For SNMP operations
- github.com/sleepinggenius2/gosmi - For MIB parsing
