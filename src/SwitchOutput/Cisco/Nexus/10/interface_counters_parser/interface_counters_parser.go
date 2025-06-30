package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InterfaceCounters represents a single interface with all its counter metrics
type InterfaceCounters struct {
	DataType       string `json:"data_type"`                  // Identifies the type of data for KQL queries
	Timestamp      string `json:"timestamp"`                  // Timestamp when the data was processed
	Date           string `json:"date"`                       // Date when the data was processed
	InterfaceName  string `json:"interface_name"`             // Interface name (e.g., Eth1/1, Po50, Vlan1, mgmt0, Tunnel1)
	InterfaceType  string `json:"interface_type"`             // Type of interface (ethernet, port-channel, vlan, management, tunnel)
	
	// Ingress (Input) Counters
	InOctets       int64  `json:"in_octets"`                  // Ingress octets
	InUcastPkts    int64  `json:"in_ucast_pkts"`              // Ingress unicast packets
	InMcastPkts    int64  `json:"in_mcast_pkts"`              // Ingress multicast packets
	InBcastPkts    int64  `json:"in_bcast_pkts"`              // Ingress broadcast packets
	
	// Egress (Output) Counters
	OutOctets      int64  `json:"out_octets"`                 // Egress octets
	OutUcastPkts   int64  `json:"out_ucast_pkts"`             // Egress unicast packets
	OutMcastPkts   int64  `json:"out_mcast_pkts"`             // Egress multicast packets
	OutBcastPkts   int64  `json:"out_bcast_pkts"`             // Egress broadcast packets
	
	// Status indicators
	HasIngressData bool   `json:"has_ingress_data"`           // True if ingress counters are available
	HasEgressData  bool   `json:"has_egress_data"`            // True if egress counters are available
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseInterfaceType determines the interface type based on the interface name
func parseInterfaceType(interfaceName string) string {
	switch {
	case strings.HasPrefix(interfaceName, "Eth"):
		return "ethernet"
	case strings.HasPrefix(interfaceName, "Po"):
		return "port-channel"
	case strings.HasPrefix(interfaceName, "Vlan"):
		return "vlan"
	case interfaceName == "mgmt0":
		return "management"
	case strings.HasPrefix(interfaceName, "Tunnel"):
		return "tunnel"
	default:
		return "unknown"
	}
}

// parseCounterValue converts string counter values to int64, handling special cases
func parseCounterValue(value string) int64 {
	value = strings.TrimSpace(value)
	
	// Handle special cases like "--" for unavailable counters
	if value == "--" || value == "" {
		return -1 // Use -1 to indicate unavailable data
	}
	
	// Convert to integer
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return -1 // Return -1 for invalid values
	}
	
	return result
}

