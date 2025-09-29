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
	DataType  string        `json:"data_type"`  // Always "cisco_nexus_inventory"
	Timestamp string        `json:"timestamp"`  // ISO 8601 timestamp
	Date      string        `json:"date"`       // Date in YYYY-MM-DD format
	Message   InventoryData `json:"message"`    // Inventory-specific data
}

// InventoryData represents the inventory data within the message field
type InventoryData struct {
	Name         string `json:"name"`          // Component name (e.g., "Chassis", "Slot 1", "Ethernet1/1")
	Description  string `json:"description"`   // Component description
	ProductID    string `json:"product_id"`    // PID (Product ID)
	VersionID    string `json:"version_id"`    // VID (Version ID)
	SerialNumber string `json:"serial_number"` // SN (Serial Number)
	ComponentType string `json:"component_type"` // Type of component (chassis, slot, power_supply, fan, transceiver)
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
	case strings.Contains(nameLower, "slot"):
		return "slot"
	case strings.Contains(nameLower, "power supply"):
		return "power_supply"
	case strings.Contains(nameLower, "fan"):
		return "fan"
	case strings.Contains(nameLower, "ethernet"):
		return "transceiver"
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

// parseInventory parses the show inventory all output
func parseInventory(content string) []StandardizedEntry {
	var inventory []StandardizedEntry
	lines := strings.Split(content, "\n")
	
	timestamp := time.Now()
	
	// Regular expressions for parsing
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
					DataType:  "cisco_nexus_inventory",
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
					DataType:  "cisco_nexus_inventory",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentItem,
				}
				inventory = append(inventory, entry)
			}
			
			// Start new item
			name := cleanQuotes(nameMatch[1])
			description := cleanQuotes(nameMatch[2])
			
			// Clean up Ethernet names (remove comma if present)
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
					DataType:  "cisco_nexus_inventory",
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
			DataType:  "cisco_nexus_inventory",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentItem,
		}
		inventory = append(inventory, entry)
	}
	
	return inventory
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

// findInventoryCommand finds the inventory command in the commands.json
func findInventoryCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "inventory" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("inventory command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show inventory all' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Cisco Nexus Inventory Parser")
		fmt.Println("Parses 'show inventory all' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  inventory_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show inventory all' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./inventory_parser -input show-inventory.txt -output output.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./inventory_parser -commands commands.json -output output.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./inventory_parser -input show-inventory.txt")
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

		command, err := findInventoryCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding inventory command: %v\n", err)
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

	// Parse the inventory data
	fmt.Fprintf(os.Stderr, "Parsing inventory data...\n")
	inventory := parseInventory(content)
	fmt.Fprintf(os.Stderr, "Found %d inventory items\n", len(inventory))

	// Output results as individual JSON objects (one per line)
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		for _, entry := range inventory {
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
		fmt.Fprintf(os.Stderr, "Inventory data written to %s\n", *outputFile)
	} else {
		for _, entry := range inventory {
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
	return "Parses 'show inventory all' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseInventory(content), nil
}