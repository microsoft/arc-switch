# Copilot Instructions for arc-switch Go Projects

This file provides guidelines for GitHub Copilot and other AI assistants when helping with this repository.

## Golang Best Practices

### Code Style and Organization

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) for style guidance
- Respect Go's [Effective Go](https://golang.org/doc/effective_go) principles
- Use `gofmt` or `goimports` for consistent formatting
- Organize imports in three groups: standard library, external packages, internal packages
- Follow package structure: `package main` for executables, descriptive names for libraries
- Place interfaces near where they're used, not in separate files

### Error Handling

- Return errors explicitly; avoid panic except for unrecoverable situations
- Handle errors immediately; don't defer checks
- Use custom error types for specific error conditions
- Use `errors.Is()` and `errors.As()` for error checking when appropriate
- Prefer `fmt.Errorf("...%w", err)` for wrapping errors

### Variable Naming

- Use short, concise variable names in small scopes
- Use descriptive names for exported functions, types, and variables
- Use camelCase for private and PascalCase for exported identifiers
- Prefer single-letter variables for indices and short loops

### Testing

- Write table-driven tests using `testing.T`
- Use `t.Parallel()` for tests that can run in parallel
- Test both expected successes and failures
- Avoid global variables in tests
- Name test files as `*_test.go`

### Performance

- Use buffered I/O for file operations
- Preallocate slices when size is known
- Use efficient data structures (maps for lookups, slices for iteration)
- Consider sync.Pool for frequently allocated objects
- Profile before optimizing

### Concurrency

- Use channels to communicate between goroutines, not shared memory
- Use context for cancellation and timing
- Be careful with goroutine leaks
- Use sync.Mutex for simple state protection
- Consider sync.WaitGroup for waiting on multiple goroutines

### SNMP Project Specifics

- Log errors with context (device info, OIDs)
- Handle SNMP timeouts gracefully
- Use JSON struct tags consistently
- Document all configuration options
- Include retry logic for transient failures
- Validate input configurations
- Follow separation of concerns between:
  - SNMP operations
  - MIB handling
  - Configuration management
  - Output formatting

### JSON Output Format

All tools must output data in JSON format with standardized fields for consistency and interoperability.

#### Standardized JSON Structure

Every parser tool must use this exact structure for all output records:

```json
{
  "data_type": "",
  "timestamp": "",
  "date": "",
  "message": {
    // Parser-specific JSON data goes here
  }
}
```

#### Required Fields

Every tool output must include these four required fields:

- `data_type`: String identifying the type of data (e.g., "interface_counters", "cisco_nexus_mac_table", "lldp_neighbors")
- `timestamp`: Full timestamp in ISO 8601 format (e.g., "2024-01-15T10:30:00Z") for machine parsing
- `date`: Date in ISO format (YYYY-MM-DD) for easy filtering and sorting
- `message`: JSON object containing all parser-specific data fields

#### Message Field Structure

All parser-specific data must be contained within the `message` field as a JSON object. This provides:
- Consistent top-level structure across all parsers
- Clear separation between metadata and data
- Compatibility with the syslogwriter library
- Easier processing and validation

#### Example: Cisco Nexus MAC Table Parser

```json
{
  "data_type": "cisco_nexus_mac_table",
  "timestamp": "2024-06-23T17:05:01Z",
  "date": "2024-06-23",
  "message": {
    "primary_entry": true,
    "gateway_mac": false,
    "routed_mac": false,
    "overlay_mac": false,
    "vlan": "7",
    "mac_address": "02ec.a004.0000",
    "type": "dynamic",
    "age": "NA",
    "secure": "F",
    "ntfy": "F",
    "port": "Eth1/1",
    "vpc_peer_link": false
  }
}
```

**Message Fields for MAC Table Parser:**

- `primary_entry`: Boolean indicating if this is a primary entry
- `gateway_mac`: Boolean indicating if this is a gateway MAC
- `routed_mac`: Boolean indicating if this is a routed MAC
- `overlay_mac`: Boolean indicating if this is an overlay MAC
- `vlan`: VLAN ID as string
- `mac_address`: MAC address in Cisco format (xxxx.xxxx.xxxx)
- `type`: Entry type (e.g., "dynamic", "static")
- `age`: Age information or "NA"
- `secure`: Security flag ("F" or "T")
- `ntfy`: Notification flag ("F" or "T")
- `port`: Port identifier
- `vpc_peer_link`: Boolean, only present when true for vPC peer-link entries

#### Example: Interface Counters Parser

