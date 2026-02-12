package system_resources_parser

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
	DataType  string              `json:"data_type"`
	Timestamp string              `json:"timestamp"`
	Date      string              `json:"date"`
	Message   SystemResourcesData `json:"message"`
}

// SystemResourcesData represents the system resources data
type SystemResourcesData struct {
	CPUUsageUser   float64 `json:"cpu_usage_user"`
	CPUUsageSystem float64 `json:"cpu_usage_system"`
	CPUUsageIdle   float64 `json:"cpu_usage_idle"`
	CPUUsageTotal  float64 `json:"cpu_usage_total"`
	MemoryTotal    int64   `json:"memory_total"`
	MemoryUsed     int64   `json:"memory_used"`
	MemoryFree     int64   `json:"memory_free"`
	MemoryBuffers  int64   `json:"memory_buffers"`
	MemoryCached   int64   `json:"memory_cached"`
	MemoryPercent  float64 `json:"memory_percent"`
	SwapTotal      int64   `json:"swap_total"`
	SwapUsed       int64   `json:"swap_used"`
	SwapFree       int64   `json:"swap_free"`
	LoadAvg1Min    string  `json:"load_avg_1min"`
	LoadAvg5Min    string  `json:"load_avg_5min"`
	LoadAvg15Min   string  `json:"load_avg_15min"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseSystemResources parses the show system-resources output for Dell OS10
func parseSystemResources(content string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	timestamp := time.Now()

	data := SystemResourcesData{}

	// Regular expressions for Dell OS10 system resources
	cpuRegex := regexp.MustCompile(`CPU\s+usage:\s*([\d.]+)%\s+user,\s*([\d.]+)%\s+system,\s*([\d.]+)%\s+idle`)
	memTotalRegex := regexp.MustCompile(`Mem:\s+(\d+)\s+total`)
	memUsedRegex := regexp.MustCompile(`(\d+)\s+used`)
	memFreeRegex := regexp.MustCompile(`(\d+)\s+free`)
	memBuffersRegex := regexp.MustCompile(`(\d+)\s+buffers`)
	memCachedRegex := regexp.MustCompile(`(\d+)\s+cached`)
	swapRegex := regexp.MustCompile(`Swap:\s+(\d+)\s+total,\s+(\d+)\s+used,\s+(\d+)\s+free`)
	loadAvgRegex := regexp.MustCompile(`load average:\s*([\d.]+),\s*([\d.]+),\s*([\d.]+)`)

	// Alternative format for "free -m" style output
	freeMemRegex := regexp.MustCompile(`^Mem:\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)`)
	freeSwapRegex := regexp.MustCompile(`^Swap:\s+(\d+)\s+(\d+)\s+(\d+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse CPU usage
		if match := cpuRegex.FindStringSubmatch(line); match != nil {
			data.CPUUsageUser, _ = strconv.ParseFloat(match[1], 64)
			data.CPUUsageSystem, _ = strconv.ParseFloat(match[2], 64)
			data.CPUUsageIdle, _ = strconv.ParseFloat(match[3], 64)
			data.CPUUsageTotal = data.CPUUsageUser + data.CPUUsageSystem
			continue
		}

		// Parse load average
		if match := loadAvgRegex.FindStringSubmatch(line); match != nil {
			data.LoadAvg1Min = match[1]
			data.LoadAvg5Min = match[2]
			data.LoadAvg15Min = match[3]
			continue
		}

		// Parse memory from "free -m" style output
		if match := freeMemRegex.FindStringSubmatch(line); match != nil {
			data.MemoryTotal, _ = strconv.ParseInt(match[1], 10, 64)
			data.MemoryUsed, _ = strconv.ParseInt(match[2], 10, 64)
			data.MemoryFree, _ = strconv.ParseInt(match[3], 10, 64)
			// match[4] is shared
			data.MemoryBuffers, _ = strconv.ParseInt(match[5], 10, 64)
			// match[6] is available
			if data.MemoryTotal > 0 {
				data.MemoryPercent = float64(data.MemoryUsed) / float64(data.MemoryTotal) * 100
			}
			continue
		}

		// Parse swap from "free -m" style output
		if match := freeSwapRegex.FindStringSubmatch(line); match != nil {
			data.SwapTotal, _ = strconv.ParseInt(match[1], 10, 64)
			data.SwapUsed, _ = strconv.ParseInt(match[2], 10, 64)
			data.SwapFree, _ = strconv.ParseInt(match[3], 10, 64)
			continue
		}

		// Parse memory total
		if match := memTotalRegex.FindStringSubmatch(line); match != nil {
			data.MemoryTotal, _ = strconv.ParseInt(match[1], 10, 64)
		}
		if match := memUsedRegex.FindStringSubmatch(line); match != nil {
			data.MemoryUsed, _ = strconv.ParseInt(match[1], 10, 64)
		}
		if match := memFreeRegex.FindStringSubmatch(line); match != nil {
			data.MemoryFree, _ = strconv.ParseInt(match[1], 10, 64)
		}
		if match := memBuffersRegex.FindStringSubmatch(line); match != nil {
			data.MemoryBuffers, _ = strconv.ParseInt(match[1], 10, 64)
		}
		if match := memCachedRegex.FindStringSubmatch(line); match != nil {
			data.MemoryCached, _ = strconv.ParseInt(match[1], 10, 64)
		}

		// Parse swap
		if match := swapRegex.FindStringSubmatch(line); match != nil {
			data.SwapTotal, _ = strconv.ParseInt(match[1], 10, 64)
			data.SwapUsed, _ = strconv.ParseInt(match[2], 10, 64)
			data.SwapFree, _ = strconv.ParseInt(match[3], 10, 64)
		}
	}

	// Calculate memory percent if not already set
	if data.MemoryPercent == 0 && data.MemoryTotal > 0 {
		data.MemoryPercent = float64(data.MemoryUsed) / float64(data.MemoryTotal) * 100
	}

	entry := StandardizedEntry{
		DataType:  "dell_os10_system_resources",
		Timestamp: timestamp.Format(time.RFC3339),
		Date:      timestamp.Format("2006-01-02"),
		Message:   data,
	}

	return []StandardizedEntry{entry}, nil
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

// findSystemResourcesCommand finds the system-resources command
func findSystemResourcesCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "system-resources" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("system-resources command not found in commands file")
}

func main() {
	inputFile := flag.String("input", "", "Input file containing system resources output")
	outputFile := flag.String("output", "", "Output file to write JSON results (default: stdout)")
	commandsFile := flag.String("commands", "", "Path to JSON file containing CLI commands")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 System Resources Parser")
		fmt.Println("Parses 'show system-resources' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  system_resources_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show system-resources' output")
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

		command, err := findSystemResourcesCommand(config)
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

	entries, err := parseSystemResources(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing system resources: %v\n", err)
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
	return "Parses 'show system-resources' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseSystemResources(string(input))
}