// parseInterfaceCounters parses the show interface counters output
func parseInterfaceCounters(content string) []InterfaceCounters {
	var interfaces []InterfaceCounters
	interfaceMap := make(map[string]*InterfaceCounters)
	
	lines := strings.Split(content, "\n")
	currentSection := ""
	
	// Regular expressions for parsing
	headerRegex := regexp.MustCompile(`Port\s+(.+)`)
	interfaceRegex := regexp.MustCompile(`^(\S+)\s+(.+)$`)
	
	timestamp := time.Now()
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and separator lines
		if line == "" || strings.Contains(line, "-----") {
			continue
		}
		
		// Skip the command line
		if strings.Contains(line, "show interface counters") {
			continue
		}
		
		// Detect section headers
		if headerMatch := headerRegex.FindStringSubmatch(line); headerMatch != nil {
			currentSection = strings.TrimSpace(headerMatch[1])
			continue
		}
		
		// Parse interface data lines
		if interfaceMatch := interfaceRegex.FindStringSubmatch(line); interfaceMatch != nil {
			interfaceName := interfaceMatch[1]
			valuesStr := strings.TrimSpace(interfaceMatch[2])
			
			// Split values (should be 2 values per line)
			valueFields := strings.Fields(valuesStr)
			if len(valueFields) != 2 {
				continue // Skip malformed lines
			}
			
			// Get or create interface entry
			if _, exists := interfaceMap[interfaceName]; !exists {
				interfaceMap[interfaceName] = &InterfaceCounters{
					DataType:       "interface_counters",
					Timestamp:      timestamp.Format(time.RFC3339),
					Date:           timestamp.Format("2006-01-02"),
					InterfaceName:  interfaceName,
					InterfaceType:  parseInterfaceType(interfaceName),
					InOctets:       -1,
					InUcastPkts:    -1,
					InMcastPkts:    -1,
					InBcastPkts:    -1,
					OutOctets:      -1,
					OutUcastPkts:   -1,
					OutMcastPkts:   -1,
					OutBcastPkts:   -1,
					HasIngressData: false,
					HasEgressData:  false,
				}
			}
			
			iface := interfaceMap[interfaceName]
			
			// Parse values based on current section
			switch currentSection {
			case "InOctets                      InUcastPkts":
				iface.InOctets = parseCounterValue(valueFields[0])
				iface.InUcastPkts = parseCounterValue(valueFields[1])
				if iface.InOctets >= 0 || iface.InUcastPkts >= 0 {
					iface.HasIngressData = true
				}
				
			case "InMcastPkts                      InBcastPkts":
				iface.InMcastPkts = parseCounterValue(valueFields[0])
				iface.InBcastPkts = parseCounterValue(valueFields[1])
				if iface.InMcastPkts >= 0 || iface.InBcastPkts >= 0 {
					iface.HasIngressData = true
				}
				
			case "OutOctets                     OutUcastPkts":
				iface.OutOctets = parseCounterValue(valueFields[0])
				iface.OutUcastPkts = parseCounterValue(valueFields[1])
				if iface.OutOctets >= 0 || iface.OutUcastPkts >= 0 {
					iface.HasEgressData = true
				}
				
			case "OutMcastPkts                     OutBcastPkts":
				iface.OutMcastPkts = parseCounterValue(valueFields[0])
				iface.OutBcastPkts = parseCounterValue(valueFields[1])
				if iface.OutMcastPkts >= 0 || iface.OutBcastPkts >= 0 {
					iface.HasEgressData = true
				}
			}
		}
	}
	
	// Convert map to slice and filter out interfaces with no data
	for _, iface := range interfaceMap {
		// Only include interfaces that have at least some counter data
		if iface.HasIngressData || iface.HasEgressData {
			interfaces = append(interfaces, *iface)
		}
	}
	
	return interfaces
}

// runCommand executes a command on the Cisco switch using vsh
func runCommand(command string) (string, error) {
	cmd := exec.Command("vsh", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute command '%s': %v", command, err)
	}
	return string(output), nil
}

// loadCommandsFromFile loads commands from the commands.json file
func loadCommandsFromFile(filename string) (*CommandConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening commands file: %v", err)
	}
	defer file.Close()

	var config CommandConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error parsing commands file: %v", err)
	}

	return &config, nil
}

// findInterfaceCountersCommand finds the interface-counter command in the commands.json
func findInterfaceCountersCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "interface-counter" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("interface-counter command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show interface counters' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Cisco Nexus Interface Counters Parser")
		fmt.Println("Parses 'show interface counters' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  interface_counters_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show interface counters' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./interface_counters_parser -input show-interface-counter.txt -output output.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./interface_counters_parser -commands commands.json -output output.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./interface_counters_parser -input show-interface-counter.txt")
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

		command, err := findInterfaceCountersCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding interface counters command: %v\n", err)
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

	// Parse the interface counters data
	fmt.Fprintf(os.Stderr, "Parsing interface counters data...\n")
	interfaces := parseInterfaceCounters(content)
	fmt.Fprintf(os.Stderr, "Found %d interfaces with counter data\n", len(interfaces))

	// Convert to JSON
	jsonData, err := json.MarshalIndent(interfaces, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if *outputFile != "" {
		err = os.WriteFile(*outputFile, jsonData, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Interface counters data written to %s\n", *outputFile)
	} else {
		fmt.Println(string(jsonData))
	}
}
