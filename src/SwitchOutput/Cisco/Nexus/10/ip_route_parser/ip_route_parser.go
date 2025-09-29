package ip_route_parser

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
	DataType  string      `json:"data_type"`  // Always "cisco_nexus_ip_route"
	Timestamp string      `json:"timestamp"`  // ISO 8601 timestamp
	Date      string      `json:"date"`       // Date in YYYY-MM-DD format
	Message   IPRouteData `json:"message"`    // IP route-specific data
}

// IPRouteData represents the IP route data within the message field
type IPRouteData struct {
	VRF          string       `json:"vrf"`           // VRF name (e.g., "default")
	Network      string       `json:"network"`       // Network address with prefix (e.g., "192.168.1.0/24")
	Prefix       string       `json:"prefix"`        // Just the IP part
	PrefixLength int          `json:"prefix_length"` // Prefix length (e.g., 24)
	UBest        int          `json:"ubest"`         // Best unicast paths
	MBest        int          `json:"mbest"`         // Best multicast paths
	RouteType    string       `json:"route_type"`    // Type (attached, bgp, etc.)
	NextHops     []NextHop    `json:"next_hops"`     // List of next-hop entries
}

// NextHop represents a single next-hop entry
type NextHop struct {
	Via         string   `json:"via"`           // Next-hop IP address
	Interface   string   `json:"interface"`     // Egress interface
	Preference  int      `json:"preference"`    // Route preference
	Metric      int      `json:"metric"`        // Route metric
	Age         string   `json:"age"`           // Age of the route
	Protocol    string   `json:"protocol"`      // Protocol (bgp, direct, local, hsrp)
	Attributes  []string `json:"attributes"`    // Additional attributes (internal, external, tag)
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parsePrefix extracts IP and prefix length from network string
func parsePrefix(network string) (string, int) {
	parts := strings.Split(network, "/")
	if len(parts) == 2 {
		prefix := parts[0]
		length, _ := strconv.Atoi(parts[1])
		return prefix, length
	}
	return network, 32 // Default to /32 for host routes
}

// parsePreferenceMetric extracts preference and metric from [x/y] format
func parsePreferenceMetric(s string) (int, int) {
	// Remove brackets and split
	s = strings.Trim(s, "[]")
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		pref, _ := strconv.Atoi(parts[0])
		metric, _ := strconv.Atoi(parts[1])
		return pref, metric
	}
	return 0, 0
}

// determineRouteType determines the route type based on protocol and attributes
func determineRouteType(protocol string, attached bool) string {
	if attached {
		return "attached"
	}
	if strings.Contains(protocol, "bgp") {
		return "bgp"
	}
	switch protocol {
	case "direct":
		return "direct"
	case "local":
		return "local"
	case "hsrp":
		return "hsrp"
	default:
		return "unknown"
	}
}

