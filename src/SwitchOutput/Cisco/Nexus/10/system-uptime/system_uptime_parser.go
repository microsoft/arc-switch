package system_uptime_parser

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
	DataType  string          `json:"data_type"`  // Always "cisco_nexus_system_uptime"
	Timestamp string          `json:"timestamp"`  // ISO 8601 timestamp
	Date      string          `json:"date"`       // Date in YYYY-MM-DD format
	Message   SystemUptimeData `json:"message"`   // System uptime-specific data
}

// SystemUptimeData represents the system uptime data within the message field
type SystemUptimeData struct {
	SystemStartTime      string `json:"system_start_time"`        // System start time
	SystemUptimeDays     string `json:"system_uptime_days"`       // System uptime days
	SystemUptimeHours    string `json:"system_uptime_hours"`      // System uptime hours
	SystemUptimeMinutes  string `json:"system_uptime_minutes"`    // System uptime minutes
	SystemUptimeSeconds  string `json:"system_uptime_seconds"`    // System uptime seconds
	SystemUptimeTotal    string `json:"system_uptime_total"`      // System uptime total in human-readable format
	KernelUptimeDays     string `json:"kernel_uptime_days"`       // Kernel uptime days
	KernelUptimeHours    string `json:"kernel_uptime_hours"`      // Kernel uptime hours
	KernelUptimeMinutes  string `json:"kernel_uptime_minutes"`    // Kernel uptime minutes
	KernelUptimeSeconds  string `json:"kernel_uptime_seconds"`    // Kernel uptime seconds
	KernelUptimeTotal    string `json:"kernel_uptime_total"`      // Kernel uptime total in human-readable format
}

// parseSystemUptime parses the show system uptime output and returns the standardized entry
func parseSystemUptime(input string) (*StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	
	// Get current timestamp
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02")
	
	var data SystemUptimeData
	
	// Try to parse as JSON first
	var jsonData map[string]interface{}
	jsonErr := json.Unmarshal([]byte(input), &jsonData)
	
	// Check if input is JSON format and successfully parsed
	if jsonErr == nil && jsonData["sys_st_time"] != nil {
		// Parse JSON format
			// Extract values from JSON
			if val, ok := jsonData["sys_st_time"].(string); ok {
				data.SystemStartTime = val
			}
			if val, ok := jsonData["sys_up_days"].(string); ok {
				data.SystemUptimeDays = val
			}
			if val, ok := jsonData["sys_up_hrs"].(string); ok {
				data.SystemUptimeHours = val
			}
			if val, ok := jsonData["sys_up_mins"].(string); ok {
				data.SystemUptimeMinutes = val
			}
			if val, ok := jsonData["sys_up_secs"].(string); ok {
				data.SystemUptimeSeconds = val
			}
			if val, ok := jsonData["kn_up_days"].(string); ok {
				data.KernelUptimeDays = val
			}
			if val, ok := jsonData["kn_up_hrs"].(string); ok {
				data.KernelUptimeHours = val
			}
			if val, ok := jsonData["kn_up_mins"].(string); ok {
				data.KernelUptimeMinutes = val
			}
			if val, ok := jsonData["kn_up_secs"].(string); ok {
				data.KernelUptimeSeconds = val
			}
			
		// Build human-readable format
		data.SystemUptimeTotal = fmt.Sprintf("%s days, %s hours, %s minutes, %s seconds",
			data.SystemUptimeDays, data.SystemUptimeHours, data.SystemUptimeMinutes, data.SystemUptimeSeconds)
		data.KernelUptimeTotal = fmt.Sprintf("%s days, %s hours, %s minutes, %s seconds",
			data.KernelUptimeDays, data.KernelUptimeHours, data.KernelUptimeMinutes, data.KernelUptimeSeconds)
	} else {
		// Parse text format
		systemStartTimeRegex := regexp.MustCompile(`System start time:\s+(.+)$`)
		systemUptimeRegex := regexp.MustCompile(`System uptime:\s+(\d+)\s+days?,\s+(\d+)\s+hours?,\s+(\d+)\s+minutes?,\s+(\d+)\s+seconds?`)
		kernelUptimeRegex := regexp.MustCompile(`Kernel uptime:\s+(\d+)\s+days?,\s+(\d+)\s+hours?,\s+(\d+)\s+minutes?,\s+(\d+)\s+seconds?`)
		
		for scanner.Scan() {
			line := scanner.Text()
			
			// Parse system start time
			if match := systemStartTimeRegex.FindStringSubmatch(line); match != nil {
				data.SystemStartTime = strings.TrimSpace(match[1])
			}
			
			// Parse system uptime
			if match := systemUptimeRegex.FindStringSubmatch(line); match != nil {
				data.SystemUptimeDays = match[1]
				data.SystemUptimeHours = match[2]
				data.SystemUptimeMinutes = match[3]
				data.SystemUptimeSeconds = match[4]
				data.SystemUptimeTotal = fmt.Sprintf("%s days, %s hours, %s minutes, %s seconds",
					data.SystemUptimeDays, data.SystemUptimeHours, data.SystemUptimeMinutes, data.SystemUptimeSeconds)
			}
			
			// Parse kernel uptime
			if match := kernelUptimeRegex.FindStringSubmatch(line); match != nil {
				data.KernelUptimeDays = match[1]
				data.KernelUptimeHours = match[2]
				data.KernelUptimeMinutes = match[3]
				data.KernelUptimeSeconds = match[4]
				data.KernelUptimeTotal = fmt.Sprintf("%s days, %s hours, %s minutes, %s seconds",
					data.KernelUptimeDays, data.KernelUptimeHours, data.KernelUptimeMinutes, data.KernelUptimeSeconds)
			}
		}
		
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	
	// Validate that we parsed some data
	if data.SystemStartTime == "" && data.SystemUptimeDays == "" {
		return nil, fmt.Errorf("could not parse system uptime data")
	}
	
	entry := &StandardizedEntry{
		DataType:  "cisco_nexus_system_uptime",
		Timestamp: timestamp,
		Date:      date,
		Message:   data,
	}
	
	return entry, nil
}

// runVsh runs the given command using the vsh CLI and returns its output as a string
func runVsh(command string) (string, error) {
	cmd := []string{"vsh", "-c", command}
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("vsh error: %v, output: %s", err, string(out))
	}
	return string(out), nil
}

