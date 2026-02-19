package main

import (
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
	DataType  string      `json:"data_type"`
	Timestamp string      `json:"timestamp"`
	Date      string      `json:"date"`
	Message   VersionData `json:"message"`
}

// VersionData represents parsed show version output
type VersionData struct {
	OSName       string `json:"os_name"`
	OSVersion    string `json:"os_version"`
	BuildVersion string `json:"build_version"`
	BuildTime    string `json:"build_time"`
	SystemType   string `json:"system_type"`
	Architecture string `json:"architecture"`
	UpTime       string `json:"up_time"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

// Command represents a single command entry
type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseVersion parses Dell OS10 show version output
func parseVersion(content string) ([]StandardizedEntry, error) {
	data := VersionData{}
	lines := strings.Split(content, "\n")
	timestamp := time.Now().UTC()

	kvRegex := regexp.MustCompile(`^(.+?):\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// First line is the OS name (e.g. "Dell SmartFabric OS10 Enterprise")
		if strings.HasPrefix(line, "Dell ") && data.OSName == "" {
			data.OSName = line
			continue
		}

		if match := kvRegex.FindStringSubmatch(line); match != nil {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			switch key {
			case "OS Version":
				data.OSVersion = value
			case "Build Version":
				data.BuildVersion = value
			case "Build Time":
				data.BuildTime = value
			case "System Type":
				data.SystemType = value
			case "Architecture":
				data.Architecture = value
			case "Up Time":
				data.UpTime = value
			}
		}
	}

	entry := StandardizedEntry{
		DataType:  "dell_os10_version",
		Timestamp: timestamp.Format(time.RFC3339),
		Date:      timestamp.Format("2006-01-02"),
		Message:   data,
	}
	return []StandardizedEntry{entry}, nil
}

// runCommand executes a command on the Dell OS10 switch using clish
func runCommand(command string) (string, error) {
	cmd := exec.Command("/opt/dell/os10/bin/clish", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("clish error: %v, output: %s", err, string(output))
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

// findCommand finds a named command in the config
func findCommand(config *CommandConfig, name string) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == name {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("%s command not found in commands file", name)
}

func main() {
	inputFile := flag.String("input", "", "Input file containing 'show version' output")
	outputFile := flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	commandsFile := flag.String("commands", "", "Commands JSON file (used when no input file is specified)")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Version Parser")
		fmt.Println("Parses 'show version' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  version_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show version' output")
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
		command, err := findCommand(config, "version")
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

	entries, err := parseVersion(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing version: %v\n", err)
		os.Exit(1)
	}

	var output *os.File
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
	return "Parses 'show version' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseVersion(string(input))
}
