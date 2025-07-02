package main

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
	DataType  string        `json:"data_type"`  // Always "cisco_nexus_mac_table"
	Timestamp string        `json:"timestamp"`  // ISO 8601 timestamp
	Date      string        `json:"date"`       // Date in YYYY-MM-DD format
	Message   MacTableData  `json:"message"`    // MAC table-specific data
}

// MacTableData represents the MAC table data within the message field
type MacTableData struct {
	PrimaryEntry bool   `json:"primary_entry"`              // * indicates primary entry
	GatewayMAC   bool   `json:"gateway_mac"`                // G indicates Gateway MAC
	RoutedMAC    bool   `json:"routed_mac"`                 // (R) indicates Routed MAC
	OverlayMAC   bool   `json:"overlay_mac"`                // O indicates Overlay MAC
	VLAN         string `json:"vlan"`                       // VLAN ID (can be - for some entries)
	MACAddress   string `json:"mac_address"`                // MAC address
	Type         string `json:"type"`                       // Type of entry (dynamic, static, etc.)
	Age          string `json:"age"`                        // Age (seconds since last seen, - for static, NA for some entries)
	Secure       string `json:"secure"`                     // Secure flag (T/F)
	NTFY         string `json:"ntfy"`                       // NTFY flag (T/F)
	Port         string `json:"port"`                       // Port identifier
	VPCPeerLink  bool   `json:"vpc_peer_link,omitempty"`    // + indicates primary entry using vPC Peer-Link
	TrueFlag     bool   `json:"true_flag,omitempty"`        // (T) indicates True
	FalseFlag    bool   `json:"false_flag,omitempty"`       // (F) indicates False
	ControlPlane bool   `json:"control_plane_mac,omitempty"`// C indicates ControlPlane MAC
	VSAN         bool   `json:"vsan,omitempty"`             // ~ indicates vsan
}

// parseMAC parses the MAC address table output and emits each entry as JSON
func parseMAC(input string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	// Get current timestamp
	now := time.Now()
	// ISO 8601 format
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02") // YYYY-MM-DD format
	
	// Read until we find the table headers
	foundHeader := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "VLAN     MAC Address") {
			foundHeader = true
			// Skip the separator line
			scanner.Scan()
			break
		}
	}
	
	if !foundHeader {
		return nil, fmt.Errorf("could not find MAC address table header")
	}

	// MAC table entry line pattern
	entryPattern := regexp.MustCompile(`^([*+GCO]?)?\s*([^\s]+)\s+([^\s]+)\s+([^\s]+)\s+([^\s]+)\s+([^\s]+)\s+([^\s]+)\s+(.+)$`)
	
	var entries []StandardizedEntry
	
	// Parse each line of the table
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		matches := entryPattern.FindStringSubmatch(line)
		if matches == nil {
			continue // Skip lines that don't match the pattern
		}
		
		// Extract fields from the matched line
		prefix := strings.TrimSpace(matches[1])
		vlan := strings.TrimSpace(matches[2])
		macAddress := strings.TrimSpace(matches[3])
		entryType := strings.TrimSpace(matches[4])
		age := strings.TrimSpace(matches[5])
		secure := strings.TrimSpace(matches[6])
		ntfy := strings.TrimSpace(matches[7])
		port := strings.TrimSpace(matches[8])
				// Check for special flags in the port field
		routedMAC := strings.Contains(port, "(R)")
		port = strings.TrimSuffix(strings.TrimSuffix(port, "(R)"), " ")
		
		entry := StandardizedEntry{
			DataType:  "cisco_nexus_mac_table",
			Timestamp: timestamp,
			Date:      date,
			Message: MacTableData{
				PrimaryEntry: strings.Contains(prefix, "*"),
				GatewayMAC:   strings.Contains(prefix, "G"),
				RoutedMAC:    routedMAC,
				OverlayMAC:   strings.Contains(prefix, "O"),
				VLAN:         vlan,
				MACAddress:   macAddress,
				Type:         entryType,
				Age:          age,
				Secure:       secure,
				NTFY:         ntfy,
				Port:         port,
				VPCPeerLink:  strings.Contains(prefix, "+"),
				ControlPlane: strings.Contains(prefix, "C"),
				VSAN:         strings.Contains(prefix, "~"),
				TrueFlag:     false, // Will be set if specifically indicated
				FalseFlag:    false, // Will be set if specifically indicated
			},
		}
		
		entries = append(entries, entry)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	return entries, nil
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
	inputFile := flag.String("input", "", "Input file containing Cisco Nexus MAC address table output")
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
		var macCmd string
		for _, c := range cmdFile.Commands {
			if c.Name == "mac-address-table" {
				macCmd = c.Command
				break
			}
		}
		if macCmd == "" {
			fmt.Fprintln(os.Stderr, "Error: No 'mac-address-table' command found in commands JSON.")
			os.Exit(1)
		}
		// Run the command using vsh
		vshOut, err := runVsh(macCmd)
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
	
	// Parse the MAC address table
	entries, err := parseMAC(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing MAC address table: %v\n", err)
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
