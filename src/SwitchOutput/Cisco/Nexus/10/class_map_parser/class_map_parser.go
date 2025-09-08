package main

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
	DataType  string          `json:"data_type"`  // Always "cisco_nexus_class_map"
	Timestamp string          `json:"timestamp"`  // ISO 8601 timestamp
	Date      string          `json:"date"`       // Date in YYYY-MM-DD format
	Message   ClassMapData    `json:"message"`    // Class map-specific data
}

// ClassMapData represents the class map data within the message field
type ClassMapData struct {
	ClassName    string            `json:"class_name"`      // Class map name (e.g., RDMA, c-out-q3)
	ClassType    string            `json:"class_type"`      // Type (qos, queuing, network-qos)
	MatchType    string            `json:"match_type"`      // Match type (match-all, match-any)
	Description  string            `json:"description"`     // Optional description
	MatchRules   []MatchRule       `json:"match_rules"`     // List of match conditions
}

// MatchRule represents a single match condition
type MatchRule struct {
	MatchType  string `json:"match_type"`   // Type of match (cos, precedence, qos-group)
	MatchValue string `json:"match_value"`  // Value being matched
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseClassMaps parses the show class-map output
func parseClassMaps(content string) []StandardizedEntry {
	var classMaps []StandardizedEntry
	lines := strings.Split(content, "\n")
	
	timestamp := time.Now()
	
	// Regular expressions for parsing
	typeHeaderRegex := regexp.MustCompile(`^\s*Type\s+(\S+)\s+class-maps`)
	classMapRegex := regexp.MustCompile(`^\s*class-map\s+type\s+(\S+)\s+(\S+)\s+(.+)$`)
	descriptionRegex := regexp.MustCompile(`^\s*Description:\s+(.+)$`)
	matchRegex := regexp.MustCompile(`^\s*match\s+(\S+)\s+(.+)$`)
	
	var currentClassMap *ClassMapData
	var currentType string
	
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		
		// Skip empty lines and separator lines
		if strings.TrimSpace(line) == "" || strings.Contains(line, "====") {
			// Save current class map if exists
			if currentClassMap != nil {
				entry := StandardizedEntry{
					DataType:  "cisco_nexus_class_map",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentClassMap,
				}
				classMaps = append(classMaps, entry)
				currentClassMap = nil
			}
			continue
		}
		
		// Skip the command line
		if strings.Contains(line, "show class-map") {
			continue
		}
		
		// Detect type headers
		if typeMatch := typeHeaderRegex.FindStringSubmatch(line); typeMatch != nil {
			currentType = strings.ToLower(typeMatch[1])
			continue
		}
		
		// Parse class-map definition
		if classMapMatch := classMapRegex.FindStringSubmatch(line); classMapMatch != nil {
			// Save previous class map if exists
			if currentClassMap != nil {
				entry := StandardizedEntry{
					DataType:  "cisco_nexus_class_map",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentClassMap,
				}
				classMaps = append(classMaps, entry)
			}
			
			// Start new class map
			currentClassMap = &ClassMapData{
				ClassType:   classMapMatch[1],
				MatchType:   classMapMatch[2],
				ClassName:   classMapMatch[3],
				Description: "",
				MatchRules:  []MatchRule{},
			}
			
			// If currentType is empty, use the type from the class-map line
			if currentType == "" {
				currentClassMap.ClassType = classMapMatch[1]
			} else {
				currentClassMap.ClassType = currentType
			}
			continue
		}
		
		// Parse description
		if currentClassMap != nil {
			if descMatch := descriptionRegex.FindStringSubmatch(line); descMatch != nil {
				currentClassMap.Description = strings.TrimSpace(descMatch[1])
				continue
			}
			
			// Parse match rules
			if matchMatch := matchRegex.FindStringSubmatch(line); matchMatch != nil {
				rule := MatchRule{
					MatchType:  matchMatch[1],
					MatchValue: strings.TrimSpace(matchMatch[2]),
				}
				currentClassMap.MatchRules = append(currentClassMap.MatchRules, rule)
				continue
			}
		}
	}
	
	// Save last class map if exists
	if currentClassMap != nil {
		entry := StandardizedEntry{
			DataType:  "cisco_nexus_class_map",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentClassMap,
		}
		classMaps = append(classMaps, entry)
	}
	
	return classMaps
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

// findClassMapCommand finds the class-map command in the commands.json
func findClassMapCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "class-map" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("class-map command not found in commands file")
}

func main() {
	var inputFile = flag.String("input", "", "Input file containing 'show class-map' output")
	var outputFile = flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	var commandsFile = flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	var help = flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		fmt.Println("Cisco Nexus Class Map Parser")
		fmt.Println("Parses 'show class-map' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  class_map_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show class-map' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Parse from input file")
		fmt.Println("  ./class_map_parser -input show-class-map.txt -output output.json")
		fmt.Println("")
		fmt.Println("  # Get data directly from switch using commands.json")
		fmt.Println("  ./class_map_parser -commands commands.json -output output.json")
		fmt.Println("")
		fmt.Println("  # Parse from input file and output to stdout")
		fmt.Println("  ./class_map_parser -input show-class-map.txt")
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

		command, err := findClassMapCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding class-map command: %v\n", err)
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

	// Parse the class map data
	fmt.Fprintf(os.Stderr, "Parsing class map data...\n")
	classMaps := parseClassMaps(content)
	fmt.Fprintf(os.Stderr, "Found %d class maps\n", len(classMaps))

	// Output results as individual JSON objects (one per line)
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		for _, entry := range classMaps {
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
		fmt.Fprintf(os.Stderr, "Class map data written to %s\n", *outputFile)
	} else {
		for _, entry := range classMaps {
			jsonData, err := json.Marshal(entry)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling entry to JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(jsonData))
		}
	}
}