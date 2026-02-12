package interface_counters_parser

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

// StandardizedEntry represents the standardized JSON structure
type StandardizedEntry struct {
	DataType  string                `json:"data_type"`
	Timestamp string                `json:"timestamp"`
	Date      string                `json:"date"`
	Message   InterfaceCountersData `json:"message"`
}

// InterfaceCountersData represents the interface counters data within the message field
type InterfaceCountersData struct {
	InterfaceName string `json:"interface_name"`
	InterfaceType string `json:"interface_type"`
	Status        string `json:"status"`
	LineProtocol  string `json:"line_protocol"`

	// Ingress (Input) Counters
	InOctets    int64 `json:"in_octets"`
	InPackets   int64 `json:"in_packets"`
	InUcastPkts int64 `json:"in_ucast_pkts"`
	InMcastPkts int64 `json:"in_mcast_pkts"`
	InBcastPkts int64 `json:"in_bcast_pkts"`

	// Egress (Output) Counters
	OutOctets    int64 `json:"out_octets"`
	OutPackets   int64 `json:"out_packets"`
	OutUcastPkts int64 `json:"out_ucast_pkts"`
	OutMcastPkts int64 `json:"out_mcast_pkts"`
	OutBcastPkts int64 `json:"out_bcast_pkts"`

	// Error Counters
	InRunts     int64 `json:"in_runts"`
	InGiants    int64 `json:"in_giants"`
	InThrottles int64 `json:"in_throttles"`
	InCRC       int64 `json:"in_crc"`
	InOverrun   int64 `json:"in_overrun"`
	InDiscarded int64 `json:"in_discarded"`

	OutThrottles  int64 `json:"out_throttles"`
	OutDiscarded  int64 `json:"out_discarded"`
	OutCollisions int64 `json:"out_collisions"`

	// Rate Info
	InputRateMbps  int `json:"input_rate_mbps"`
	InputPktsSec   int `json:"input_pkts_sec"`
	OutputRateMbps int `json:"output_rate_mbps"`
	OutputPktsSec  int `json:"output_pkts_sec"`

	// Status indicators
	HasIngressData bool `json:"has_ingress_data"`
	HasEgressData  bool `json:"has_egress_data"`
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
	lowerName := strings.ToLower(interfaceName)
	switch {
	case strings.HasPrefix(lowerName, "ethernet"):
		return "ethernet"
	case strings.HasPrefix(lowerName, "port-channel"):
		return "port-channel"
	case strings.HasPrefix(lowerName, "vlan"):
		return "vlan"
	case strings.HasPrefix(lowerName, "management"):
		return "management"
	case strings.HasPrefix(lowerName, "loopback"):
		return "loopback"
	default:
		return "unknown"
	}
}

// parseInt64 safely converts string to int64
func parseInt64(value string) int64 {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", "")
	if value == "" || value == "--" {
		return -1
	}
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return -1
	}
	return result
}

