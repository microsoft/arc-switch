package system_uptime_parser

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
	DataType  string           `json:"data_type"`
	Timestamp string           `json:"timestamp"`
	Date      string           `json:"date"`
	Message   SystemUptimeData `json:"message"`
}

// SystemUptimeData represents the system uptime data
type SystemUptimeData struct {
	UptimeDays       int    `json:"uptime_days"`
	UptimeHours      int    `json:"uptime_hours"`
	UptimeMinutes    int    `json:"uptime_minutes"`
	UptimeTotalHours int    `json:"uptime_total_hours"`
	Users            int    `json:"users"`
	LoadAvg1Min      string `json:"load_avg_1min"`
	LoadAvg5Min      string `json:"load_avg_5min"`
	LoadAvg15Min     string `json:"load_avg_15min"`
	CurrentTime      string `json:"current_time"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseSystemUptime parses the show uptime output for Dell OS10
func parseSystemUptime(content string) (*StandardizedEntry, error) {
	timestamp := time.Now()

	data := SystemUptimeData{}

	// Dell OS10 uptime format examples:
	// " 10:23:45 up 45 days, 3:12, 2 users, load average: 0.15, 0.10, 0.05"
	// " 10:23:45 up 3:12, 2 users, load average: 0.15, 0.10, 0.05"
	// " 10:23:45 up 45 min, 1 user, load average: 0.15, 0.10, 0.05"

	// Extract current time
	timeRegex := regexp.MustCompile(`^\s*(\d+:\d+:\d+)\s+up`)
	if match := timeRegex.FindStringSubmatch(content); match != nil {
		data.CurrentTime = match[1]
	}

	// Extract days if present
	daysRegex := regexp.MustCompile(`up\s+(\d+)\s+days?`)
	if match := daysRegex.FindStringSubmatch(content); match != nil {
		data.UptimeDays, _ = strconv.Atoi(match[1])
	}

	// Extract hours:minutes format (after days or standalone)
	hoursMinRegex := regexp.MustCompile(`(?:days?,?\s+)?(\d+):(\d+)`)
	if match := hoursMinRegex.FindStringSubmatch(content); match != nil {
		data.UptimeHours, _ = strconv.Atoi(match[1])
		data.UptimeMinutes, _ = strconv.Atoi(match[2])
	}

	// Extract minutes only format
	minOnlyRegex := regexp.MustCompile(`up\s+(\d+)\s+min`)
	if match := minOnlyRegex.FindStringSubmatch(content); match != nil {
		data.UptimeMinutes, _ = strconv.Atoi(match[1])
	}

	// Extract users
	usersRegex := regexp.MustCompile(`(\d+)\s+users?`)
	if match := usersRegex.FindStringSubmatch(content); match != nil {
		data.Users, _ = strconv.Atoi(match[1])
	}

	// Extract load average
	loadRegex := regexp.MustCompile(`load average:\s*([\d.]+),\s*([\d.]+),\s*([\d.]+)`)
	if match := loadRegex.FindStringSubmatch(content); match != nil {
		data.LoadAvg1Min = match[1]
		data.LoadAvg5Min = match[2]
		data.LoadAvg15Min = match[3]
	}

	// Calculate total hours
	data.UptimeTotalHours = data.UptimeDays*24 + data.UptimeHours

	// Validate we got some data
	if data.UptimeDays == 0 && data.UptimeHours == 0 && data.UptimeMinutes == 0 && data.LoadAvg1Min == "" {
		return nil, fmt.Errorf("could not parse uptime data from input")
	}

	entry := &StandardizedEntry{
		DataType:  "dell_os10_system_uptime",
		Timestamp: timestamp.Format(time.RFC3339),
		Date:      timestamp.Format("2006-01-02"),
		Message:   data,
	}

	return entry, nil
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

// findSystemUptimeCommand finds the system-uptime command
func findSystemUptimeCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "system-uptime" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("system-uptime command not found in commands file")
}

func main() {
	inputFile := flag.String("input", "", "Input file containing uptime output")
	outputFile := flag.String("output", "", "Output file to write JSON results (default: stdout)")
	commandsFile := flag.String("commands", "", "Path to JSON file containing CLI commands")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 System Uptime Parser")
		fmt.Println("Parses 'show uptime' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  system_uptime_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show uptime' output")
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

		command, err := findSystemUptimeCommand(config)
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

	entry, err := parseSystemUptime(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing system uptime: %v\n", err)
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
	if err := encoder.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding entry: %v\n", err)
		os.Exit(1)
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show uptime' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseSystemUptime(string(input))
}