```json
{
  "data_type": "interface_counters",
  "timestamp": "2024-01-15T10:30:00Z",
  "date": "2024-01-15",
  "message": {
    "interface_name": "Eth1/1",
    "interface_type": "ethernet",
    "in_octets": 205027653248,
    "in_ucast_pkts": 650373664,
    "in_mcast_pkts": 2262324,
    "in_bcast_pkts": 68097,
    "out_octets": 3195383643785,
    "out_ucast_pkts": 2314463086,
    "out_mcast_pkts": 365931965,
    "out_bcast_pkts": 53571839,
    "has_ingress_data": true,
    "has_egress_data": true
  }
}
```

**Message Fields for Interface Counters Parser:**

- `interface_name`: Interface identifier (e.g., "Eth1/1", "Po50", "Vlan1")
- `interface_type`: Type category ("ethernet", "port-channel", "vlan", "management", "tunnel")
- `in_octets`: Ingress octets counter (-1 if unavailable)
- `in_ucast_pkts`: Ingress unicast packets counter (-1 if unavailable)
- `in_mcast_pkts`: Ingress multicast packets counter (-1 if unavailable)
- `in_bcast_pkts`: Ingress broadcast packets counter (-1 if unavailable)
- `out_octets`: Egress octets counter (-1 if unavailable)
- `out_ucast_pkts`: Egress unicast packets counter (-1 if unavailable)
- `out_mcast_pkts`: Egress multicast packets counter (-1 if unavailable)
- `out_bcast_pkts`: Egress broadcast packets counter (-1 if unavailable)
- `has_ingress_data`: Boolean indicating if ingress counters are available
- `has_egress_data`: Boolean indicating if egress counters are available

#### Integration with SyslogWriter

All parsers should be designed to work with the standardized syslogwriter library located in `src/SyslogTools/syslogwriter/`. This library expects the exact JSON structure defined above:

```go
import "github.com/arc-switch/syslogwriter"

// Initialize syslog writer
writer, err := syslogwriter.NewWithDefaults("parser-name")
if err != nil {
    log.Fatal(err)
}
defer writer.Close()

// Create standardized JSON structure
entry := map[string]interface{}{
    "data_type": "your_parser_type",
    "timestamp": time.Now().Format(time.RFC3339),
    "date": time.Now().Format("2006-01-02"),
    "message": map[string]interface{}{
        // Your parser-specific fields here
        "field1": "value1",
        "field2": 12345,
    },
}

// Convert to JSON and write to syslog
jsonData, err := json.Marshal(entry)
if err != nil {
    log.Printf("Failed to marshal entry: %v", err)
    return
}

if err := writer.WriteEntry(string(jsonData)); err != nil {
    log.Printf("Failed to write to syslog: %v", err)
}
```

#### Validation Requirements

When creating new parsers, ensure:

1. **Structure Compliance**: Always use the four required fields (data_type, timestamp, date, message)
2. **Field Validation**: Validate that all required fields are present and non-empty
3. **Message Content**: All parser-specific data goes in the message field only
4. **JSON Validity**: Ensure output is valid JSON and can be parsed
5. **Syslog Compatibility**: Test integration with the syslogwriter library

#### Parser Development Guidelines

- Use consistent data_type naming: `vendor_device_data_type` (e.g., "cisco_nexus_mac_table")
- Include comprehensive error handling for malformed input
- Provide verbose output options for debugging
- Document all message field structures in the parser's README
- Include sample input and output files for testing
- Write unit tests that validate JSON structure compliance
- **Test parser output with the validation script**: Use `./validate-parser-output.sh` to verify compliance with the standardized JSON structure

#### Output Validation

The project includes a validation script at the root level (`validate-parser-output.sh`) that verifies parser outputs conform to the standardized JSON structure. This script supports multiple JSON formats:

- Single JSON object (compact or pretty-printed)
- JSON array containing multiple objects  
- Concatenated JSON objects (newline-separated)
- Live parser output via stdin

**Usage examples:**
```bash
# Validate a parser output file
./validate-parser-output.sh parser-output.json

# Validate live parser output
./parser -input data.txt | ./validate-parser-output.sh

# Test your parser's compliance
cd src/SwitchOutput/Cisco/Nexus/10/ip_arp_parser
./ip_arp_parser -input ../show-ip-arp.txt | /workspaces/arc-switch2/validate-parser-output.sh
```

**Always run the validation script on your parser output before committing code.**

### Project Structure

- Keep `main.go` small and focused on setup and coordination
- Use separate packages for specialized functionality
- Place most logic in importable packages for testing
- Use appropriate directory structure:
  - `/cmd` - Main applications
  - `/pkg` - Library code
  - `/internal` - Private application code
  - `/api` - API definitions
  - `/configs` - Configuration file templates

### Dependencies

- Minimize external dependencies
- Use Go modules for dependency management
- Vendor dependencies for reproducible builds
- Use specific versions in go.mod, not master branch

