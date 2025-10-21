package environment_temperature_parser

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
	DataType  string                  `json:"data_type"`  // Always "cisco_nexus_environment_temperature"
	Timestamp string                  `json:"timestamp"`  // ISO 8601 timestamp
	Date      string                  `json:"date"`       // Date in YYYY-MM-DD format
	Message   EnvironmentTemperatureData `json:"message"`    // Temperature-specific data
}

// EnvironmentTemperatureData represents the temperature data within the message field
type EnvironmentTemperatureData struct {
	Module          string `json:"module"`             // Module number
	Sensor          string `json:"sensor"`             // Sensor name (FRONT, BACK, CPU, etc.)
	MajorThreshold  string `json:"major_threshold"`    // Major threshold in Celsius
	MinorThreshold  string `json:"minor_threshold"`    // Minor threshold in Celsius
	CurrentTemp     string `json:"current_temp"`       // Current temperature in Celsius
	Status          string `json:"status"`             // Status (Ok, Alert, etc.)
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseTemperature parses the show environment temperature output
func parseTemperature(content string) []StandardizedEntry {
	var entries []StandardizedEntry
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	timestamp := time.Now()
	
	// Regular expression for parsing temperature table lines
	// Format: Module   Sensor        MajorThresh   MinorThres   CurTemp     Status
	// Example: 1        FRONT           80              70          28         Ok
	tempLineRegex := regexp.MustCompile(`^\s*(\d+)\s+(\S+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(.+?)\s*$`)
	
	inTable := false
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip command line
		if strings.Contains(line, "show environment temperature") {
			continue
		}
		
		// Skip header lines
		if strings.Contains(line, "Temperature:") {
			continue
		}
		
		// Detect table separator
		if strings.Contains(line, "----") {
			inTable = true
			continue
		}
		
		// Skip column headers
		if strings.Contains(line, "Module") && strings.Contains(line, "Sensor") {
			continue
		}
		if strings.Contains(line, "Celsius") {
			continue
		}
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			inTable = false
			continue
		}
		
		// Parse temperature data lines
		if inTable {
			if match := tempLineRegex.FindStringSubmatch(line); match != nil {
				tempData := EnvironmentTemperatureData{
					Module:         match[1],
					Sensor:         match[2],
					MajorThreshold: match[3],
					MinorThreshold: match[4],
					CurrentTemp:    match[5],
					Status:         strings.TrimSpace(match[6]),
				}
				
				entry := StandardizedEntry{
					DataType:  "cisco_nexus_environment_temperature",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   tempData,
				}
				
				entries = append(entries, entry)
			}
		}
	}
	
	return entries
}

// loadCommandsFromFile loads the commands.json file
func loadCommandsFromFile(filename string) (*CommandConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config CommandConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// findCommand finds a specific command in the commands.json
func findCommand(config *CommandConfig, commandName string) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == commandName {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("environment-temperature command not found in commands file")
}

func Main() {
	var inputFile = flag.String("input", "", "Input file containing 'show environment temperature' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Cisco Nexus Environment Temperature Parser")
		fmt.Println("Parses 'show environment temperature' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  environment_temperature_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show environment temperature' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./environment_temperature_parser -input show-environment-temperature.txt -output output.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./environment_temperature_parser -commands commands.json -output output.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./environment_temperature_parser -input show-environment-temperature.txt")
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
		// Load commands from file
		config, err := loadCommandsFromFile(*commandsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading commands file: %v\n", err)
			os.Exit(1)
		}

		// Find the environment-temperature command
		cmdStr, err := findCommand(config, "environment-temperature")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding command: %v\n", err)
			os.Exit(1)
		}

		// Execute the command using vsh
		cmd := exec.Command("vsh", "-c", cmdStr)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
			os.Exit(1)
		}

		content = string(output)
	} else {
		fmt.Fprintf(os.Stderr, "Error: Must specify either -input or -commands\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the temperature data
	entries := parseTemperature(content)

	// Output results
	if *outputFile != "" {
		// Write to file
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		for _, entry := range entries {
			if err := encoder.Encode(entry); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Fprintf(os.Stderr, "Successfully parsed %d temperature entries\n", len(entries))
		fmt.Fprintf(os.Stderr, "Output written to: %s\n", *outputFile)
	} else {
		// Write to stdout
		encoder := json.NewEncoder(os.Stdout)
		for _, entry := range entries {
			if err := encoder.Encode(entry); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

// GetDescription returns the parser description
func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show environment temperature' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseTemperature(content), nil
}