// parseInt safely converts string to int
func parseInt(value string) int {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", "")
	if value == "" || value == "--" {
		return -1
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return result
}

// parseInterfaceCounters parses the show interface output from Dell OS10
func parseInterfaceCounters(content string) []StandardizedEntry {
	var interfaces []StandardizedEntry
	lines := strings.Split(content, "\n")
	timestamp := time.Now()

	// Regular expressions for parsing
	interfaceRegex := regexp.MustCompile(`^(Ethernet \S+|Port-channel \d+|Vlan \d+|Loopback \d+|Management \S+) is (\S+), line protocol is (\S+)`)
	interfaceRegex2 := regexp.MustCompile(`^(Ethernet \S+|Port-channel \d+|Vlan \d+|Loopback \d+|Management \S+) is (\S+),`)

	var currentInterface *InterfaceCountersData
	parsingInputStats := false
	parsingOutputStats := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for new interface block
		if match := interfaceRegex.FindStringSubmatch(line); match != nil {
			// Save previous interface
			if currentInterface != nil && (currentInterface.HasIngressData || currentInterface.HasEgressData) {
				entry := StandardizedEntry{
					DataType:  "dell_os10_interface_counters",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentInterface,
				}
				interfaces = append(interfaces, entry)
			}

			// Start new interface
			currentInterface = &InterfaceCountersData{
				InterfaceName:  match[1],
				InterfaceType:  parseInterfaceType(match[1]),
				Status:         match[2],
				LineProtocol:   match[3],
				InOctets:       -1,
				InPackets:      -1,
				InUcastPkts:    -1,
				InMcastPkts:    -1,
				InBcastPkts:    -1,
				OutOctets:      -1,
				OutPackets:     -1,
				OutUcastPkts:   -1,
				OutMcastPkts:   -1,
				OutBcastPkts:   -1,
				InRunts:        -1,
				InGiants:       -1,
				InThrottles:    -1,
				InCRC:          -1,
				InOverrun:      -1,
				InDiscarded:    -1,
				OutThrottles:   -1,
				OutDiscarded:   -1,
				OutCollisions:  -1,
				HasIngressData: false,
				HasEgressData:  false,
			}
			parsingInputStats = false
			parsingOutputStats = false
			continue
		}

		// Alternative interface line format (without line protocol)
		if match := interfaceRegex2.FindStringSubmatch(line); match != nil && currentInterface == nil {
			currentInterface = &InterfaceCountersData{
				InterfaceName:  match[1],
				InterfaceType:  parseInterfaceType(match[1]),
				Status:         match[2],
				LineProtocol:   "unknown",
				InOctets:       -1,
				InPackets:      -1,
				InUcastPkts:    -1,
				InMcastPkts:    -1,
				InBcastPkts:    -1,
				OutOctets:      -1,
				OutPackets:     -1,
				OutUcastPkts:   -1,
				OutMcastPkts:   -1,
				OutBcastPkts:   -1,
				InRunts:        -1,
				InGiants:       -1,
				InThrottles:    -1,
				InCRC:          -1,
				InOverrun:      -1,
				InDiscarded:    -1,
				OutThrottles:   -1,
				OutDiscarded:   -1,
				OutCollisions:  -1,
				HasIngressData: false,
				HasEgressData:  false,
			}
			parsingInputStats = false
			parsingOutputStats = false
			continue
		}

		if currentInterface == nil {
			continue
		}

		// Detect statistics sections
		if strings.HasPrefix(line, "Input statistics:") {
			parsingInputStats = true
			parsingOutputStats = false
			continue
		}
		if strings.HasPrefix(line, "Output statistics:") {
			parsingInputStats = false
			parsingOutputStats = true
			continue
		}
		if strings.HasPrefix(line, "Rate Info") || strings.HasPrefix(line, "Time since") {
			parsingInputStats = false
			parsingOutputStats = false
		}

		// Parse input statistics
		if parsingInputStats {
			// Parse "258756070469 packets, 224206026489044 octets"
			if strings.Contains(line, "packets,") && strings.Contains(line, "octets") {
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					currentInterface.InPackets = parseInt64(parts[0])
					currentInterface.InOctets = parseInt64(parts[2])
					currentInterface.HasIngressData = true
				}
			}
			// Parse "3322520 Multicasts, 5843785 Broadcasts, 258746758082 Unicasts"
			if strings.Contains(line, "Multicasts") && strings.Contains(line, "Broadcasts") && strings.Contains(line, "Unicasts") {
				parts := strings.Fields(line)
				if len(parts) >= 6 {
					currentInterface.InMcastPkts = parseInt64(parts[0])
					currentInterface.InBcastPkts = parseInt64(parts[2])
					currentInterface.InUcastPkts = parseInt64(parts[4])
					currentInterface.HasIngressData = true
				}
			}
			// Parse "0 runts, 0 giants, 104 throttles"
			if strings.Contains(line, "runts") && strings.Contains(line, "giants") && strings.Contains(line, "throttles") {
				parts := strings.Fields(line)
				if len(parts) >= 6 {
					currentInterface.InRunts = parseInt64(parts[0])
					currentInterface.InGiants = parseInt64(parts[2])
					currentInterface.InThrottles = parseInt64(parts[4])
				}
			}
			// Parse "0 CRC, 0 overrun, 0 discarded"
			if strings.Contains(line, "CRC") && strings.Contains(line, "overrun") && strings.Contains(line, "discarded") {
				parts := strings.Fields(line)
				if len(parts) >= 6 {
					currentInterface.InCRC = parseInt64(parts[0])
					currentInterface.InOverrun = parseInt64(parts[2])
					currentInterface.InDiscarded = parseInt64(parts[4])
				}
			}
		}

		// Parse output statistics
		if parsingOutputStats {
			// Parse "464017689029 packets, 460659024245083 octets"
			if strings.Contains(line, "packets,") && strings.Contains(line, "octets") {
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					currentInterface.OutPackets = parseInt64(parts[0])
					currentInterface.OutOctets = parseInt64(parts[2])
					currentInterface.HasEgressData = true
				}
			}
			// Parse "114945286 Multicasts, 27839366 Broadcasts, 463860550477 Unicasts"
			if strings.Contains(line, "Multicasts") && strings.Contains(line, "Broadcasts") && strings.Contains(line, "Unicasts") {
				parts := strings.Fields(line)
				if len(parts) >= 6 {
					currentInterface.OutMcastPkts = parseInt64(parts[0])
					currentInterface.OutBcastPkts = parseInt64(parts[2])
					currentInterface.OutUcastPkts = parseInt64(parts[4])
					currentInterface.HasEgressData = true
				}
			}
			// Parse "0 throttles, 296897 discarded, 0 Collisions"
			if strings.Contains(line, "throttles") && strings.Contains(line, "discarded") && strings.Contains(line, "Collisions") {
				parts := strings.Fields(line)
				if len(parts) >= 6 {
					currentInterface.OutThrottles = parseInt64(parts[0])
					currentInterface.OutDiscarded = parseInt64(parts[2])
					currentInterface.OutCollisions = parseInt64(parts[4])
				}
			}
		}

		// Parse rate info
		// "Input 11 Mbits/sec, 1806 packets/sec, 0% of line rate"
		if strings.HasPrefix(line, "Input") && strings.Contains(line, "Mbits/sec") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentInterface.InputRateMbps = parseInt(parts[1])
				currentInterface.InputPktsSec = parseInt(parts[3])
			}
		}
		// "Output 14 Mbits/sec, 2102 packets/sec, 0% of line rate"
		if strings.HasPrefix(line, "Output") && strings.Contains(line, "Mbits/sec") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentInterface.OutputRateMbps = parseInt(parts[1])
				currentInterface.OutputPktsSec = parseInt(parts[3])
			}
		}
	}

	// Save last interface
	if currentInterface != nil && (currentInterface.HasIngressData || currentInterface.HasEgressData) {
		entry := StandardizedEntry{
			DataType:  "dell_os10_interface_counters",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentInterface,
		}
		interfaces = append(interfaces, entry)
	}

	return interfaces
}

// runCommand executes a command on the Dell OS10 switch using clish
func runCommand(command string) (string, error) {
	cmd := exec.Command("/opt/dell/os10/bin/clish", "-c", command)
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
	var inputFile = flag.String("input", "", "Input file containing 'show interface' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Interface Counters Parser")
		fmt.Println("Parses 'show interface' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  interface_counters_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show interface' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		return
	}

	var content string

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

	// Output results as individual JSON objects (one per line)
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		for _, entry := range interfaces {
			jsonData, err := json.Marshal(entry)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling entry to JSON: %v\n", err)
				os.Exit(1)
			}
			_, err = file.Write(append(jsonData, '\n'))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Fprintf(os.Stderr, "Interface counters data written to %s\n", *outputFile)
	} else {
		for _, entry := range interfaces {
			jsonData, err := json.Marshal(entry)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling entry to JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(jsonData))
		}
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

// GetDescription returns the parser description
func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show interface' output for Dell OS10"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseInterfaceCounters(content), nil
}