## Markdown Best Practices

Follow these guidelines to ensure all markdown files comply with linting rules and GitHub best practices.

### Document Structure

- Start every markdown file with a single H1 header (`#`)
- Use proper heading hierarchy (H1 → H2 → H3, don't skip levels)
- Include a blank line after each heading
- End files with a single blank line
- Use consistent heading case (prefer title case for main sections)

### Code Blocks and Syntax Highlighting

- Always specify language for fenced code blocks:
  ```bash
  # Good
  ./command --help
  ```
  
  ```text
  # For plain text output
  Interface Status: UP
  ```

- Use backticks for inline code: `variable_name`, `function()`
- Use triple backticks for multi-line code blocks
- Specify appropriate language identifiers: `bash`, `go`, `json`, `yaml`, `text`, `markdown`

### Lists and Formatting

- Use `-` (hyphen) for unordered lists consistently
- Use `1.` for ordered lists (let markdown auto-number)
- Add blank lines around lists for proper spacing
- Indent nested list items with 2 spaces
- Use **bold** for emphasis, *italic* for subtle emphasis
- Use `inline code` for technical terms, commands, and filenames

### Links and References

- Use descriptive link text, avoid "click here" or bare URLs
- Prefer reference-style links for readability in long documents:
  ```markdown
  See the [Go documentation][go-docs] for more details.
  
  [go-docs]: https://golang.org/doc/
  ```
- Use relative paths for internal repository links
- Ensure all links are valid and accessible

### Tables

- Use proper table formatting with aligned columns
- Include header separators with at least 3 dashes
- Align text content left, numbers right:
  ```markdown
  | Tool Name        | Type     | Status |
  |------------------|----------|--------|
  | mac_parser       | CLI      | Active |
  | interface_parser | CLI      | Active |
  ```

### Line Length and Wrapping

- Aim for 80-120 character line length for readability
- Break long lines at natural points (after punctuation, before conjunctions)
- Don't break lines in the middle of inline code or links
- Use soft line breaks (don't force hard wraps unless necessary)

### File Organization

- Use descriptive filenames with hyphens: `installation-guide.md`
- Include a brief description at the top of each document
- Organize content with clear sections and subsections
- Use table of contents for longer documents (>10 sections)

### Common Markdown Linting Rules (markdownlint)

Ensure compliance with these key rules:

- **MD001**: Heading levels should only increment by one level at a time
- **MD003**: Use consistent heading style (ATX-style with `#`)
- **MD004**: Use consistent unordered list style (hyphens `-`)
- **MD007**: Unordered list indentation should be 2 spaces
- **MD009**: No trailing spaces at end of lines
- **MD010**: No hard tabs, use spaces only
- **MD012**: No multiple consecutive blank lines
- **MD013**: Line length should be reasonable (80-120 chars)
- **MD018**: No space after `#` in headings
- **MD022**: Headings should be surrounded by blank lines
- **MD025**: Only one H1 heading per document
- **MD026**: No trailing punctuation in headings
- **MD030**: Spaces after list markers (1 space for unordered, 2 for ordered)
- **MD032**: Lists should be surrounded by blank lines
- **MD040**: Fenced code blocks should have a language specified
- **MD041**: First line in file should be a top-level heading
- **MD047**: Files should end with a single newline character

### GitHub-Specific Considerations

- Use GitHub Flavored Markdown (GFM) features appropriately
- Test rendering in GitHub's preview before committing
- Use task lists for TODO items: `- [ ] Task name`
- Use GitHub's alert syntax for important information:
  ```markdown
  > [!NOTE]
  > This is a note
  
  > [!WARNING]
  > This is a warning
  ```
- Include shields/badges consistently at the top of README files
- Use proper emoji codes when appropriate: `:rocket:` → 🚀

### Documentation Standards

- Write in clear, concise language
- Use active voice when possible
- Include examples for all instructions
- Document prerequisites and assumptions
- Provide troubleshooting sections for complex procedures
- Keep documentation up-to-date with code changes
- Use consistent terminology throughout the project

### README File Requirements

Every tool/project should include a README.md with:

1. **Title and Description**: Clear project name and purpose
2. **Installation Instructions**: How to build/install the tool
3. **Usage Examples**: Command-line examples with expected output
4. **Configuration**: Available options and settings
5. **Input/Output Formats**: Expected data formats and schemas
6. **Examples**: Sample input/output files
7. **Contributing**: How to contribute to the project
8. **License**: License information or reference

### Validation and Tools

- Use markdownlint or similar tools to validate markdown files
- Set up pre-commit hooks to check markdown formatting
- Configure editor extensions for real-time markdown linting
- Test documentation rendering in GitHub preview mode
- Validate all external links regularly
