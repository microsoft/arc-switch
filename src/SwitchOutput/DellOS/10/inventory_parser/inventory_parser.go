package inventory_parser

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// StandardizedEntry represents the standardized JSON structure
type StandardizedEntry struct {
	DataType  string        `json:"data_type"`
	Timestamp string        `json:"timestamp"`
	Date      string        `json:"date"`
	Message   InventoryData `json:"message"`
}

// InventoryData represents the inventory data within the message field
type InventoryData struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	ProductID     string `json:"product_id"`
	VersionID     string `json:"version_id"`
	SerialNumber  string `json:"serial_number"`
	ComponentType string `json:"component_type"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// determineComponentType determines the component type based on the name
func determineComponentType(name string) string {
	nameLower := strings.ToLower(name)

	switch {
	case strings.Contains(nameLower, "chassis"):
		return "chassis"
	case strings.Contains(nameLower, "unit"):
		return "unit"
	case strings.Contains(nameLower, "power supply") || strings.Contains(nameLower, "psu"):
		return "power_supply"
	case strings.Contains(nameLower, "fan"):
		return "fan"
	case strings.Contains(nameLower, "ethernet") || strings.Contains(nameLower, "sfp"):
		return "transceiver"
	case strings.Contains(nameLower, "module"):
		return "module"
	default:
		return "unknown"
	}
}

// cleanQuotes removes surrounding quotes from a string
func cleanQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// parseInventory parses the show inventory output for Dell OS10
func parseInventory(content string) []StandardizedEntry {
	var inventory []StandardizedEntry
	lines := strings.Split(content, "\n")

	timestamp := time.Now()

	// Dell OS10 show inventory format:
	// NAME: "Chassis", DESCR: "Dell EMC Networking S4148-ON"
	// PID: S4148-ON          , VID: 01 , SN: ABC1234567
	//
	// NAME: "Unit 1", DESCR: "Dell EMC Networking S4148-ON"
	// PID: S4148-ON          , VID: 01 , SN: ABC1234567
	//
	// NAME: "Power Supply 1", DESCR: "Dell EMC AC Power Supply"
	// PID: DPS-550AB-39 A    , VID: 01 , SN: XYZ9876543

	nameRegex := regexp.MustCompile(`NAME:\s*"?([^"]*)"?\s*,\s*DESCR:\s*"?([^"]*)"?\s*$`)
	pidRegex := regexp.MustCompile(`PID:\s*([^,]*)\s*,\s*VID:\s*([^,]*)\s*,\s*SN:\s*(.*)$`)

	var currentItem *InventoryData

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			// Save current item if exists
			if currentItem != nil {
				entry := StandardizedEntry{
					DataType:  "dell_os10_inventory",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentItem,
				}
				inventory = append(inventory, entry)
				currentItem = nil
			}
			continue
		}

		// Skip the command line
		if strings.Contains(line, "show inventory") {
			continue
		}

		// Parse NAME and DESCR line
		if nameMatch := nameRegex.FindStringSubmatch(line); nameMatch != nil {
			// Save previous item if exists
			if currentItem != nil {
				entry := StandardizedEntry{
					DataType:  "dell_os10_inventory",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentItem,
				}
				inventory = append(inventory, entry)
			}

			// Start new item
			name := cleanQuotes(nameMatch[1])
			description := cleanQuotes(nameMatch[2])

			// Clean up names
			name = strings.TrimSuffix(name, ",")

			currentItem = &InventoryData{
				Name:          name,
				Description:   description,
				ComponentType: determineComponentType(name),
			}
			continue
		}

		// Parse PID, VID, SN line
		if currentItem != nil {
			if pidMatch := pidRegex.FindStringSubmatch(line); pidMatch != nil {
				currentItem.ProductID = strings.TrimSpace(pidMatch[1])
				currentItem.VersionID = strings.TrimSpace(pidMatch[2])
				currentItem.SerialNumber = strings.TrimSpace(pidMatch[3])

				// Add the completed item
				entry := StandardizedEntry{
					DataType:  "dell_os10_inventory",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentItem,
				}
				inventory = append(inventory, entry)
				currentItem = nil
				continue
			}
		}
	}

	// Save last item if exists
	if currentItem != nil {
		entry := StandardizedEntry{
			DataType:  "dell_os10_inventory",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentItem,
		}
		inventory = append(inventory, entry)
	}

	return inventory
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
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading commands file: %v", err)
	}

	var config CommandConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing commands file: %v", err)
	}

	return &config, nil
}

// findInventoryCommand finds the inventory command
func findInventoryCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "inventory" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("inventory command not found in commands file")
}

func main() {
	inputFile := flag.String("input", "", "Input file containing 'show inventory' output")
	outputFile := flag.String("output", "", "Output file to write JSON results (default: stdout)")
	commandsFile := flag.String("commands", "", "Path to JSON file containing CLI commands")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Inventory Parser")
		fmt.Println("Parses 'show inventory' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  inventory_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show inventory' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		return
	}

	var inputData string

	if *inputFile != "" {
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
		inputData = string(data)
	} else if *commandsFile != "" {
		config, err := loadCommandsFromFile(*commandsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading commands file: %v\n", err)
			os.Exit(1)
		}

		command, err := findInventoryCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		output, err := runCommand(command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running command: %v\n", err)
			os.Exit(1)
		}
		inputData = output
	} else {
		fmt.Fprintln(os.Stderr, "Error: You must specify either -input or -commands.")
		os.Exit(1)
	}

	entries := parseInventory(inputData)

	var output *os.File
	var err error
	if *outputFile == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer output.Close()
	}

	encoder := json.NewEncoder(output)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding entry: %v\n", err)
			os.Exit(1)
		}
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show inventory' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseInventory(string(input)), nil
}
