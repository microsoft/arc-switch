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
	DataType  string              `json:"data_type"`  // Always "cisco_nexus_lldp_neighbor"
	Timestamp string              `json:"timestamp"`  // ISO 8601 timestamp
	Date      string              `json:"date"`       // Date in YYYY-MM-DD format
	Message   LLDPNeighborData    `json:"message"`    // LLDP neighbor-specific data
}

// LLDPNeighborData represents the LLDP neighbor data within the message field
type LLDPNeighborData struct {
	ChassisID            string              `json:"chassis_id"`
	PortID               string              `json:"port_id"`
	LocalPortID          string              `json:"local_port_id"`
	PortDescription      string              `json:"port_description,omitempty"`
	SystemName           string              `json:"system_name,omitempty"`
	SystemDescription    string              `json:"system_description,omitempty"`
	TimeRemaining        int                 `json:"time_remaining"` // In seconds
	SystemCapabilities   []string            `json:"system_capabilities"`
	EnabledCapabilities  []string            `json:"enabled_capabilities"`
	ManagementAddress    string              `json:"management_address,omitempty"`
	ManagementAddressIPv6 string             `json:"management_address_ipv6,omitempty"`
	VlanID               string              `json:"vlan_id,omitempty"`
	MaxFrameSize         int                 `json:"max_frame_size"`
	VlanNames            map[string]string   `json:"vlan_names,omitempty"`
	LinkAggregation      LinkAggregationInfo `json:"link_aggregation"`
}

