package environment_temperature_parser

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
	Message   EnvironmentTemperatureData `json:"message"`
}

// EnvironmentTemperatureData represents the temperature data
type EnvironmentTemperatureData struct {
	Unit            string  `json:"unit"`
	Sensor          string  `json:"sensor"`
	CurrentTemp     float64 `json:"current_temp"`
	MinorThreshold  float64 `json:"minor_threshold"`
	MajorThreshold  float64 `json:"major_threshold"`
	Status          string  `json:"status"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseTemperature parses the show environment temperature output for Dell OS10
func parseTemperature(content string) []StandardizedEntry {
	var entries []StandardizedEntry
	scanner := bufio.NewScanner(strings.NewReader(content))
	timestamp := time.Now()

	// Dell OS10 format:
	// Unit  Sensor        Current   Minor     Major     Status
	// ----  ------        -------   -----     -----     ------
	// 1     CPU           45        70        80        Ok
	// 1     FRONT         28        55        65        Ok

	tempLineRegex := regexp.MustCompile(`^\s*(\S+)\s+(\S+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\S+)\s*$`)
	inTable := false

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect header
		if strings.Contains(line, "Unit") && strings.Contains(line, "Sensor") && strings.Contains(line, "Current") {
			inTable = true
			continue
		}

		// Skip separator lines
		if strings.Contains(line, "----") {
			continue
		}

		// Parse data lines
		if inTable {
			if match := tempLineRegex.FindStringSubmatch(line); match != nil {
				currentTemp, _ := strconv.ParseFloat(match[3], 64)
				minorThresh, _ := strconv.ParseFloat(match[4], 64)
				majorThresh, _ := strconv.ParseFloat(match[5], 64)

				entry := StandardizedEntry{
					DataType:  "dell_os10_environment_temperature",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message: EnvironmentTemperatureData{
						Unit:           match[1],
						Sensor:         match[2],
						CurrentTemp:    currentTemp,
						MinorThreshold: minorThresh,
						MajorThreshold: majorThresh,
						Status:         match[6],
					},
				}
				entries = append(entries, entry)
			}
		}
	}

	return entries
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

// findTemperatureCommand finds the environment-temperature command
func findTemperatureCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "environment-temperature" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("environment-temperature command not found in commands file")
}

func main() {
	inputFile := flag.String("input", "", "Input file containing 'show environment temperature' output")
	outputFile := flag.String("output", "", "Output file to write JSON results (default: stdout)")
	commandsFile := flag.String("commands", "", "Path to JSON file containing CLI commands")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Environment Temperature Parser")
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

		command, err := findTemperatureCommand(config)
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

	entries := parseTemperature(inputData)

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
	return "Parses 'show environment temperature' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseTemperature(string(input)), nil
}
