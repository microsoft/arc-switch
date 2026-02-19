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
	DataType  string          `json:"data_type"`
	Timestamp string          `json:"timestamp"`
	Date      string          `json:"date"`
	Message   ProcessesCpuData `json:"message"`
}

// ProcessesCpuData represents parsed show processes cpu output
type ProcessesCpuData struct {
	UnitID         int           `json:"unit_id"`
	OverallCPU5Sec float64      `json:"overall_cpu_5sec_pct"`
	OverallCPU1Min float64      `json:"overall_cpu_1min_pct"`
	OverallCPU5Min float64      `json:"overall_cpu_5min_pct"`
	Processes      []ProcessInfo `json:"processes"`
}

// ProcessInfo represents a single process entry
type ProcessInfo struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	RuntimeSec int64   `json:"runtime_seconds"`
	CPU5Sec    float64 `json:"cpu_5sec_pct"`
	CPU1Min    float64 `json:"cpu_1min_pct"`
	CPU5Min    float64 `json:"cpu_5min_pct"`
}

type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseProcessesCpu parses Dell OS10 show processes cpu output
func parseProcessesCpu(content string) ([]StandardizedEntry, error) {
	data := ProcessesCpuData{}
	lines := strings.Split(content, "\n")
	timestamp := time.Now().UTC()

	unitRegex := regexp.MustCompile(`CPU Statistics of Unit (\d+)`)
	overallRegex := regexp.MustCompile(`^Overall\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)`)
	processRegex := regexp.MustCompile(`^(\d+)\s+(\S+)\s+(\d+)\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)`)

	inProcessTable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if match := unitRegex.FindStringSubmatch(trimmed); match != nil {
			data.UnitID, _ = strconv.Atoi(match[1])
			continue
		}

		if match := overallRegex.FindStringSubmatch(trimmed); match != nil {
			data.OverallCPU5Sec, _ = strconv.ParseFloat(match[1], 64)
			data.OverallCPU1Min, _ = strconv.ParseFloat(match[2], 64)
			data.OverallCPU5Min, _ = strconv.ParseFloat(match[3], 64)
			continue
		}

		if strings.HasPrefix(trimmed, "PID") && strings.Contains(trimmed, "Process") {
			inProcessTable = true
			continue
		}

		if inProcessTable {
			if match := processRegex.FindStringSubmatch(trimmed); match != nil {
				proc := ProcessInfo{
					Name: match[2],
				}
				proc.PID, _ = strconv.Atoi(match[1])
				proc.RuntimeSec, _ = strconv.ParseInt(match[3], 10, 64)
				proc.CPU5Sec, _ = strconv.ParseFloat(match[4], 64)
				proc.CPU1Min, _ = strconv.ParseFloat(match[5], 64)
				proc.CPU5Min, _ = strconv.ParseFloat(match[6], 64)
				data.Processes = append(data.Processes, proc)
			}
		}
	}

	entry := StandardizedEntry{
		DataType:  "dell_os10_processes_cpu",
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
	inputFile := flag.String("input", "", "Input file containing 'show processes cpu' output")
	outputFile := flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	commandsFile := flag.String("commands", "", "Commands JSON file")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Processes CPU Parser")
		fmt.Println("Parses 'show processes cpu' output and converts to JSON format.")
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
		command, err := findCommand(config, "processes-cpu")
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

	entries, err := parseProcessesCpu(inputData)
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
	return "Parses 'show processes cpu' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseProcessesCpu(string(input))
}