// LinkAggregationInfo represents link aggregation details
type LinkAggregationInfo struct {
	Capability string `json:"capability"`
	Status     string `json:"status"`
	LinkAggID  int    `json:"link_agg_id"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseCapabilities extracts capability codes from string like "B, R"
func parseCapabilities(caps string) []string {
	if caps == "not advertised" || caps == "" {
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

// parseTimeRemaining extracts seconds from strings like "40 seconds"
func parseTimeRemaining(timeStr string) int {
	parts := strings.Fields(timeStr)
	if len(parts) >= 1 {
		seconds, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0
		}
		return seconds
	}
	return 0
}

// parseMaxFrameSize extracts frame size from string
func parseMaxFrameSize(frameStr string) int {
	if frameStr == "not advertised" || frameStr == "" {
		return 0
	}
	size, err := strconv.Atoi(strings.TrimSpace(frameStr))
	if err != nil {
		return 0
	}
	return size
}

// parseVlanNames parses VLAN names from string like "1: default, 2: Unused_Ports, 6: HNV_PA"
func parseVlanNames(vlanStr string) map[string]string {
	if vlanStr == "not advertised" || vlanStr == "" {
		return nil
	}
	
	vlans := make(map[string]string)
	// Remove any leading/trailing whitespace and newlines
	vlanStr = strings.TrimSpace(vlanStr)
	
	// Split by comma, but be careful of multi-line entries
	parts := strings.Split(vlanStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if colonIdx := strings.Index(part, ":"); colonIdx != -1 {
			vlanID := strings.TrimSpace(part[:colonIdx])
			vlanName := strings.TrimSpace(part[colonIdx+1:])
			if vlanID != "" && vlanName != "" {
				vlans[vlanID] = vlanName
			}
		}
	}
	
	if len(vlans) == 0 {
		return nil
	}
	return vlans
}

// cleanValue removes "null" and returns empty string, otherwise returns trimmed value
func cleanValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "null" || value == "not advertised" {
		return ""
	}
	return value
}

// parseLLDPNeighbors parses the show lldp neighbors detail output
func parseLLDPNeighbors(content string) []StandardizedEntry {
	var neighbors []StandardizedEntry
	lines := strings.Split(content, "\n")
	
	timestamp := time.Now()
	
	// Regular expressions for parsing
	chassisRegex := regexp.MustCompile(`^Chassis id:\s+(.+)$`)
	portRegex := regexp.MustCompile(`^Port id:\s+(.+)$`)
	localPortRegex := regexp.MustCompile(`^Local Port id:\s+(.+)$`)
	portDescRegex := regexp.MustCompile(`^Port Description:\s+(.+)$`)
	systemNameRegex := regexp.MustCompile(`^System Name:\s+(.+)$`)
	systemDescRegex := regexp.MustCompile(`^System Description:\s+(.+)$`)
	timeRemainingRegex := regexp.MustCompile(`^Time remaining:\s+(.+)$`)
	sysCapRegex := regexp.MustCompile(`^System Capabilities:\s+(.+)$`)
	enabledCapRegex := regexp.MustCompile(`^Enabled Capabilities:\s+(.+)$`)
	mgmtAddrRegex := regexp.MustCompile(`^Management Address:\s+(.+)$`)
	mgmtAddrIPv6Regex := regexp.MustCompile(`^Management Address IPV6:\s+(.+)$`)
	vlanIDRegex := regexp.MustCompile(`^Vlan ID:\s+(.+)$`)
	maxFrameRegex := regexp.MustCompile(`^Max Frame Size:\s+(.+)$`)
	vlanNameRegex := regexp.MustCompile(`^\[Vlan ID: Vlan Name\]\s+(.+)$`)
	linkAggCapRegex := regexp.MustCompile(`^Capability:\s+(.+)$`)
	linkAggStatusRegex := regexp.MustCompile(`^Status\s*:\s+(.+)$`)
	linkAggIDRegex := regexp.MustCompile(`^Link agg ID\s*:\s+(.+)$`)
	
	var currentNeighbor *LLDPNeighborData
	var inSystemDesc bool
	var systemDescLines []string
	var inVlanNames bool
	var vlanNameLines []string
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Skip empty lines and header lines
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "Capability codes:") ||
			strings.HasPrefix(line, "  (") || strings.HasPrefix(line, "Device ID") ||
			strings.HasPrefix(line, "Total entries") {
			// Save current neighbor if exists
			if currentNeighbor != nil && !inSystemDesc {
				entry := StandardizedEntry{
					DataType:  "cisco_nexus_lldp_neighbor",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentNeighbor,
				}
				neighbors = append(neighbors, entry)
				currentNeighbor = nil
			}
			inSystemDesc = false
			systemDescLines = []string{}
			inVlanNames = false
			vlanNameLines = []string{}
			continue
		}
		
		// Skip the command line
		if strings.Contains(line, "show lldp neighbors") {
			continue
		}
		
		// Check for chassis ID (start of new neighbor)
		if chassisMatch := chassisRegex.FindStringSubmatch(line); chassisMatch != nil {
			// Save previous neighbor if exists
			if currentNeighbor != nil {
				entry := StandardizedEntry{
					DataType:  "cisco_nexus_lldp_neighbor",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentNeighbor,
				}
				neighbors = append(neighbors, entry)
			}
			
			// Start new neighbor
			currentNeighbor = &LLDPNeighborData{
				ChassisID: strings.TrimSpace(chassisMatch[1]),
				LinkAggregation: LinkAggregationInfo{},
				SystemCapabilities: []string{},
				EnabledCapabilities: []string{},
			}
			inSystemDesc = false
			systemDescLines = []string{}
			inVlanNames = false
			vlanNameLines = []string{}
			continue
		}
		
		if currentNeighbor == nil {
			continue
		}
		
		// Port ID
		if portMatch := portRegex.FindStringSubmatch(line); portMatch != nil {
			currentNeighbor.PortID = strings.TrimSpace(portMatch[1])
			continue
		}
		
		// Local Port ID
		if localPortMatch := localPortRegex.FindStringSubmatch(line); localPortMatch != nil {
			currentNeighbor.LocalPortID = strings.TrimSpace(localPortMatch[1])
			continue
		}
		
		// Port Description
		if portDescMatch := portDescRegex.FindStringSubmatch(line); portDescMatch != nil {
			currentNeighbor.PortDescription = cleanValue(portDescMatch[1])
			continue
		}
		
		// System Name
		if systemNameMatch := systemNameRegex.FindStringSubmatch(line); systemNameMatch != nil {
			currentNeighbor.SystemName = cleanValue(systemNameMatch[1])
			continue
		}
		
		// System Description (can be multi-line)
		if systemDescMatch := systemDescRegex.FindStringSubmatch(line); systemDescMatch != nil {
			desc := cleanValue(systemDescMatch[1])
			if desc != "" {
				systemDescLines = append(systemDescLines, desc)
				inSystemDesc = true
			}
			continue
		}
		
		// Continue collecting system description lines
		if inSystemDesc && !strings.HasPrefix(line, "Time remaining:") {
			systemDescLines = append(systemDescLines, strings.TrimSpace(line))
			continue
		}
		
		// Time Remaining
		if timeMatch := timeRemainingRegex.FindStringSubmatch(line); timeMatch != nil {
			if inSystemDesc {
				currentNeighbor.SystemDescription = strings.Join(systemDescLines, "\n")
				inSystemDesc = false
				systemDescLines = []string{}
			}
			currentNeighbor.TimeRemaining = parseTimeRemaining(timeMatch[1])
			continue
		}
		
		// System Capabilities
		if sysCapMatch := sysCapRegex.FindStringSubmatch(line); sysCapMatch != nil {
			currentNeighbor.SystemCapabilities = parseCapabilities(sysCapMatch[1])
			continue
		}
		
		// Enabled Capabilities
		if enabledCapMatch := enabledCapRegex.FindStringSubmatch(line); enabledCapMatch != nil {
			currentNeighbor.EnabledCapabilities = parseCapabilities(enabledCapMatch[1])
			continue
		}
		
		// Management Address
		if mgmtAddrMatch := mgmtAddrRegex.FindStringSubmatch(line); mgmtAddrMatch != nil {
			currentNeighbor.ManagementAddress = cleanValue(mgmtAddrMatch[1])
			continue
		}
		
		// Management Address IPv6
		if mgmtAddrIPv6Match := mgmtAddrIPv6Regex.FindStringSubmatch(line); mgmtAddrIPv6Match != nil {
			currentNeighbor.ManagementAddressIPv6 = cleanValue(mgmtAddrIPv6Match[1])
			continue
		}
		
		// VLAN ID
		if vlanIDMatch := vlanIDRegex.FindStringSubmatch(line); vlanIDMatch != nil && !strings.Contains(line, "[Vlan ID:") {
			currentNeighbor.VlanID = cleanValue(vlanIDMatch[1])
			continue
		}
		
		// Max Frame Size
		if maxFrameMatch := maxFrameRegex.FindStringSubmatch(line); maxFrameMatch != nil {
			currentNeighbor.MaxFrameSize = parseMaxFrameSize(maxFrameMatch[1])
			continue
		}
		
		// VLAN Name TLV (can be multi-line)
		if vlanNameMatch := vlanNameRegex.FindStringSubmatch(line); vlanNameMatch != nil {
			vlanStr := vlanNameMatch[1]
			if vlanStr != "not advertised" {
				vlanNameLines = append(vlanNameLines, vlanStr)
				inVlanNames = true
			}
			continue
		}
		
		// Continue collecting VLAN names
		if inVlanNames && !strings.Contains(line, "Link Aggregation TLV") && !strings.Contains(line, "Capability:") {
			// Check if this line starts with a number (VLAN ID)
			trimmedLine := strings.TrimSpace(line)
			if len(trimmedLine) > 0 && (trimmedLine[0] >= '0' && trimmedLine[0] <= '9') {
				vlanNameLines = append(vlanNameLines, trimmedLine)
			} else if strings.Contains(trimmedLine, ":") {
				// Continuation of previous line
				vlanNameLines[len(vlanNameLines)-1] += " " + trimmedLine
			}
			continue
		}
		
		// Link Aggregation TLV section
		if strings.Contains(line, "Link Aggregation TLV") {
			if inVlanNames && len(vlanNameLines) > 0 {
				currentNeighbor.VlanNames = parseVlanNames(strings.Join(vlanNameLines, ", "))
				inVlanNames = false
				vlanNameLines = []string{}
			}
			continue
		}
		
		// Link Aggregation Capability
		if linkAggCapMatch := linkAggCapRegex.FindStringSubmatch(line); linkAggCapMatch != nil {
			currentNeighbor.LinkAggregation.Capability = cleanValue(linkAggCapMatch[1])
			continue
		}
		
		// Link Aggregation Status
		if linkAggStatusMatch := linkAggStatusRegex.FindStringSubmatch(line); linkAggStatusMatch != nil {
			currentNeighbor.LinkAggregation.Status = cleanValue(linkAggStatusMatch[1])
			continue
		}
		
		// Link Aggregation ID
		if linkAggIDMatch := linkAggIDRegex.FindStringSubmatch(line); linkAggIDMatch != nil {
			id, _ := strconv.Atoi(strings.TrimSpace(linkAggIDMatch[1]))
			currentNeighbor.LinkAggregation.LinkAggID = id
			continue
		}
	}
	
	// Save last neighbor if exists
	if currentNeighbor != nil {
		entry := StandardizedEntry{
			DataType:  "cisco_nexus_lldp_neighbor",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentNeighbor,
		}
		neighbors = append(neighbors, entry)
	}
	
	return neighbors
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

// findLLDPCommand finds the lldp-neighbor command in the commands.json
func findLLDPCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "lldp-neighbor" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("lldp-neighbor command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show lldp neighbors detail' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Cisco Nexus LLDP Neighbor Parser")
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
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./lldp_neighbor_parser -input show-lldp.txt -output output.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./lldp_neighbor_parser -commands commands.json -output output.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./lldp_neighbor_parser -input show-lldp.txt")
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
			fmt.Fprintf(os.Stderr, "Error finding lldp-neighbor command: %v\n", err)
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
	return "Parses 'show lldp neighbor detail' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseLLDPNeighbors(content), nil
}