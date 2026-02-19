package main

import (
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
	DataType  string     `json:"data_type"`
	Timestamp string     `json:"timestamp"`
	Date      string     `json:"date"`
	Message   UptimeData `json:"message"`
}

// UptimeData represents parsed show uptime output
type UptimeData struct {
	RawUptime    string `json:"raw_uptime"`
	Weeks        int    `json:"weeks"`
	Days         int    `json:"days"`
	Hours        int    `json:"hours"`
	Minutes      int    `json:"minutes"`
	Seconds      int    `json:"seconds"`
	TotalSeconds int64  `json:"total_seconds"`
}

type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseSystemUptime parses Dell OS10 show uptime output
// Example: "9 weeks 3 days 01:24:11"
func parseSystemUptime(content string) ([]StandardizedEntry, error) {
	raw := strings.TrimSpace(content)
	timestamp := time.Now().UTC()
	data := UptimeData{RawUptime: raw}

	weeksRegex := regexp.MustCompile(`(\d+)\s+weeks?`)
	if match := weeksRegex.FindStringSubmatch(raw); match != nil {
		data.Weeks, _ = strconv.Atoi(match[1])
	}

	daysRegex := regexp.MustCompile(`(\d+)\s+days?`)
	if match := daysRegex.FindStringSubmatch(raw); match != nil {
		data.Days, _ = strconv.Atoi(match[1])
	}

	hmsRegex := regexp.MustCompile(`(\d+):(\d+):(\d+)`)
	if match := hmsRegex.FindStringSubmatch(raw); match != nil {
		data.Hours, _ = strconv.Atoi(match[1])
		data.Minutes, _ = strconv.Atoi(match[2])
		data.Seconds, _ = strconv.Atoi(match[3])
	}

	data.TotalSeconds = int64(data.Weeks)*7*24*3600 +
		int64(data.Days)*24*3600 +
		int64(data.Hours)*3600 +
		int64(data.Minutes)*60 +
		int64(data.Seconds)

	entry := StandardizedEntry{
		DataType:  "dell_os10_system_uptime",
		Timestamp: timestamp.Format(time.RFC3339),
		Date:      timestamp.Format("2006-01-02"),
		Message:   data,
	}
	return []StandardizedEntry{entry}, nil
}

func runCommand(command string) (string, error) {
	cmd := exec.Command("/opt/dell/os10/bin/clish", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("clish error: %v, output: %s", err, string(output))
	}
	return string(output), nil
}

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

func findCommand(config *CommandConfig, name string) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == name {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("%s command not found in commands file", name)
}

func main() {
	inputFile := flag.String("input", "", "Input file containing 'show uptime' output")
	outputFile := flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	commandsFile := flag.String("commands", "", "Commands JSON file")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 System Uptime Parser")
		fmt.Println("Parses 'show uptime' output and converts to JSON format.")
		fmt.Println("\nOptions:")
		fmt.Println("  -input <file>     Input file")
		fmt.Println("  -output <file>    Output file (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file")
		fmt.Println("  -help             Show this help message")
		return
	}

	var inputData string
	var err error

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
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		command, err := findCommand(config, "system-uptime")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		inputData, err = runCommand(command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Error: You must specify either -input or -commands.")
		os.Exit(1)
	}

	entries, err := parseSystemUptime(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var output *os.File
	if *outputFile == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer output.Close()
	}

	encoder := json.NewEncoder(output)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding: %v\n", err)
			os.Exit(1)
		}
	}
}

type UnifiedParser struct{}

func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show uptime' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseSystemUptime(string(input))
}
