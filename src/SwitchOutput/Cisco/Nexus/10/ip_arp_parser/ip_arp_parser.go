package ip_arp_parser

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

// StandardEntry represents the standardized JSON structure for syslog compatibility
type StandardEntry struct {
	DataType  string      `json:"data_type"`  // Identifies the type of data
	Timestamp string      `json:"timestamp"`  // Timestamp when the data was processed
	Date      string      `json:"date"`       // Date when the data was processed
	Message   ARPTableEntry `json:"message"`  // ARP-specific data
}

// ARPTableEntry represents a single entry in the Cisco Nexus IP ARP table
type ARPTableEntry struct {
	IPAddress            string `json:"ip_address"`                       // IP address
	Age                  string `json:"age"`                              // Age (format: HH:MM:SS or decimal seconds)
	MACAddress           string `json:"mac_address"`                      // MAC address
	Interface            string `json:"interface"`                        // Interface name (Vlan, Ethernet, port-channel)
	
	// Flag indicators based on Cisco documentation
	NonActiveFHRP        bool   `json:"non_active_fhrp,omitempty"`        // * - Adjacencies learnt on non-active FHRP router
	CFSoESync            bool   `json:"cfsoe_sync,omitempty"`             // + - Adjacencies synced via CFSoE
	ThrottledGlean       bool   `json:"throttled_glean,omitempty"`        // # - Adjacencies Throttled for Glean
	ControlPlaneL2RIB    bool   `json:"control_plane_l2rib,omitempty"`    // CP - Added via L2RIB, Control plane Adjacencies
	PeerSyncL2RIB        bool   `json:"peer_sync_l2rib,omitempty"`        // PS - Added via L2RIB, Peer Sync
	ReOriginatedPeerSync bool   `json:"re_originated_peer_sync,omitempty"`// RO - Re-Originated Peer Sync Entry
	StaticDownInterface  bool   `json:"static_down_interface,omitempty"`  // D - Static Adjacencies attached to down interface
	
	// Additional metadata
	InterfaceType        string `json:"interface_type"`                   // Type of interface (vlan, ethernet, port-channel)
	FlagsRaw             string `json:"flags_raw,omitempty"`              // Raw flags field for debugging
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseARP parses the IP ARP table output and emits each entry as JSON
func parseARP(input string) ([]StandardEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	
	// Get current timestamp
	now := time.Now()
	// ISO 8601 format for timestamp
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02") // ISO format
	
	var entries []StandardEntry
	
	// Read until we find the table headers
	foundHeader := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Skip the command line and switch prompt
		if strings.Contains(line, "show ip arp") || strings.HasSuffix(line, "#") {
			continue
		}
		
		// Skip flags explanation lines
		if strings.HasPrefix(line, "Flags:") || strings.HasPrefix(line, "       ") {
			continue
		}
		
		// Skip context and summary lines
		if strings.Contains(line, "IP ARP Table for context") || 
		   strings.Contains(line, "Total number of entries:") {
			continue
		}
		
		// Look for the table header
		if strings.Contains(line, "Address") && strings.Contains(line, "Age") && 
		   strings.Contains(line, "MAC Address") && strings.Contains(line, "Interface") {
			foundHeader = true
			continue
		}
		
		// Parse data lines after header is found
		if foundHeader {
			entry := parseARPLine(line, timestamp, date)
			if entry != nil {
				standardEntry := StandardEntry{
					DataType:  "cisco_nexus_arp_entry",
					Timestamp: timestamp,
					Date:      date,
					Message:   *entry,
				}
				entries = append(entries, standardEntry)
			}
		}
	}
	
	return entries, nil
}