// parseIPRoutes parses the show ip route output
func parseIPRoutes(content string) []StandardizedEntry {
	var routes []StandardizedEntry
	lines := strings.Split(content, "\n")
	
	timestamp := time.Now()
	
	// Regular expressions for parsing
	vrfRegex := regexp.MustCompile(`IP Route Table for VRF "([^"]+)"`)
	networkRegex := regexp.MustCompile(`^(\S+),\s+ubest/mbest:\s+(\d+)/(\d+)(.*)$`)
	nextHopRegex := regexp.MustCompile(`^\s*\*via\s+(\S+),\s+(.*)$`)
	
	var currentVRF string = "default"
	var currentRoute *IPRouteData
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Skip empty lines and header lines
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "'") || strings.HasPrefix(line, "%") {
			continue
		}
		
		// Skip the command line
		if strings.Contains(line, "show ip route") {
			continue
		}
		
		// Check for VRF specification
		if vrfMatch := vrfRegex.FindStringSubmatch(line); vrfMatch != nil {
			currentVRF = vrfMatch[1]
			continue
		}
		
		// Check for network entry
		if networkMatch := networkRegex.FindStringSubmatch(line); networkMatch != nil {
			// Save previous route if exists
			if currentRoute != nil {
				entry := StandardizedEntry{
					DataType:  "cisco_nexus_ip_route",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentRoute,
				}
				routes = append(routes, entry)
			}
			
			// Start new route
			network := networkMatch[1]
			prefix, prefixLen := parsePrefix(network)
			ubest, _ := strconv.Atoi(networkMatch[2])
			mbest, _ := strconv.Atoi(networkMatch[3])
			
			// Check if "attached" is in the remainder
			attached := strings.Contains(networkMatch[4], "attached")
			
			currentRoute = &IPRouteData{
				VRF:          currentVRF,
				Network:      network,
				Prefix:       prefix,
				PrefixLength: prefixLen,
				UBest:        ubest,
				MBest:        mbest,
				NextHops:     []NextHop{},
			}
			
			// Set initial route type
			if attached {
				currentRoute.RouteType = "attached"
			}
			continue
		}
		
		// Parse next-hop entries
		if currentRoute != nil {
			if nextHopMatch := nextHopRegex.FindStringSubmatch(line); nextHopMatch != nil {
				nextHop := NextHop{
					Via:        nextHopMatch[1],
					Attributes: []string{},
				}
				
				// Parse the remainder of the line
				remainder := nextHopMatch[2]
				parts := strings.Split(remainder, ",")
				
				for _, part := range parts {
					part = strings.TrimSpace(part)
					
					// Check for interface (starts with capital letter or 'lo')
					if len(part) > 0 && (strings.HasPrefix(part, "Eth") || strings.HasPrefix(part, "Vlan") || 
						strings.HasPrefix(part, "Po") || strings.HasPrefix(part, "Lo") || strings.HasPrefix(part, "Tunnel")) {
						nextHop.Interface = part
					} else if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
						// Preference/metric
						nextHop.Preference, nextHop.Metric = parsePreferenceMetric(part)
					} else if strings.Contains(part, "w") || strings.Contains(part, "d") {
						// Age (e.g., "14w1d", "37w6d")
						nextHop.Age = part
					} else if strings.HasPrefix(part, "bgp-") || part == "direct" || part == "local" || part == "hsrp" {
						// Protocol
						nextHop.Protocol = part
						// Update route type if not already set
						if currentRoute.RouteType == "" || currentRoute.RouteType == "attached" {
							currentRoute.RouteType = determineRouteType(part, currentRoute.RouteType == "attached")
						}
					} else if part == "internal" || part == "external" {
						// BGP attributes
						nextHop.Attributes = append(nextHop.Attributes, part)
					} else if strings.HasPrefix(part, "tag ") {
						// Tag attribute
						nextHop.Attributes = append(nextHop.Attributes, part)
					}
				}
				
				currentRoute.NextHops = append(currentRoute.NextHops, nextHop)
			}
		}
	}
	
	// Save last route if exists
	if currentRoute != nil {
		entry := StandardizedEntry{
			DataType:  "cisco_nexus_ip_route",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentRoute,
		}
		routes = append(routes, entry)
	}
	
	return routes
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

// findIPRouteCommand finds the ip-route command in the commands.json
func findIPRouteCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "ip-route" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("ip-route command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show ip route' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Cisco Nexus IP Route Parser")
		fmt.Println("Parses 'show ip route' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  ip_route_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show ip route' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./ip_route_parser -input show-ip-route.txt -output output.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./ip_route_parser -commands commands.json -output output.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./ip_route_parser -input show-ip-route.txt")
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

		command, err := findIPRouteCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding ip-route command: %v\n", err)
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

	// Parse the IP route data
	fmt.Fprintf(os.Stderr, "Parsing IP route data...\n")
	routes := parseIPRoutes(content)
	fmt.Fprintf(os.Stderr, "Found %d routes\n", len(routes))

	// Output results as individual JSON objects (one per line)
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		for _, entry := range routes {
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
		fmt.Fprintf(os.Stderr, "IP route data written to %s\n", *outputFile)
	} else {
		for _, entry := range routes {
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
	return "Parses 'show ip route' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseIPRoutes(content), nil
}