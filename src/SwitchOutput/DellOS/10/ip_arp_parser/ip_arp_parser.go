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

// StandardizedEntry represents the standardized JSON structure for syslog compatibility
type StandardizedEntry struct {
	DataType  string        `json:"data_type"`
	Timestamp string        `json:"timestamp"`
	Date      string        `json:"date"`
	Message   ARPTableEntry `json:"message"`
}

// ARPTableEntry represents a single entry in the Dell OS10 IP ARP table
type ARPTableEntry struct {
	IPAddress       string `json:"ip_address"`
	HardwareAddress string `json:"hardware_address"`
	Interface       string `json:"interface"`
	EgressInterface string `json:"egress_interface,omitempty"`
	InterfaceType   string `json:"interface_type"`
	PrivateVLAN     string `json:"private_vlan,omitempty"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// determineInterfaceType determines the interface type based on the interface name
func determineInterfaceType(interfaceName string) string {
	lowerName := strings.ToLower(interfaceName)

	if strings.HasPrefix(lowerName, "vlan") {
		return "vlan"
	} else if strings.HasPrefix(lowerName, "ethernet") {
		return "ethernet"
	} else if strings.HasPrefix(lowerName, "port-channel") {
		return "port-channel"
	} else if strings.HasPrefix(lowerName, "mgmt") || strings.HasPrefix(lowerName, "management") {
		return "management"
	} else if strings.HasPrefix(lowerName, "loopback") {
		return "loopback"
	} else if strings.HasPrefix(lowerName, "virtual-network") {
		return "virtual-network"
	}

	return "other"
}

// parseARP parses the IP ARP table output and emits each entry as JSON
func parseARP(input string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))

	// Get current timestamp
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02")

	var entries []StandardizedEntry

	// Find the header line
	foundHeader := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Look for the table header (Dell OS10 format)
		// "Address          Hardware address      Interface          Egress Interface"
		if strings.Contains(line, "Address") && strings.Contains(line, "Hardware address") {
			foundHeader = true
			// Skip separator line
			scanner.Scan()
			break
		}
	}

	if !foundHeader {
		return nil, fmt.Errorf("could not find ARP table header")
	}

	// Parse data lines
	// Dell OS10 ARP format:
	// Address          Hardware address      Interface          Egress Interface
	// 192.168.2.2      90:b1:1c:f4:a6:e6    ethernet1/1/49:1   ethernet1/1/49:1
	// Can also have "pv <vlan-id>" for private VLAN
	arpLineRegex := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)\s+([0-9a-f:]+)\s+(\S+)(?:\s+(\S+))?(?:\s+pv\s+(\d+))?$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and separator lines
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}

		// Skip summary lines
		if strings.Contains(line, "Total Entries") || strings.Contains(line, "Static Entries") {
			continue
		}

		matches := arpLineRegex.FindStringSubmatch(line)
		if matches == nil {
			// Try simpler pattern without egress interface
			simpleRegex := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)\s+([0-9a-f:]+)\s+(\S+)`)
			simpleMatches := simpleRegex.FindStringSubmatch(line)
			if simpleMatches != nil {
				entry := StandardizedEntry{
					DataType:  "dell_os10_arp_entry",
					Timestamp: timestamp,
					Date:      date,
					Message: ARPTableEntry{
						IPAddress:       simpleMatches[1],
						HardwareAddress: simpleMatches[2],
						Interface:       simpleMatches[3],
						InterfaceType:   determineInterfaceType(simpleMatches[3]),
					},
				}
				entries = append(entries, entry)
			}
			continue
		}

		entry := StandardizedEntry{
			DataType:  "dell_os10_arp_entry",
			Timestamp: timestamp,
			Date:      date,
			Message: ARPTableEntry{
				IPAddress:       matches[1],
				HardwareAddress: matches[2],
				Interface:       matches[3],
				InterfaceType:   determineInterfaceType(matches[3]),
			},
		}

		// Egress interface (optional)
		if len(matches) > 4 && matches[4] != "" && !strings.HasPrefix(matches[4], "pv") {
			entry.Message.EgressInterface = matches[4]
		}

		// Private VLAN (optional)
		if len(matches) > 5 && matches[5] != "" {
			entry.Message.PrivateVLAN = matches[5]
		}

		entries = append(entries, entry)
	}

	return entries, nil
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

// runCommand executes the given command using clish and returns its output
func runCommand(command string) (string, error) {
	cmd := exec.Command("/opt/dell/os10/bin/clish", "-c", command)
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
		fmt.Println("Dell OS10 IP ARP Parser")
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
	return "Parses 'show ip arp' output for Dell OS10"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseARP(content)
}
