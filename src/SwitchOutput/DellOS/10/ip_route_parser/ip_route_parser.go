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
	DataType  string      `json:"data_type"`
	Timestamp string      `json:"timestamp"`
	Date      string      `json:"date"`
	Message   IPRouteData `json:"message"`
}

// IPRouteData represents the IP route data within the message field
type IPRouteData struct {
	VRF           string    `json:"vrf"`
	Destination   string    `json:"destination"`
	Prefix        string    `json:"prefix"`
	PrefixLength  int       `json:"prefix_length"`
	Gateway       string    `json:"gateway"`
	Interface     string    `json:"interface"`
	RouteType     string    `json:"route_type"`
	RouteTypeCode string    `json:"route_type_code"`
	Distance      int       `json:"distance"`
	Metric        int       `json:"metric"`
	LastChange    string    `json:"last_change"`
	IsActive      bool      `json:"is_active"`
	IsDefault     bool      `json:"is_default"`
	IsSummary     bool      `json:"is_summary"`
	NextHops      []NextHop `json:"next_hops,omitempty"`
}

// NextHop represents a single next-hop entry (for ECMP routes)
type NextHop struct {
	Gateway   string `json:"gateway"`
	Interface string `json:"interface"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseRouteType converts route code to human-readable type
func parseRouteType(code string) string {
	code = strings.TrimSpace(code)
	switch code {
	case "C":
		return "connected"
	case "S":
		return "static"
	case "B":
		return "bgp"
	case "B IN":
		return "bgp-internal"
	case "B EX":
		return "bgp-external"
	case "O":
		return "ospf"
	case "O IA":
		return "ospf-inter-area"
	case "O N1":
		return "ospf-nssa-type1"
	case "O N2":
		return "ospf-nssa-type2"
	case "O E1":
		return "ospf-external-type1"
	case "O E2":
		return "ospf-external-type2"
	default:
		return "unknown"
	}
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

// parseDistMetric extracts distance and metric from "dist/metric" format
func parseDistMetric(s string) (int, int) {
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		dist, _ := strconv.Atoi(parts[0])
		metric, _ := strconv.Atoi(parts[1])
		return dist, metric
	}
	return 0, 0
}

// parseIPRoutes parses the show ip route output for Dell OS10
func parseIPRoutes(content string) []StandardizedEntry {
	var routes []StandardizedEntry
	lines := strings.Split(content, "\n")
	timestamp := time.Now()

	var currentVRF = "default"
	inRouteTable := false

	// Regular expression for Dell OS10 route lines
	// Format: "C   10.1.1.0/24      via 10.1.1.1      vlan100      0/0           01:16:56"
	// Or: "B EX 10.1.2.0/24     via 10.1.2.1      vlan101      20/0          01:16:56"
	routeLineRegex := regexp.MustCompile(`^([CSBOP>*+]+(?:\s+(?:IN|EX|IA|N1|N2|E1|E2))?)\s+(\d+\.\d+\.\d+\.\d+/\d+)\s+via\s+(\S+)\s+(\S+)\s+(\d+/\d+)\s+(\S+)`)

	// Alternative format for directly connected routes
	directRouteRegex := regexp.MustCompile(`^([CSBOP>*+]+(?:\s+(?:IN|EX|IA|N1|N2|E1|E2))?)\s+(\d+\.\d+\.\d+\.\d+/\d+)\s+Direct-connect\s+(\S+)\s+(\d+/\d+)\s+(\S+)`)

	// VRF regex
	vrfRegex := regexp.MustCompile(`VRF:\s+(\S+)`)

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for VRF specification
		if match := vrfRegex.FindStringSubmatch(line); match != nil {
			currentVRF = match[1]
			continue
		}

		// Detect start of route table (after header line with dashes)
		if strings.HasPrefix(strings.TrimSpace(line), "---") {
			inRouteTable = true
			continue
		}

		// Skip code definitions and header lines
		if strings.Contains(line, "Codes:") || strings.Contains(line, "Gateway of last resort") ||
			strings.Contains(line, "Destination") || strings.Contains(line, "connected S -") {
			continue
		}

		if !inRouteTable {
			// Try to match route lines even before seeing dashes
			if !strings.HasPrefix(line, "C") && !strings.HasPrefix(line, "S") &&
				!strings.HasPrefix(line, "B") && !strings.HasPrefix(line, "O") &&
				!strings.HasPrefix(line, ">") && !strings.HasPrefix(line, "*") {
				continue
			}
		}

		// Parse route lines
		if match := routeLineRegex.FindStringSubmatch(line); match != nil {
			routeCode := strings.TrimSpace(match[1])
			destination := match[2]
			gateway := match[3]
			iface := match[4]
			distMetric := match[5]
			lastChange := match[6]

			prefix, prefixLen := parsePrefix(destination)
			dist, metric := parseDistMetric(distMetric)

			// Parse flags from route code
			isActive := !strings.Contains(routeCode, ">")
			isDefault := strings.Contains(routeCode, "*")
			isSummary := strings.Contains(routeCode, "+")

			// Extract the actual route type code (C, S, B, O, etc.)
			typeCode := strings.TrimLeft(routeCode, ">*+")
			typeCode = strings.TrimSpace(typeCode)

			route := StandardizedEntry{
				DataType:  "dell_os10_ip_route",
				Timestamp: timestamp.Format(time.RFC3339),
				Date:      timestamp.Format("2006-01-02"),
				Message: IPRouteData{
					VRF:           currentVRF,
					Destination:   destination,
					Prefix:        prefix,
					PrefixLength:  prefixLen,
					Gateway:       gateway,
					Interface:     iface,
					RouteType:     parseRouteType(typeCode),
					RouteTypeCode: typeCode,
					Distance:      dist,
					Metric:        metric,
					LastChange:    lastChange,
					IsActive:      isActive,
					IsDefault:     isDefault,
					IsSummary:     isSummary,
				},
			}
			routes = append(routes, route)
			continue
		}

		// Parse directly connected routes
		if match := directRouteRegex.FindStringSubmatch(line); match != nil {
			routeCode := strings.TrimSpace(match[1])
			destination := match[2]
			iface := match[3]
			distMetric := match[4]
			lastChange := match[5]

			prefix, prefixLen := parsePrefix(destination)
			dist, metric := parseDistMetric(distMetric)

			// Parse flags from route code
			isActive := !strings.Contains(routeCode, ">")
			isDefault := strings.Contains(routeCode, "*")
			isSummary := strings.Contains(routeCode, "+")

			typeCode := strings.TrimLeft(routeCode, ">*+")
			typeCode = strings.TrimSpace(typeCode)

			route := StandardizedEntry{
				DataType:  "dell_os10_ip_route",
				Timestamp: timestamp.Format(time.RFC3339),
				Date:      timestamp.Format("2006-01-02"),
				Message: IPRouteData{
					VRF:           currentVRF,
					Destination:   destination,
					Prefix:        prefix,
					PrefixLength:  prefixLen,
					Gateway:       "direct",
					Interface:     iface,
					RouteType:     parseRouteType(typeCode),
					RouteTypeCode: typeCode,
					Distance:      dist,
					Metric:        metric,
					LastChange:    lastChange,
					IsActive:      isActive,
					IsDefault:     isDefault,
					IsSummary:     isSummary,
				},
			}
			routes = append(routes, route)
		}
	}

	return routes
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
		fmt.Println("Dell OS10 IP Route Parser")
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
	return "Parses 'show ip route' output for Dell OS10"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseIPRoutes(content), nil
}
