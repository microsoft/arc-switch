package interface_counters_error_parser

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
	DataType  string                     `json:"data_type"`
	Timestamp string                     `json:"timestamp"`
	Date      string                     `json:"date"`
	Message   InterfaceErrorCountersData `json:"message"`
}

// InterfaceErrorCountersData represents the interface error counters data
type InterfaceErrorCountersData struct {
	InterfaceName string `json:"interface_name"`
	InterfaceType string `json:"interface_type"`

	// Input error counters
	RxErr      int64 `json:"rx_err"`
	RxDrop     int64 `json:"rx_drop"`
	RxOverrun  int64 `json:"rx_overrun"`
	RxCRC      int64 `json:"rx_crc"`
	RxRunts    int64 `json:"rx_runts"`
	RxGiants   int64 `json:"rx_giants"`
	RxThrottle int64 `json:"rx_throttle"`
	RxDiscards int64 `json:"rx_discards"`

	// Output error counters
	TxErr       int64 `json:"tx_err"`
	TxDrop      int64 `json:"tx_drop"`
	TxOverrun   int64 `json:"tx_overrun"`
	TxCollision int64 `json:"tx_collision"`
	TxThrottle  int64 `json:"tx_throttle"`
	TxDiscards  int64 `json:"tx_discards"`

	// Status
	HasErrorData bool `json:"has_error_data"`
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

// parseCounterValue converts string counter values to int64
func parseCounterValue(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "--" || value == "" || value == "N/A" {
		return -1
	}
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return -1
	}
	return result
}

// parseInterfaceErrorCounters parses the show interface counters errors output
func parseInterfaceErrorCounters(content string) []StandardizedEntry {
	var interfaces []StandardizedEntry
	lines := strings.Split(content, "\n")
	timestamp := time.Now()

	// Dell OS10 format:
	// Interface    RX-Err  RX-Drop  RX-OVR  TX-Err  TX-Drop  TX-OVR
	// ethernet1/1/1    0       0        0       0       0        0

	// Find header line to determine column positions
	headerRegex := regexp.MustCompile(`^\s*Interface\s+`)
	dataRegex := regexp.MustCompile(`^\s*(\S+)\s+(\d+|--)\s+(\d+|--)\s+(\d+|--)\s+(\d+|--)\s+(\d+|--)\s+(\d+|--)`)

	inTable := false

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect header
		if headerRegex.MatchString(line) {
			inTable = true
			continue
		}

		// Skip separator lines
		if strings.Contains(line, "----") {
			continue
		}

		// Parse data lines
		if inTable {
			if match := dataRegex.FindStringSubmatch(line); match != nil {
				interfaceName := match[1]

				entry := StandardizedEntry{
					DataType:  "dell_os10_interface_error_counters",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message: InterfaceErrorCountersData{
						InterfaceName: interfaceName,
						InterfaceType: parseInterfaceType(interfaceName),
						RxErr:         parseCounterValue(match[2]),
						RxDrop:        parseCounterValue(match[3]),
						RxOverrun:     parseCounterValue(match[4]),
						TxErr:         parseCounterValue(match[5]),
						TxDrop:        parseCounterValue(match[6]),
						TxOverrun:     parseCounterValue(match[7]),
						RxCRC:         -1,
						RxRunts:       -1,
						RxGiants:      -1,
						RxThrottle:    -1,
						RxDiscards:    -1,
						TxCollision:   -1,
						TxThrottle:    -1,
						TxDiscards:    -1,
						HasErrorData:  true,
					},
				}
				interfaces = append(interfaces, entry)
			}
		}
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

// findInterfaceErrorCountersCommand finds the interface-error-counter command
func findInterfaceErrorCountersCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "interface-error-counter" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("interface-error-counter command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show interface counters errors' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Interface Error Counters Parser")
		fmt.Println("Parses 'show interface counters errors' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  interface_counters_error_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show interface counters errors' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		return
	}

	var content string

	if *inputFile != "" {
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
		config, err := loadCommandsFromFile(*commandsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading commands file: %v\n", err)
			os.Exit(1)
		}

		command, err := findInterfaceErrorCountersCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding command: %v\n", err)
			os.Exit(1)
		}

		content, err = runCommand(command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: Either -input or -commands parameter is required\n")
		os.Exit(1)
	}

	interfaces := parseInterfaceErrorCounters(content)

	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		for _, entry := range interfaces {
			jsonData, _ := json.Marshal(entry)
			file.Write(append(jsonData, '\n'))
		}
	} else {
		for _, entry := range interfaces {
			jsonData, _ := json.Marshal(entry)
			fmt.Println(string(jsonData))
		}
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show interface counters errors' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseInterfaceErrorCounters(string(input)), nil
}
