package interface_status_parser

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// StandardizedEntry represents the standardized JSON structure
type StandardizedEntry struct {
	DataType  string              `json:"data_type"`  // Always "cisco_nexus_interface_status"
	Timestamp string              `json:"timestamp"`  // ISO 8601 timestamp
	Date      string              `json:"date"`       // Date in YYYY-MM-DD format
	Message   InterfaceStatusData `json:"message"`    // Interface status-specific data
}

// InterfaceStatusData represents the interface status data within the message field
type InterfaceStatusData struct {
	Interfaces []InterfaceEntry `json:"interfaces"`
}

// InterfaceEntry represents a single interface entry
type InterfaceEntry struct {
	Port   string `json:"port"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Vlan   string `json:"vlan"`
	Duplex string `json:"duplex"`
	Speed  string `json:"speed"`
	Type   string `json:"type"`
}

// parseInterfaceStatus parses the interface status output
func parseInterfaceStatus(input string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))

	// Get current timestamp
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02")

	data := InterfaceStatusData{
		Interfaces: make([]InterfaceEntry, 0),
	}

	// Regular expression to match separator lines
	separatorPattern := regexp.MustCompile(`^-{10,}$`)

	// Regular expression to match header lines
	headerPattern := regexp.MustCompile(`^\s*Port\s+Name\s+Status\s+Vlan\s+Duplex\s+Speed\s+Type\s*$`)

	// Track lines for potential continuation handling
	var pendingEntry *InterfaceEntry

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Skip separator lines
		if separatorPattern.MatchString(strings.TrimSpace(line)) {
			continue
		}

		// Skip header lines
		if headerPattern.MatchString(line) {
			continue
		}

		// Skip lines that are just the switch prompt
		if strings.Contains(line, "#") && !strings.HasPrefix(strings.TrimSpace(line), "Eth") &&
			!strings.HasPrefix(strings.TrimSpace(line), "mgmt") &&
			!strings.HasPrefix(strings.TrimSpace(line), "Po") &&
			!strings.HasPrefix(strings.TrimSpace(line), "Lo") &&
			!strings.HasPrefix(strings.TrimSpace(line), "Vlan") {
			continue
		}

		// Try to parse as an interface line
		entry, isContinuation := parseLine(line)

		if isContinuation && pendingEntry != nil {
			// This is a continuation line (just the type suffix)
			pendingEntry.Type = pendingEntry.Type + strings.TrimSpace(line)
			data.Interfaces = append(data.Interfaces, *pendingEntry)
			pendingEntry = nil
			continue
		}

		if entry != nil {
			// Check if this entry might have a continuation
			if isIncompleteLine(line) {
				pendingEntry = entry
			} else {
				data.Interfaces = append(data.Interfaces, *entry)
				pendingEntry = nil
			}
		}
	}

	// Add any remaining pending entry
	if pendingEntry != nil {
		data.Interfaces = append(data.Interfaces, *pendingEntry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	entry := StandardizedEntry{
		DataType:  "cisco_nexus_interface_status",
		Timestamp: timestamp,
		Date:      date,
		Message:   data,
	}

	return []StandardizedEntry{entry}, nil
}

// parseLine parses a single line of interface status output
// Returns the entry and a boolean indicating if this is just a continuation line
func parseLine(line string) (*InterfaceEntry, bool) {
	trimmed := strings.TrimSpace(line)

	// Check if this is a continuation line (just type suffix like "CC")
	if len(trimmed) <= 4 && !strings.HasPrefix(trimmed, "Eth") &&
		!strings.HasPrefix(trimmed, "mgmt") &&
		!strings.HasPrefix(trimmed, "Po") &&
		!strings.HasPrefix(trimmed, "Lo") &&
		!strings.HasPrefix(trimmed, "Vlan") {
		return nil, true
	}

	// Parse the interface line using fixed-width column positions
	// The output format is relatively consistent with these approximate positions
	entry := parseFixedWidthLine(line)
	if entry != nil {
		return entry, false
	}

	return nil, false
}

// parseFixedWidthLine parses a line using field splitting
func parseFixedWidthLine(line string) *InterfaceEntry {
	// Split the line into fields
	fields := strings.Fields(line)

	if len(fields) < 4 {
		return nil
	}

	entry := &InterfaceEntry{}

	// First field is always the port
	entry.Port = fields[0]

	// Validate that this looks like a valid port
	if !isValidPort(entry.Port) {
		return nil
	}

	// Determine the structure based on status field position
	// Status can be: connected, notconnec, disabled, down, routed
	statusIndex := findStatusIndex(fields)

	if statusIndex == -1 {
		return nil
	}

	// Name is everything between port and status
	if statusIndex > 1 {
		entry.Name = strings.Join(fields[1:statusIndex], " ")
	} else {
		entry.Name = "--"
	}

	entry.Status = fields[statusIndex]

	// Parse remaining fields after status
	remaining := fields[statusIndex+1:]

	if len(remaining) >= 1 {
		entry.Vlan = remaining[0]
	}
	if len(remaining) >= 2 {
		entry.Duplex = remaining[1]
	}
	if len(remaining) >= 3 {
		entry.Speed = remaining[2]
	}
	if len(remaining) >= 4 {
		entry.Type = strings.Join(remaining[3:], "")
	}

	return entry
}

// isValidPort checks if a string looks like a valid port identifier
func isValidPort(port string) bool {
	validPrefixes := []string{"Eth", "mgmt", "Po", "Lo", "Vlan"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(port, prefix) {
			return true
		}
	}
	return false
}

// findStatusIndex finds the index of the status field in the fields slice
func findStatusIndex(fields []string) int {
	validStatuses := []string{"connected", "notconnec", "disabled", "down", "routed", "sfpAbsent", "xcvrAbsen", "noOperMem", "channelDo"}
	for i, field := range fields {
		for _, status := range validStatuses {
			if field == status {
				return i
			}
		}
	}
	return -1
}

// isIncompleteLine checks if a line appears to be incomplete (type field is cut off)
func isIncompleteLine(line string) bool {
	// Check if line ends with a hyphen followed by incomplete type
	// Common patterns: QSFP-100G-P (missing CC), QSFP-40G-CSR (missing 4)
	trimmed := strings.TrimSpace(line)
	incompletePatterns := []string{"-P", "-SR", "-LR", "-CR", "-AOC", "-PSM", "-CSR"}
	for _, pattern := range incompletePatterns {
		if strings.HasSuffix(trimmed, pattern) {
			return true
		}
	}
	return false
}

// runVsh runs the given command using the vsh CLI and returns its output as a string
func runVsh(command string) (string, error) {
	cmd := []string{"vsh", "-c", command}
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("vsh error: %v, output: %s", err, string(out))
	}
	return string(out), nil
}

func main() {
	// Define command line flags
	inputFile := flag.String("input", "", "Input file containing Cisco Nexus interface status output")
	outputFile := flag.String("output", "", "Output file to write JSON results (default: stdout)")
	commandsFile := flag.String("commands", "", "Path to JSON file containing CLI commands")
	flag.Parse()

	if (*inputFile != "" && *commandsFile != "") || (*inputFile == "" && *commandsFile == "") {
		fmt.Fprintln(os.Stderr, "Error: You must specify exactly one of -input or -commands.")
		os.Exit(1)
	}

	var inputData string

	if *commandsFile != "" {
		// Read commands JSON file
		data, err := os.ReadFile(*commandsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading commands file: %v\n", err)
			os.Exit(1)
		}
		var cmdFile struct {
			Commands []struct {
				Name    string `json:"name"`
				Command string `json:"command"`
			} `json:"commands"`
		}
		if err := json.Unmarshal(data, &cmdFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing commands JSON: %v\n", err)
			os.Exit(1)
		}
		var intStatusCmd string
		for _, c := range cmdFile.Commands {
			if c.Name == "interface-status" {
				intStatusCmd = c.Command
				break
			}
		}
		if intStatusCmd == "" {
			fmt.Fprintln(os.Stderr, "Error: No 'interface-status' command found in commands JSON.")
			os.Exit(1)
		}
		// Run the command using vsh
		vshOut, err := runVsh(intStatusCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running vsh: %v\n", err)
			os.Exit(1)
		}
		inputData = vshOut
	} else if *inputFile != "" {
		// Read from file
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
		inputData = string(data)
	}

	// Parse the interface status
	entries, err := parseInterfaceStatus(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing interface status: %v\n", err)
		os.Exit(1)
	}

	// Output the results
	var output *os.File
	if *outputFile == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer output.Close()
	}

	// Write each entry as a separate JSON object, one per line (JSON Lines format)
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "")
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding entry: %v\n", err)
			os.Exit(1)
		}
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

// GetDescription returns the parser description
func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show interface status' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseInterfaceStatus(content)
}
