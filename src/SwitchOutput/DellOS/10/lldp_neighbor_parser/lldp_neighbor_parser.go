package lldp_neighbor_parser

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
	DataType  string           `json:"data_type"`
	Timestamp string           `json:"timestamp"`
	Date      string           `json:"date"`
	Message   LLDPNeighborData `json:"message"`
}

// LLDPNeighborData represents the LLDP neighbor data within the message field
type LLDPNeighborData struct {
	LocalPortID              string   `json:"local_port_id"`
	RemoteChassisID          string   `json:"remote_chassis_id"`
	RemoteChassisIDSubtype   string   `json:"remote_chassis_id_subtype,omitempty"`
	RemotePortID             string   `json:"remote_port_id"`
	RemotePortSubtype        string   `json:"remote_port_subtype,omitempty"`
	RemotePortDescription    string   `json:"remote_port_description,omitempty"`
	RemoteSystemName         string   `json:"remote_system_name,omitempty"`
	RemoteSystemDescription  string   `json:"remote_system_description,omitempty"`
	RemoteTTL                int      `json:"remote_ttl"`
	RemoteMaxFrameSize       int      `json:"remote_max_frame_size"`
	RemoteAggregationStatus  string   `json:"remote_aggregation_status,omitempty"`
	ManagementAddressIPv4    string   `json:"management_address_ipv4,omitempty"`
	ManagementAddressIPv6    string   `json:"management_address_ipv6,omitempty"`
	SystemCapabilities       []string `json:"system_capabilities,omitempty"`
	EnabledCapabilities      []string `json:"enabled_capabilities,omitempty"`
	TimeSinceLastChange      string   `json:"time_since_last_change,omitempty"`
	AutoNegSupported         bool     `json:"auto_neg_supported"`
	AutoNegEnabled           bool     `json:"auto_neg_enabled"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseCapabilities extracts capabilities from strings like "Repeater, Bridge, Router"
func parseCapabilities(caps string) []string {
	if caps == "" || strings.ToLower(caps) == "not advertised" {
		return []string{}
	}

	result := []string{}
	parts := strings.Split(caps, ",")
	for _, part := range parts {
		cap := strings.TrimSpace(part)
		if cap != "" {
			result = append(result, cap)
		}
	}
	return result
}

// parseLLDPNeighbors parses the show lldp neighbors detail output for Dell OS10
func parseLLDPNeighbors(content string) []StandardizedEntry {
	var neighbors []StandardizedEntry
	lines := strings.Split(content, "\n")
	timestamp := time.Now()

	// Regular expressions for parsing Dell OS10 LLDP output
	chassisIDRegex := regexp.MustCompile(`^Remote Chassis ID:\s+(.+)$`)
	chassisSubtypeRegex := regexp.MustCompile(`^Remote Chassis ID Subtype:\s+(.+)$`)
	portIDRegex := regexp.MustCompile(`^Remote Port ID:\s+(.+)$`)
	portSubtypeRegex := regexp.MustCompile(`^Remote Port Subtype:\s+(.+)$`)
	portDescRegex := regexp.MustCompile(`^Remote Port Description:\s+(.+)$`)
	localPortRegex := regexp.MustCompile(`^Local Port ID:\s+(.+)$`)
	systemNameRegex := regexp.MustCompile(`^Remote System Name:\s+(.+)$`)
	systemDescRegex := regexp.MustCompile(`^Remote System Desc:\s+(.+)$`)
	ttlRegex := regexp.MustCompile(`^Remote TTL:\s+(\d+)`)
	maxFrameRegex := regexp.MustCompile(`^Remote Max Frame Size:\s+(\d+)`)
	aggStatusRegex := regexp.MustCompile(`^Remote Aggregation Status:\s+(.+)$`)
	mgmtAddrIPv4Regex := regexp.MustCompile(`^Remote Management Address \(IPv4\):\s+(.+)$`)
	mgmtAddrIPv6Regex := regexp.MustCompile(`^Remote Management Address \(IPv6\):\s+(.+)$`)
	existingCapsRegex := regexp.MustCompile(`^Existing System Capabilities:\s+(.+)$`)
	enabledCapsRegex := regexp.MustCompile(`^Enabled System Capabilities:\s+(.+)$`)
	timeSinceRegex := regexp.MustCompile(`^Time since last information change of this neighbor:\s+(.+)$`)
	autoNegSupportedRegex := regexp.MustCompile(`^\s*Auto-neg supported:\s+(\d+)`)
	autoNegEnabledRegex := regexp.MustCompile(`^\s*Auto-neg enabled:\s+(\d+)`)

	var currentNeighbor *LLDPNeighborData
	var inSystemDesc bool
	var systemDescLines []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for separator (end of neighbor block)
		if strings.HasPrefix(strings.TrimSpace(line), "----") {
			if currentNeighbor != nil && currentNeighbor.LocalPortID != "" {
				// Save system description if collecting
				if inSystemDesc && len(systemDescLines) > 0 {
					currentNeighbor.RemoteSystemDescription = strings.Join(systemDescLines, "\n")
				}
				entry := StandardizedEntry{
					DataType:  "dell_os10_lldp_neighbor",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentNeighbor,
				}
				neighbors = append(neighbors, entry)
			}
			currentNeighbor = nil
			inSystemDesc = false
			systemDescLines = []string{}
			continue
		}

		// Remote Chassis ID Subtype
		if match := chassisSubtypeRegex.FindStringSubmatch(line); match != nil {
			if currentNeighbor == nil {
				currentNeighbor = &LLDPNeighborData{
					SystemCapabilities:  []string{},
					EnabledCapabilities: []string{},
				}
			}
			currentNeighbor.RemoteChassisIDSubtype = strings.TrimSpace(match[1])
			continue
		}

		// Remote Chassis ID
		if match := chassisIDRegex.FindStringSubmatch(line); match != nil {
			if currentNeighbor == nil {
				currentNeighbor = &LLDPNeighborData{
					SystemCapabilities:  []string{},
					EnabledCapabilities: []string{},
				}
			}
			currentNeighbor.RemoteChassisID = strings.TrimSpace(match[1])
			continue
		}

		if currentNeighbor == nil {
			continue
		}

		// Stop collecting system description if we hit a new field
		if inSystemDesc && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if strings.Contains(line, ":") || strings.HasPrefix(strings.TrimSpace(line), "----") {
				currentNeighbor.RemoteSystemDescription = strings.Join(systemDescLines, "\n")
				inSystemDesc = false
				systemDescLines = []string{}
			}
		}

		// Continue collecting multi-line system description
		if inSystemDesc {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				systemDescLines = append(systemDescLines, trimmed)
			}
			continue
		}

		// Remote Port Subtype
		if match := portSubtypeRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.RemotePortSubtype = strings.TrimSpace(match[1])
			continue
		}

		// Remote Port ID
		if match := portIDRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.RemotePortID = strings.TrimSpace(match[1])
			continue
		}

		// Remote Port Description
		if match := portDescRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.RemotePortDescription = strings.TrimSpace(match[1])
			continue
		}

		// Local Port ID
		if match := localPortRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.LocalPortID = strings.TrimSpace(match[1])
			continue
		}

		// Remote System Name
		if match := systemNameRegex.FindStringSubmatch(line); match != nil {
			value := strings.TrimSpace(match[1])
			if strings.ToLower(value) != "not advertised" {
				currentNeighbor.RemoteSystemName = value
			}
			continue
		}

		// Remote System Description (can be multi-line)
		if match := systemDescRegex.FindStringSubmatch(line); match != nil {
			value := strings.TrimSpace(match[1])
			if strings.ToLower(value) != "not advertised" {
				systemDescLines = append(systemDescLines, value)
				inSystemDesc = true
			}
			continue
		}

		// Remote TTL
		if match := ttlRegex.FindStringSubmatch(line); match != nil {
			ttl, _ := strconv.Atoi(match[1])
			currentNeighbor.RemoteTTL = ttl
			continue
		}

		// Remote Max Frame Size
		if match := maxFrameRegex.FindStringSubmatch(line); match != nil {
			size, _ := strconv.Atoi(match[1])
			currentNeighbor.RemoteMaxFrameSize = size
			continue
		}

		// Remote Aggregation Status
		if match := aggStatusRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.RemoteAggregationStatus = strings.TrimSpace(match[1])
			continue
		}

		// Management Address IPv4
		if match := mgmtAddrIPv4Regex.FindStringSubmatch(line); match != nil {
			currentNeighbor.ManagementAddressIPv4 = strings.TrimSpace(match[1])
			continue
		}

		// Management Address IPv6
		if match := mgmtAddrIPv6Regex.FindStringSubmatch(line); match != nil {
			currentNeighbor.ManagementAddressIPv6 = strings.TrimSpace(match[1])
			continue
		}

		// Existing System Capabilities
		if match := existingCapsRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.SystemCapabilities = parseCapabilities(match[1])
			continue
		}

		// Enabled System Capabilities
		if match := enabledCapsRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.EnabledCapabilities = parseCapabilities(match[1])
			continue
		}

		// Time since last change
		if match := timeSinceRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.TimeSinceLastChange = strings.TrimSpace(match[1])
			continue
		}

		// Auto-negotiation supported
		if match := autoNegSupportedRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.AutoNegSupported = match[1] == "1"
			continue
		}

		// Auto-negotiation enabled
		if match := autoNegEnabledRegex.FindStringSubmatch(line); match != nil {
			currentNeighbor.AutoNegEnabled = match[1] == "1"
			continue
		}
	}

	// Save last neighbor if exists
	if currentNeighbor != nil && currentNeighbor.LocalPortID != "" {
		if inSystemDesc && len(systemDescLines) > 0 {
			currentNeighbor.RemoteSystemDescription = strings.Join(systemDescLines, "\n")
		}
		entry := StandardizedEntry{
			DataType:  "dell_os10_lldp_neighbor",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentNeighbor,
		}
		neighbors = append(neighbors, entry)
	}

	return neighbors
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

// findLLDPCommand finds the lldp-neighbors command in the commands.json
func findLLDPCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "lldp-neighbors" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("lldp-neighbors command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show lldp neighbors detail' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 LLDP Neighbor Parser")
		fmt.Println("Parses 'show lldp neighbors detail' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  lldp_neighbor_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show lldp neighbors detail' output")
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

		command, err := findLLDPCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding lldp-neighbors command: %v\n", err)
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

	// Parse the LLDP neighbor data
	fmt.Fprintf(os.Stderr, "Parsing LLDP neighbor data...\n")
	neighbors := parseLLDPNeighbors(content)
	fmt.Fprintf(os.Stderr, "Found %d LLDP neighbors\n", len(neighbors))

	// Output results as individual JSON objects (one per line)
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		for _, entry := range neighbors {
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
		fmt.Fprintf(os.Stderr, "LLDP neighbor data written to %s\n", *outputFile)
	} else {
		for _, entry := range neighbors {
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
	return "Parses 'show lldp neighbors detail' output for Dell OS10"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseLLDPNeighbors(content), nil
}