// parseARPLine parses a single line from the ARP table
func parseARPLine(line, timestamp, date string) *ARPTableEntry {
	// Skip empty lines
	if strings.TrimSpace(line) == "" {
		return nil
	}
	
	// Regex to parse ARP table entries
	// Format: IP_ADDRESS    AGE       MAC_ADDRESS     INTERFACE       FLAGS
	re := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)\s+([0-9.:]+)\s+([0-9a-f]{4}\.[0-9a-f]{4}\.[0-9a-f]{4})\s+(\S+)\s*(.*)$`)
	matches := re.FindStringSubmatch(line)
	
	if len(matches) < 5 {
		return nil
	}
	
	entry := &ARPTableEntry{
		IPAddress:   matches[1],
		Age:         matches[2],
		MACAddress:  matches[3],
		Interface:   matches[4],
		FlagsRaw:    strings.TrimSpace(matches[5]),
	}
	
	// Determine interface type
	entry.InterfaceType = determineInterfaceType(entry.Interface)
	
	// Parse flags
	parseFlags(entry, entry.FlagsRaw)
	
	return entry
}

// determineInterfaceType determines the interface type based on the interface name
func determineInterfaceType(interfaceName string) string {
	interfaceName = strings.ToLower(interfaceName)
	
	if strings.HasPrefix(interfaceName, "vlan") {
		return "vlan"
	} else if strings.HasPrefix(interfaceName, "ethernet") || strings.HasPrefix(interfaceName, "eth") {
		return "ethernet"
	} else if strings.HasPrefix(interfaceName, "port-channel") || strings.HasPrefix(interfaceName, "po") {
		return "port-channel"
	} else if strings.HasPrefix(interfaceName, "mgmt") {
		return "management"
	} else if strings.HasPrefix(interfaceName, "tunnel") {
		return "tunnel"
	} else if strings.HasPrefix(interfaceName, "loopback") {
		return "loopback"
	}
	
	return "other"
}

// parseFlags parses the flags field and sets the appropriate boolean fields
func parseFlags(entry *ARPTableEntry, flags string) {
	if strings.Contains(flags, "*") {
		entry.NonActiveFHRP = true
	}
	if strings.Contains(flags, "+") {
		entry.CFSoESync = true
	}
	if strings.Contains(flags, "#") {
		entry.ThrottledGlean = true
	}
	if strings.Contains(flags, "CP") {
		entry.ControlPlaneL2RIB = true
	}
	if strings.Contains(flags, "PS") {
		entry.PeerSyncL2RIB = true
	}
	if strings.Contains(flags, "RO") {
		entry.ReOriginatedPeerSync = true
	}
	if strings.Contains(flags, "D") {
		entry.StaticDownInterface = true
	}
}

// loadCommandsFromFile loads the commands configuration from a JSON file
func loadCommandsFromFile(filename string) (*CommandConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading commands file: %v", err)
	}

	var config CommandConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing commands file: %v", err)
	}

	return &config, nil
}

// findArpCommand finds the arp-table command in the commands.json
func findArpCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "arp-table" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("arp-table command not found in commands file")
}

// runCommand executes the given command using vsh and returns its output
func runCommand(command string) (string, error) {
	cmd := exec.Command("vsh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %v, output: %s", err, string(output))
	}
	return string(output), nil
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show ip arp' output")
	var outputFile = flag.String("output", "", "Output file for JSON results (default: stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Cisco Nexus IP ARP Parser")
		fmt.Println("Parses 'show ip arp' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  ip_arp_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show ip arp' output")
		fmt.Println("  -output <file>    Output file for JSON results (default: stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./ip_arp_parser -input show-ip-arp.txt -output arp-results.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./ip_arp_parser -commands ../commands.json -output arp-results.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./ip_arp_parser -input show-ip-arp.txt")
		return
	}

	var content string
	var err error

	// Determine input source
	if *inputFile != "" {
		// Read from input file
		file, err := os.Open(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}

		content = strings.Join(lines, "\n")
	} else if *commandsFile != "" {
		// Get data from switch using commands file
		fmt.Fprintf(os.Stderr, "Loading commands from file: %s\n", *commandsFile)
		
		config, err := loadCommandsFromFile(*commandsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading commands file: %v\n", err)
			os.Exit(1)
		}

		command, err := findArpCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding ARP command: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Executing command: %s\n", command)
		content, err = runCommand(command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: Either -input or -commands parameter is required\n")
		fmt.Fprintf(os.Stderr, "Use -help for usage information\n")
		os.Exit(1)
	}

	// Parse the ARP table data
	fmt.Fprintf(os.Stderr, "Parsing ARP table data...\n")
	entries, err := parseARP(content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing ARP table: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Found %d ARP entries\n", len(entries))

	// Output results
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
		fmt.Fprintf(os.Stderr, "ARP data written to %s\n", *outputFile)
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
	return "Parses 'show ip arp' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseARP(content)
}
