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
	DataType  string          `json:"data_type"`
	Timestamp string          `json:"timestamp"`
	Date      string          `json:"date"`
	Message   TransceiverData `json:"message"`
}

// TransceiverData represents the transceiver data
type TransceiverData struct {
	InterfaceName      string          `json:"interface_name"`
	TransceiverPresent bool            `json:"transceiver_present"`
	Type               string          `json:"type,omitempty"`
	Vendor             string          `json:"vendor,omitempty"`
	PartNumber         string          `json:"part_number,omitempty"`
	SerialNumber       string          `json:"serial_number,omitempty"`
	Revision           string          `json:"revision,omitempty"`
	Connector          string          `json:"connector,omitempty"`
	CableLength        string          `json:"cable_length,omitempty"`
	DOMSupported       bool            `json:"dom_supported"`
	DOMData            *DOMDiagnostics `json:"dom_data,omitempty"`
}

// DOMDiagnostics represents Digital Optical Monitoring data
type DOMDiagnostics struct {
	Temperature float64 `json:"temperature"`
	Voltage     float64 `json:"voltage"`
	TxBias      float64 `json:"tx_bias"`
	TxPower     float64 `json:"tx_power"`
	RxPower     float64 `json:"rx_power"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseFloat extracts float value from strings like "25.5C" or "-10.5dBm"
func parseFloat(value string) float64 {
	value = strings.TrimSpace(value)
	// Remove common suffixes
	value = strings.TrimSuffix(value, "C")
	value = strings.TrimSuffix(value, "V")
	value = strings.TrimSuffix(value, "mA")
	value = strings.TrimSuffix(value, "dBm")
	value = strings.TrimSuffix(value, "mW")
	value = strings.TrimSpace(value)

	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return result
}

// parseTransceivers parses the show interface transceiver output for Dell OS10
func parseTransceivers(content string) []StandardizedEntry {
	var transceivers []StandardizedEntry
	lines := strings.Split(content, "\n")
	timestamp := time.Now()

	// Regular expressions for parsing Dell OS10 transceiver output
	interfaceRegex := regexp.MustCompile(`^(ethernet\d+/\d+/\d+):?$`)
	presentRegex := regexp.MustCompile(`Transceiver is (present|not present)`)
	typeRegex := regexp.MustCompile(`^\s*Identifier:\s+(.+)$`)
	vendorRegex := regexp.MustCompile(`^\s*Vendor Name:\s+(.+)$`)
	partNumRegex := regexp.MustCompile(`^\s*Vendor PN:\s+(.+)$`)
	serialRegex := regexp.MustCompile(`^\s*Vendor SN:\s+(.+)$`)
	revisionRegex := regexp.MustCompile(`^\s*Vendor Rev:\s+(.+)$`)
	connectorRegex := regexp.MustCompile(`^\s*Connector:\s+(.+)$`)
	cableLengthRegex := regexp.MustCompile(`^\s*Length.*:\s+(.+)$`)
	tempRegex := regexp.MustCompile(`^\s*Temperature:\s+([\d.-]+)`)
	voltageRegex := regexp.MustCompile(`^\s*Voltage:\s+([\d.-]+)`)
	txBiasRegex := regexp.MustCompile(`^\s*(?:Tx ?Bias|TxBias|Current):\s+([\d.-]+)`)
	txPowerRegex := regexp.MustCompile(`^\s*(?:Tx ?Power|TxPower):\s+([\d.-]+)`)
	rxPowerRegex := regexp.MustCompile(`^\s*(?:Rx ?Power|RxPower):\s+([\d.-]+)`)

	var currentTransceiver *TransceiverData
	var domData *DOMDiagnostics

	for _, line := range lines {
		// Check for new interface
		if match := interfaceRegex.FindStringSubmatch(strings.ToLower(strings.TrimSpace(line))); match != nil {
			// Save previous transceiver
			if currentTransceiver != nil {
				if domData != nil && (domData.Temperature != 0 || domData.Voltage != 0) {
					currentTransceiver.DOMSupported = true
					currentTransceiver.DOMData = domData
				}
				entry := StandardizedEntry{
					DataType:  "dell_os10_transceiver",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentTransceiver,
				}
				transceivers = append(transceivers, entry)
			}

			currentTransceiver = &TransceiverData{
				InterfaceName: match[1],
			}
			domData = &DOMDiagnostics{}
			continue
		}

		if currentTransceiver == nil {
			continue
		}

		// Check transceiver presence
		if match := presentRegex.FindStringSubmatch(line); match != nil {
			currentTransceiver.TransceiverPresent = (match[1] == "present")
			continue
		}

		// Parse transceiver details
		if match := typeRegex.FindStringSubmatch(line); match != nil {
			currentTransceiver.Type = strings.TrimSpace(match[1])
			currentTransceiver.TransceiverPresent = true
			continue
		}

		if match := vendorRegex.FindStringSubmatch(line); match != nil {
			currentTransceiver.Vendor = strings.TrimSpace(match[1])
			continue
		}

		if match := partNumRegex.FindStringSubmatch(line); match != nil {
			currentTransceiver.PartNumber = strings.TrimSpace(match[1])
			continue
		}

		if match := serialRegex.FindStringSubmatch(line); match != nil {
			currentTransceiver.SerialNumber = strings.TrimSpace(match[1])
			continue
		}

		if match := revisionRegex.FindStringSubmatch(line); match != nil {
			currentTransceiver.Revision = strings.TrimSpace(match[1])
			continue
		}

		if match := connectorRegex.FindStringSubmatch(line); match != nil {
			currentTransceiver.Connector = strings.TrimSpace(match[1])
			continue
		}

		if match := cableLengthRegex.FindStringSubmatch(line); match != nil {
			currentTransceiver.CableLength = strings.TrimSpace(match[1])
			continue
		}

		// Parse DOM data
		if match := tempRegex.FindStringSubmatch(line); match != nil {
			domData.Temperature = parseFloat(match[1])
			continue
		}

		if match := voltageRegex.FindStringSubmatch(line); match != nil {
			domData.Voltage = parseFloat(match[1])
			continue
		}

		if match := txBiasRegex.FindStringSubmatch(line); match != nil {
			domData.TxBias = parseFloat(match[1])
			continue
		}

		if match := txPowerRegex.FindStringSubmatch(line); match != nil {
			domData.TxPower = parseFloat(match[1])
			continue
		}

		if match := rxPowerRegex.FindStringSubmatch(line); match != nil {
			domData.RxPower = parseFloat(match[1])
			continue
		}
	}

	// Save last transceiver
	if currentTransceiver != nil {
		if domData != nil && (domData.Temperature != 0 || domData.Voltage != 0) {
			currentTransceiver.DOMSupported = true
			currentTransceiver.DOMData = domData
		}
		entry := StandardizedEntry{
			DataType:  "dell_os10_transceiver",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentTransceiver,
		}
		transceivers = append(transceivers, entry)
	}

	return transceivers
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

// findTransceiverCommand finds the transceiver command
func findTransceiverCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "transceiver" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("transceiver command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show interface transceiver' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Transceiver Parser")
		fmt.Println("Parses 'show interface transceiver' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  transceiver_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show interface transceiver' output")
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
		content = strings.Join(lines, "\n")
	} else if *commandsFile != "" {
		config, err := loadCommandsFromFile(*commandsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading commands file: %v\n", err)
			os.Exit(1)
		}

		command, err := findTransceiverCommand(config)
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

	transceivers := parseTransceivers(content)

	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		for _, entry := range transceivers {
			jsonData, _ := json.Marshal(entry)
			file.Write(append(jsonData, '\n'))
		}
	} else {
		for _, entry := range transceivers {
			jsonData, _ := json.Marshal(entry)
			fmt.Println(string(jsonData))
		}
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show interface transceiver' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseTransceivers(string(input)), nil
}