func main() {
	// Define command line flags
	inputFile := flag.String("input", "", "Input file containing Cisco Nexus system uptime output")
	outputFile := flag.String("output", "", "Output file to write JSON results (default: stdout)")
	commandsFile := flag.String("commands", "", "Path to JSON file containing CLI commands")
	flag.Parse()

	if (*inputFile != "" && *commandsFile != "") || (*inputFile == "" && *commandsFile == "") {
		fmt.Fprintln(os.Stderr, "Error: You must specify exactly one of -input or -commands.")
		os.Exit(1)
	}

	var inputData string

	if *commandsFile != "" {
		// Read commands JSON file
		data, err := os.ReadFile(*commandsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading commands file: %v\n", err)
			os.Exit(1)
		}
		var cmdFile struct {
			Commands []struct {
				Name    string `json:"name"`
				Command string `json:"command"`
			} `json:"commands"`
		}
		if err := json.Unmarshal(data, &cmdFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing commands JSON: %v\n", err)
			os.Exit(1)
		}
		var uptimeCmd string
		for _, c := range cmdFile.Commands {
			if c.Name == "system-uptime" {
				uptimeCmd = c.Command
				break
			}
		}
		if uptimeCmd == "" {
			fmt.Fprintln(os.Stderr, "Error: No 'system-uptime' command found in commands JSON.")
			os.Exit(1)
		}
		// Run the command using vsh
		vshOut, err := runVsh(uptimeCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running vsh: %v\n", err)
			os.Exit(1)
		}
		inputData = vshOut
	} else if *inputFile != "" {
		// Read from file
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
		inputData = string(data)
	}
	
	// Parse the system uptime
	entry, err := parseSystemUptime(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing system uptime: %v\n", err)
		os.Exit(1)
	}
	
	// Output the results
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
	
	// Write as JSON
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding entry: %v\n", err)
		os.Exit(1)
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

// GetDescription returns the parser description
func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show system uptime' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseSystemUptime(content)
}
