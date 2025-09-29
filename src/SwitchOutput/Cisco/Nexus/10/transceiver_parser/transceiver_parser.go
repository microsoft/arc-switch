package transceiver_parser

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
	DataType  string          `json:"data_type"`  // Always "cisco_nexus_transceiver"
	Timestamp string          `json:"timestamp"`  // ISO 8601 timestamp
	Date      string          `json:"date"`       // Date in YYYY-MM-DD format
	Message   TransceiverData `json:"message"`    // Transceiver-specific data
}

// TransceiverData represents the transceiver data within the message field
type TransceiverData struct {
	InterfaceName      string           `json:"interface_name"`       // Interface name (e.g., Ethernet1/1)
	TransceiverPresent bool             `json:"transceiver_present"`  // Whether transceiver is present
	Type               string           `json:"type,omitempty"`       // Transceiver type (e.g., SFP-H25GB-CU3M, 10Gbase-SR)
	Manufacturer       string           `json:"manufacturer,omitempty"` // Name/manufacturer
	PartNumber         string           `json:"part_number,omitempty"`
	Revision           string           `json:"revision,omitempty"`
	SerialNumber       string           `json:"serial_number,omitempty"`
	NominalBitrate     int              `json:"nominal_bitrate,omitempty"` // In MBit/sec
	LinkLength         string           `json:"link_length,omitempty"`     // Link length description
	CableType          string           `json:"cable_type,omitempty"`
	CiscoID            string           `json:"cisco_id,omitempty"`
	CiscoExtendedID    string           `json:"cisco_extended_id,omitempty"`
	CiscoPartNumber    string           `json:"cisco_part_number,omitempty"`
	CiscoProductID     string           `json:"cisco_product_id,omitempty"`
	CiscoVersionID     string           `json:"cisco_version_id,omitempty"`
	DOMSupported       bool             `json:"dom_supported"`
	DOMData            *DOMDiagnostics  `json:"dom_data,omitempty"`
}

// DOMDiagnostics represents Digital Optical Monitoring data
type DOMDiagnostics struct {
	Temperature      *DOMParameter `json:"temperature,omitempty"`
	Voltage          *DOMParameter `json:"voltage,omitempty"`
	Current          *DOMParameter `json:"current,omitempty"`
	TxPower          *DOMParameter `json:"tx_power,omitempty"`
	RxPower          *DOMParameter `json:"rx_power,omitempty"`
	TransmitFaultCount int         `json:"transmit_fault_count"`
}

// DOMParameter represents a single DOM measurement with thresholds
type DOMParameter struct {
	CurrentValue  float64 `json:"current_value"`
	Unit          string  `json:"unit"`
	AlarmHigh     float64 `json:"alarm_high"`
	AlarmLow      float64 `json:"alarm_low"`
	WarningHigh   float64 `json:"warning_high"`
	WarningLow    float64 `json:"warning_low"`
	Status        string  `json:"status,omitempty"` // normal, high-alarm, low-alarm, high-warning, low-warning
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseFloatValue extracts float value and unit from strings like "34.22 C" or "3.26 V"
func parseFloatValue(value string) (float64, string) {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) >= 1 {
		val, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, ""
		}
		unit := ""
		if len(parts) >= 2 {
			unit = parts[1]
		}
		return val, unit
	}
	return 0, ""
}

// parseBitrate extracts numeric bitrate from strings like "25500 MBit/sec"
func parseBitrate(value string) int {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) >= 1 {
		val, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0
		}
		return val
	}
	return 0
}

// determineStatus checks if value is within thresholds and returns status
func determineStatus(value, alarmHigh, alarmLow, warningHigh, warningLow float64) string {
	if value >= alarmHigh {
		return "high-alarm"
	}
	if value <= alarmLow {
		return "low-alarm"
	}
	if value >= warningHigh {
		return "high-warning"
	}
	if value <= warningLow {
		return "low-warning"
	}
	return "normal"
}

