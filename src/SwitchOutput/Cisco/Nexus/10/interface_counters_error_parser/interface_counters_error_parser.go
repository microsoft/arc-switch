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
	DataType  string                     `json:"data_type"`  // Always "cisco_nexus_interface_error_counters"
	Timestamp string                     `json:"timestamp"`  // ISO 8601 timestamp
	Date      string                     `json:"date"`       // Date in YYYY-MM-DD format
	Message   InterfaceErrorCountersData `json:"message"`    // Interface error counters-specific data
}

// InterfaceErrorCountersData represents the interface error counters data within the message field
type InterfaceErrorCountersData struct {
	InterfaceName string `json:"interface_name"` // Interface name (e.g., Eth1/1, Po50, mgmt0)
	InterfaceType string `json:"interface_type"` // Type of interface (ethernet, port-channel, management)

	// First section: Align-Err, FCS-Err, Xmit-Err, Rcv-Err, UnderSize, OutDiscards
	AlignErr    int64 `json:"align_err"`     // Alignment errors
	FCSErr      int64 `json:"fcs_err"`       // Frame Check Sequence errors
	XmitErr     int64 `json:"xmit_err"`      // Transmit errors
	RcvErr      int64 `json:"rcv_err"`       // Receive errors
	UnderSize   int64 `json:"under_size"`    // Undersized packets
	OutDiscards int64 `json:"out_discards"`  // Output discards

	// Second section: Single-Col, Multi-Col, Late-Col, Exces-Col, Carri-Sen, Runts
	SingleCol int64 `json:"single_col"`  // Single collisions
	MultiCol  int64 `json:"multi_col"`   // Multiple collisions
	LateCol   int64 `json:"late_col"`    // Late collisions
	ExcesCol  int64 `json:"exces_col"`   // Excessive collisions
	CarriSen  int64 `json:"carri_sen"`   // Carrier sense errors
	Runts     int64 `json:"runts"`       // Runt packets

	// Third section: Giants, SQETest-Err, Deferred-Tx, IntMacTx-Er, IntMacRx-Er, Symbol-Err
	Giants      int64 `json:"giants"`        // Giant packets
	SQETestErr  int64 `json:"sqetest_err"`   // SQE test errors
	DeferredTx  int64 `json:"deferred_tx"`   // Deferred transmissions
	IntMacTxEr  int64 `json:"intmac_tx_er"`  // Internal MAC transmit errors
	IntMacRxEr  int64 `json:"intmac_rx_er"`  // Internal MAC receive errors
	SymbolErr   int64 `json:"symbol_err"`    // Symbol errors

	// Fourth section: InDiscards
	InDiscards int64 `json:"in_discards"` // Input discards

	// Fifth section: Stomped-CRC (only for ethernet interfaces)
	StompedCRC int64 `json:"stomped_crc"` // Stomped CRC errors

	// Status indicators
	HasErrorData bool `json:"has_error_data"` // True if error counters are available
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

// parseInterfaceErrorCounters parses the show interface counters errors output
func parseInterfaceErrorCounters(content string) []StandardizedEntry {
	var interfaces []StandardizedEntry
	interfaceMap := make(map[string]*InterfaceErrorCountersData)

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
		if strings.Contains(line, "show interface counters errors") {
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

			// Split values by whitespace
			valueFields := strings.Fields(valuesStr)

			// Get or create interface entry
			if _, exists := interfaceMap[interfaceName]; !exists {
				interfaceMap[interfaceName] = &InterfaceErrorCountersData{
					InterfaceName: interfaceName,
					InterfaceType: parseInterfaceType(interfaceName),
					AlignErr:      -1,
					FCSErr:        -1,
					XmitErr:       -1,
					RcvErr:        -1,
					UnderSize:     -1,
					OutDiscards:   -1,
					SingleCol:     -1,
					MultiCol:      -1,
					LateCol:       -1,
					ExcesCol:      -1,
					CarriSen:      -1,
					Runts:         -1,
					Giants:        -1,
					SQETestErr:    -1,
					DeferredTx:    -1,
					IntMacTxEr:    -1,
					IntMacRxEr:    -1,
					SymbolErr:     -1,
					InDiscards:    -1,
					StompedCRC:    -1,
					HasErrorData:  false,
				}
			}

			iface := interfaceMap[interfaceName]

			// Parse values based on current section
			switch {
			case strings.Contains(currentSection, "Align-Err") && strings.Contains(currentSection, "FCS-Err"):
				// Section 1: Align-Err, FCS-Err, Xmit-Err, Rcv-Err, UnderSize, OutDiscards
				if len(valueFields) >= 6 {
					iface.AlignErr = parseCounterValue(valueFields[0])
					iface.FCSErr = parseCounterValue(valueFields[1])
					iface.XmitErr = parseCounterValue(valueFields[2])
					iface.RcvErr = parseCounterValue(valueFields[3])
					iface.UnderSize = parseCounterValue(valueFields[4])
					iface.OutDiscards = parseCounterValue(valueFields[5])
					if iface.AlignErr >= 0 || iface.FCSErr >= 0 || iface.XmitErr >= 0 ||
						iface.RcvErr >= 0 || iface.UnderSize >= 0 || iface.OutDiscards >= 0 {
						iface.HasErrorData = true
					}
				}

			case strings.Contains(currentSection, "Single-Col") && strings.Contains(currentSection, "Multi-Col"):
				// Section 2: Single-Col, Multi-Col, Late-Col, Exces-Col, Carri-Sen, Runts
				if len(valueFields) >= 6 {
					iface.SingleCol = parseCounterValue(valueFields[0])
					iface.MultiCol = parseCounterValue(valueFields[1])
					iface.LateCol = parseCounterValue(valueFields[2])
					iface.ExcesCol = parseCounterValue(valueFields[3])
					iface.CarriSen = parseCounterValue(valueFields[4])
					iface.Runts = parseCounterValue(valueFields[5])
					if iface.SingleCol >= 0 || iface.MultiCol >= 0 || iface.LateCol >= 0 ||
						iface.ExcesCol >= 0 || iface.CarriSen >= 0 || iface.Runts >= 0 {
						iface.HasErrorData = true
					}
				}

			case strings.Contains(currentSection, "Giants") && strings.Contains(currentSection, "SQETest-Err"):
				// Section 3: Giants, SQETest-Err, Deferred-Tx, IntMacTx-Er, IntMacRx-Er, Symbol-Err
				if len(valueFields) >= 6 {
					iface.Giants = parseCounterValue(valueFields[0])
					iface.SQETestErr = parseCounterValue(valueFields[1])
					iface.DeferredTx = parseCounterValue(valueFields[2])
					iface.IntMacTxEr = parseCounterValue(valueFields[3])
					iface.IntMacRxEr = parseCounterValue(valueFields[4])
					iface.SymbolErr = parseCounterValue(valueFields[5])
					if iface.Giants >= 0 || iface.SQETestErr >= 0 || iface.DeferredTx >= 0 ||
						iface.IntMacTxEr >= 0 || iface.IntMacRxEr >= 0 || iface.SymbolErr >= 0 {
						iface.HasErrorData = true
					}
				}

			case strings.Contains(currentSection, "InDiscards"):
				// Section 4: InDiscards
				if len(valueFields) >= 1 {
					iface.InDiscards = parseCounterValue(valueFields[0])
					if iface.InDiscards >= 0 {
						iface.HasErrorData = true
					}
				}

			case strings.Contains(currentSection, "Stomped-CRC"):
				// Section 5: Stomped-CRC
				if len(valueFields) >= 1 {
					iface.StompedCRC = parseCounterValue(valueFields[0])
					if iface.StompedCRC >= 0 {
						iface.HasErrorData = true
					}
				}
			}
		}
	}

	// Convert map to slice and filter out interfaces with no data
	for _, ifaceData := range interfaceMap {
		// Only include interfaces that have at least some error counter data
		if ifaceData.HasErrorData {
			entry := StandardizedEntry{
				DataType:  "cisco_nexus_interface_error_counters",
				Timestamp: timestamp.Format(time.RFC3339),
				Date:      timestamp.Format("2006-01-02"),
				Message:   *ifaceData,
			}
			interfaces = append(interfaces, entry)
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

// findInterfaceErrorCountersCommand finds the interface-error-counter command in the commands.json
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
		fmt.Println("Cisco Nexus Interface Error Counters Parser")
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
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./interface_counters_error_parser -input show-interface-counter-errors.txt -output output.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./interface_counters_error_parser -commands commands.json -output output.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./interface_counters_error_parser -input show-interface-counter-errors.txt")
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

		command, err := findInterfaceErrorCountersCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding interface error counters command: %v\n", err)
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

	// Parse the interface error counters data
	fmt.Fprintf(os.Stderr, "Parsing interface error counters data...\n")
	interfaces := parseInterfaceErrorCounters(content)
	fmt.Fprintf(os.Stderr, "Found %d interfaces with error counter data\n", len(interfaces))

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
		fmt.Fprintf(os.Stderr, "Interface error counters data written to %s\n", *outputFile)
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
	return "Parses 'show interface counters errors' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseInterfaceErrorCounters(content), nil
}