// parseTransceivers parses the show interface transceiver details output
func parseTransceivers(content string) []StandardizedEntry {
	var transceivers []StandardizedEntry
	lines := strings.Split(content, "\n")
	
	timestamp := time.Now()
	
	// Regular expressions for parsing
	interfaceRegex := regexp.MustCompile(`^(Ethernet\d+/\d+)$`)
	presentRegex := regexp.MustCompile(`transceiver is (present|not present)`)
	typeRegex := regexp.MustCompile(`^\s*type is (.+)$`)
	nameRegex := regexp.MustCompile(`^\s*name is (.+)$`)
	partNumberRegex := regexp.MustCompile(`^\s*part number is (.+)$`)
	revisionRegex := regexp.MustCompile(`^\s*revision is (.+)$`)
	serialNumberRegex := regexp.MustCompile(`^\s*serial number is (.+)$`)
	bitrateRegex := regexp.MustCompile(`^\s*nominal bitrate is (.+)$`)
	linkLengthRegex := regexp.MustCompile(`^\s*Link length supported for (.+)$`)
	cableTypeRegex := regexp.MustCompile(`^\s*cable type is (.+)$`)
	ciscoIDRegex := regexp.MustCompile(`^\s*cisco id is (.+)$`)
	ciscoExtIDRegex := regexp.MustCompile(`^\s*cisco extended id number is (.+)$`)
	ciscoPartRegex := regexp.MustCompile(`^\s*cisco part number is (.+)$`)
	ciscoProductRegex := regexp.MustCompile(`^\s*cisco product id is (.+)$`)
	ciscoVersionRegex := regexp.MustCompile(`^\s*cisco version id is (.+)$`)
	domSupportRegex := regexp.MustCompile(`DOM is (not )?supported`)
	domHeaderRegex := regexp.MustCompile(`SFP Detail Diagnostics Information`)
	transmitFaultRegex := regexp.MustCompile(`Transmit Fault Count = (\d+)`)
	
	var currentTransceiver *TransceiverData
	var inDOMSection bool
	var domData *DOMDiagnostics
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		// Skip the command line
		if strings.Contains(line, "show interface transceiver") {
			continue
		}
		
		// Check for interface name
		if interfaceMatch := interfaceRegex.FindStringSubmatch(strings.TrimSpace(line)); interfaceMatch != nil {
			// Save previous transceiver if exists
			if currentTransceiver != nil {
				if inDOMSection && domData != nil {
					currentTransceiver.DOMData = domData
				}
				entry := StandardizedEntry{
					DataType:  "cisco_nexus_transceiver",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentTransceiver,
				}
				transceivers = append(transceivers, entry)
			}
			
			// Start new transceiver
			currentTransceiver = &TransceiverData{
				InterfaceName: interfaceMatch[1],
				DOMSupported:  false,
			}
			inDOMSection = false
			domData = nil
			continue
		}
		
		if currentTransceiver == nil {
			continue
		}
		
		// Check if transceiver is present
		if presentMatch := presentRegex.FindStringSubmatch(line); presentMatch != nil {
			currentTransceiver.TransceiverPresent = (presentMatch[1] == "present")
			continue
		}
		
		// Parse transceiver details only if present
		if currentTransceiver.TransceiverPresent {
			// Type
			if typeMatch := typeRegex.FindStringSubmatch(line); typeMatch != nil {
				currentTransceiver.Type = strings.TrimSpace(typeMatch[1])
				continue
			}
			
			// Name/Manufacturer
			if nameMatch := nameRegex.FindStringSubmatch(line); nameMatch != nil {
				currentTransceiver.Manufacturer = strings.TrimSpace(nameMatch[1])
				continue
			}
			
			// Part Number
			if partMatch := partNumberRegex.FindStringSubmatch(line); partMatch != nil {
				currentTransceiver.PartNumber = strings.TrimSpace(partMatch[1])
				continue
			}
			
			// Revision
			if revMatch := revisionRegex.FindStringSubmatch(line); revMatch != nil {
				currentTransceiver.Revision = strings.TrimSpace(revMatch[1])
				continue
			}
			
			// Serial Number
			if serialMatch := serialNumberRegex.FindStringSubmatch(line); serialMatch != nil {
				currentTransceiver.SerialNumber = strings.TrimSpace(serialMatch[1])
				continue
			}
			
			// Nominal Bitrate
			if bitrateMatch := bitrateRegex.FindStringSubmatch(line); bitrateMatch != nil {
				currentTransceiver.NominalBitrate = parseBitrate(bitrateMatch[1])
				continue
			}
			
			// Link Length
			if linkMatch := linkLengthRegex.FindStringSubmatch(line); linkMatch != nil {
				currentTransceiver.LinkLength = strings.TrimSpace(linkMatch[1])
				continue
			}
			
			// Cable Type
			if cableMatch := cableTypeRegex.FindStringSubmatch(line); cableMatch != nil {
				currentTransceiver.CableType = strings.TrimSpace(cableMatch[1])
				continue
			}
			
			// Cisco IDs
			if ciscoIDMatch := ciscoIDRegex.FindStringSubmatch(line); ciscoIDMatch != nil {
				currentTransceiver.CiscoID = strings.TrimSpace(ciscoIDMatch[1])
				continue
			}
			
			if ciscoExtMatch := ciscoExtIDRegex.FindStringSubmatch(line); ciscoExtMatch != nil {
				currentTransceiver.CiscoExtendedID = strings.TrimSpace(ciscoExtMatch[1])
				continue
			}
			
			if ciscoPartMatch := ciscoPartRegex.FindStringSubmatch(line); ciscoPartMatch != nil {
				currentTransceiver.CiscoPartNumber = strings.TrimSpace(ciscoPartMatch[1])
				continue
			}
			
			if ciscoProductMatch := ciscoProductRegex.FindStringSubmatch(line); ciscoProductMatch != nil {
				currentTransceiver.CiscoProductID = strings.TrimSpace(ciscoProductMatch[1])
				continue
			}
			
			if ciscoVersionMatch := ciscoVersionRegex.FindStringSubmatch(line); ciscoVersionMatch != nil {
				currentTransceiver.CiscoVersionID = strings.TrimSpace(ciscoVersionMatch[1])
				continue
			}
			
			// DOM Support
			if domMatch := domSupportRegex.FindStringSubmatch(line); domMatch != nil {
				if domMatch[1] == "not " {
					currentTransceiver.DOMSupported = false
				}
				continue
			}

			// DOM Data Section - presence of this section means DOM is supported
			if domHeaderRegex.MatchString(line) {
				currentTransceiver.DOMSupported = true  // Set to true when DOM data is present
				inDOMSection = true
				domData = &DOMDiagnostics{}
				continue
			}
			
			// DOM Data Section
			if domHeaderRegex.MatchString(line) {
				inDOMSection = true
				domData = &DOMDiagnostics{}
				continue
			}
			
			// Parse DOM data
			if inDOMSection && domData != nil {
				// Temperature line
				if strings.Contains(line, "Temperature") && !strings.Contains(line, "----") {
					fields := strings.Fields(line)
					if len(fields) >= 11 {
						temp := &DOMParameter{}
						temp.CurrentValue, temp.Unit = parseFloatValue(fields[1] + " " + fields[2])
						temp.AlarmHigh, _ = parseFloatValue(fields[3] + " " + fields[4])
						temp.AlarmLow, _ = parseFloatValue(fields[5] + " " + fields[6])
						temp.WarningHigh, _ = parseFloatValue(fields[7] + " " + fields[8])
						temp.WarningLow, _ = parseFloatValue(fields[9] + " " + fields[10])
						temp.Status = determineStatus(temp.CurrentValue, temp.AlarmHigh, temp.AlarmLow, temp.WarningHigh, temp.WarningLow)
						domData.Temperature = temp
					}
				}
				
				// Voltage line
				if strings.Contains(line, "Voltage") && !strings.Contains(line, "----") {
					fields := strings.Fields(line)
					if len(fields) >= 11 {
						volt := &DOMParameter{}
						volt.CurrentValue, volt.Unit = parseFloatValue(fields[1] + " " + fields[2])
						volt.AlarmHigh, _ = parseFloatValue(fields[3] + " " + fields[4])
						volt.AlarmLow, _ = parseFloatValue(fields[5] + " " + fields[6])
						volt.WarningHigh, _ = parseFloatValue(fields[7] + " " + fields[8])
						volt.WarningLow, _ = parseFloatValue(fields[9] + " " + fields[10])
						volt.Status = determineStatus(volt.CurrentValue, volt.AlarmHigh, volt.AlarmLow, volt.WarningHigh, volt.WarningLow)
						domData.Voltage = volt
					}
				}
				
				// Current line
				if strings.Contains(line, "Current") && !strings.Contains(line, "----") && !strings.Contains(line, "Measurement") {
					fields := strings.Fields(line)
					if len(fields) >= 11 {
						curr := &DOMParameter{}
						curr.CurrentValue, curr.Unit = parseFloatValue(fields[1] + " " + fields[2])
						curr.AlarmHigh, _ = parseFloatValue(fields[3] + " " + fields[4])
						curr.AlarmLow, _ = parseFloatValue(fields[5] + " " + fields[6])
						curr.WarningHigh, _ = parseFloatValue(fields[7] + " " + fields[8])
						curr.WarningLow, _ = parseFloatValue(fields[9] + " " + fields[10])
						curr.Status = determineStatus(curr.CurrentValue, curr.AlarmHigh, curr.AlarmLow, curr.WarningHigh, curr.WarningLow)
						domData.Current = curr
					}
				}
				
				// Tx Power line
				if strings.Contains(line, "Tx Power") {
					fields := strings.Fields(line)
					if len(fields) >= 12 {
						tx := &DOMParameter{}
						tx.CurrentValue, tx.Unit = parseFloatValue(fields[2] + " " + fields[3])
						tx.AlarmHigh, _ = parseFloatValue(fields[4] + " " + fields[5])
						tx.AlarmLow, _ = parseFloatValue(fields[6] + " " + fields[7])
						tx.WarningHigh, _ = parseFloatValue(fields[8] + " " + fields[9])
						tx.WarningLow, _ = parseFloatValue(fields[10] + " " + fields[11])
						tx.Status = determineStatus(tx.CurrentValue, tx.AlarmHigh, tx.AlarmLow, tx.WarningHigh, tx.WarningLow)
						domData.TxPower = tx
					}
				}
				
				// Rx Power line
				if strings.Contains(line, "Rx Power") {
					fields := strings.Fields(line)
					if len(fields) >= 12 {
						rx := &DOMParameter{}
						rx.CurrentValue, rx.Unit = parseFloatValue(fields[2] + " " + fields[3])
						rx.AlarmHigh, _ = parseFloatValue(fields[4] + " " + fields[5])
						rx.AlarmLow, _ = parseFloatValue(fields[6] + " " + fields[7])
						rx.WarningHigh, _ = parseFloatValue(fields[8] + " " + fields[9])
						rx.WarningLow, _ = parseFloatValue(fields[10] + " " + fields[11])
						rx.Status = determineStatus(rx.CurrentValue, rx.AlarmHigh, rx.AlarmLow, rx.WarningHigh, rx.WarningLow)
						domData.RxPower = rx
					}
				}
				
				// Transmit Fault Count
				if faultMatch := transmitFaultRegex.FindStringSubmatch(line); faultMatch != nil {
					count, _ := strconv.Atoi(faultMatch[1])
					domData.TransmitFaultCount = count
				}
			}
		}
	}
	
	// Save last transceiver if exists
	if currentTransceiver != nil {
		if inDOMSection && domData != nil {
			currentTransceiver.DOMData = domData
		}
		entry := StandardizedEntry{
			DataType:  "cisco_nexus_transceiver",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentTransceiver,
		}
		transceivers = append(transceivers, entry)
	}
	
	return transceivers
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

// findTransceiverCommand finds the transceiver command in the commands.json
func findTransceiverCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "transceiver" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("transceiver command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show interface transceiver details' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Cisco Nexus Transceiver Parser")
		fmt.Println("Parses 'show interface transceiver details' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  transceiver_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show interface transceiver details' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./transceiver_parser -input show-transceiver.txt -output output.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./transceiver_parser -commands commands.json -output output.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./transceiver_parser -input show-transceiver.txt")
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

		command, err := findTransceiverCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding transceiver command: %v\n", err)
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

	// Parse the transceiver data
	fmt.Fprintf(os.Stderr, "Parsing transceiver data...\n")
	transceivers := parseTransceivers(content)
	fmt.Fprintf(os.Stderr, "Found %d transceivers\n", len(transceivers))

	// Output results as individual JSON objects (one per line)
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		for _, entry := range transceivers {
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
		fmt.Fprintf(os.Stderr, "Transceiver data written to %s\n", *outputFile)
	} else {
		for _, entry := range transceivers {
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
	return "Parses 'show interface transceiver' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseTransceivers(content), nil
}