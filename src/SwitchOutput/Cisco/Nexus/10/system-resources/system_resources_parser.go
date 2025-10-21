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
	DataType  string              `json:"data_type"`  // Always "cisco_nexus_system_resources"
	Timestamp string              `json:"timestamp"`  // ISO 8601 timestamp
	Date      string              `json:"date"`       // Date in YYYY-MM-DD format
	Message   SystemResourcesData `json:"message"`    // System resources-specific data
}

// SystemResourcesData represents the system resources data within the message field
type SystemResourcesData struct {
	LoadAvg1Min         string    `json:"load_avg_1min"`
	LoadAvg5Min         string    `json:"load_avg_5min"`
	LoadAvg15Min        string    `json:"load_avg_15min"`
	ProcessesTotal      int       `json:"processes_total"`
	ProcessesRunning    int       `json:"processes_running"`
	CPUStateUser        string    `json:"cpu_state_user"`
	CPUStateKernel      string    `json:"cpu_state_kernel"`
	CPUStateIdle        string    `json:"cpu_state_idle"`
	CPUUsage            []CPUCore `json:"cpu_usage"`
	MemoryUsageTotal    int64     `json:"memory_usage_total"`
	MemoryUsageUsed     int64     `json:"memory_usage_used"`
	MemoryUsageFree     int64     `json:"memory_usage_free"`
	KernelVmallocTotal  int64     `json:"kernel_vmalloc_total"`
	KernelVmallocFree   int64     `json:"kernel_vmalloc_free"`
	KernelBuffers       int64     `json:"kernel_buffers"`
	KernelCached        int64     `json:"kernel_cached"`
	CurrentMemoryStatus string    `json:"current_memory_status"`
}

// CPUCore represents per-CPU core statistics
type CPUCore struct {
	CPUID  string `json:"cpuid"`
	User   string `json:"user"`
	Kernel string `json:"kernel"`
	Idle   string `json:"idle"`
}

// parseSystemResources parses the system resources output
func parseSystemResources(input string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	
	// Get current timestamp
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02")
	
	data := SystemResourcesData{
		CPUUsage: make([]CPUCore, 0),
	}
	
	// Regular expressions for parsing
	loadAvgPattern := regexp.MustCompile(`Load average:\s+1 minute:\s+([\d.]+)\s+5 minutes:\s+([\d.]+)\s+15 minutes:\s+([\d.]+)`)
	processesPattern := regexp.MustCompile(`Processes\s*:\s*(\d+)\s+total,\s*(\d+)\s+running`)
	cpuStatesPattern := regexp.MustCompile(`CPU states\s*:\s*([\d.]+)%\s+user,\s*([\d.]+)%\s+kernel,\s*([\d.]+)%\s+idle`)
	perCPUPattern := regexp.MustCompile(`CPU(\d+)\s+states\s*:\s*([\d.]+)%\s+user,\s*([\d.]+)%\s+kernel,\s*([\d.]+)%\s+idle`)
	memoryPattern := regexp.MustCompile(`Memory usage:\s*(\d+)K\s+total,\s*(\d+)K\s+used,\s*(\d+)K\s+free`)
	kernelVmallocPattern := regexp.MustCompile(`Kernel vmalloc:\s*(\d+)K\s+total,\s*(\d+)K\s+free`)
	kernelBuffersPattern := regexp.MustCompile(`Kernel buffers:\s*(\d+)K\s+Used`)
	kernelCachedPattern := regexp.MustCompile(`Kernel cached\s*:\s*(\d+)K\s+Used`)
	memoryStatusPattern := regexp.MustCompile(`Current memory status:\s*(\w+)`)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Parse load average
		if matches := loadAvgPattern.FindStringSubmatch(line); matches != nil {
			data.LoadAvg1Min = matches[1]
			data.LoadAvg5Min = matches[2]
			data.LoadAvg15Min = matches[3]
			continue
		}
		
		// Parse processes
		if matches := processesPattern.FindStringSubmatch(line); matches != nil {
			total, _ := strconv.Atoi(matches[1])
			running, _ := strconv.Atoi(matches[2])
			data.ProcessesTotal = total
			data.ProcessesRunning = running
			continue
		}
		
		// Parse overall CPU states
		if matches := cpuStatesPattern.FindStringSubmatch(line); matches != nil {
			data.CPUStateUser = matches[1]
			data.CPUStateKernel = matches[2]
			data.CPUStateIdle = matches[3]
			continue
		}
		
		// Parse per-CPU states
		if matches := perCPUPattern.FindStringSubmatch(line); matches != nil {
			cpuCore := CPUCore{
				CPUID:  matches[1],
				User:   matches[2],
				Kernel: matches[3],
				Idle:   matches[4],
			}
			data.CPUUsage = append(data.CPUUsage, cpuCore)
			continue
		}
		
		// Parse memory usage
		if matches := memoryPattern.FindStringSubmatch(line); matches != nil {
			total, _ := strconv.ParseInt(matches[1], 10, 64)
			used, _ := strconv.ParseInt(matches[2], 10, 64)
			free, _ := strconv.ParseInt(matches[3], 10, 64)
			data.MemoryUsageTotal = total
			data.MemoryUsageUsed = used
			data.MemoryUsageFree = free
			continue
		}
		
		// Parse kernel vmalloc
		if matches := kernelVmallocPattern.FindStringSubmatch(line); matches != nil {
			total, _ := strconv.ParseInt(matches[1], 10, 64)
			free, _ := strconv.ParseInt(matches[2], 10, 64)
			data.KernelVmallocTotal = total
			data.KernelVmallocFree = free
			continue
		}
		
		// Parse kernel buffers
		if matches := kernelBuffersPattern.FindStringSubmatch(line); matches != nil {
			buffers, _ := strconv.ParseInt(matches[1], 10, 64)
			data.KernelBuffers = buffers
			continue
		}
		
		// Parse kernel cached
		if matches := kernelCachedPattern.FindStringSubmatch(line); matches != nil {
			cached, _ := strconv.ParseInt(matches[1], 10, 64)
			data.KernelCached = cached
			continue
		}
		
		// Parse memory status
		if matches := memoryStatusPattern.FindStringSubmatch(line); matches != nil {
			data.CurrentMemoryStatus = matches[1]
			continue
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	entry := StandardizedEntry{
		DataType:  "cisco_nexus_system_resources",
		Timestamp: timestamp,
		Date:      date,
		Message:   data,
	}
	
	return []StandardizedEntry{entry}, nil
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
	inputFile := flag.String("input", "", "Input file containing Cisco Nexus system resources output")
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
		var sysResCmd string
		for _, c := range cmdFile.Commands {
			if c.Name == "system-resources" {
				sysResCmd = c.Command
				break
			}
		}
		if sysResCmd == "" {
			fmt.Fprintln(os.Stderr, "Error: No 'system-resources' command found in commands JSON.")
			os.Exit(1)
		}
		// Run the command using vsh
		vshOut, err := runVsh(sysResCmd)
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
	
	// Parse the system resources
	entries, err := parseSystemResources(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing system resources: %v\n", err)
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
	
	// Write each entry as a separate JSON object, one per line (JSON Lines format)
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "")
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding entry: %v\n", err)
			os.Exit(1)
		}
	}
}

// UnifiedParser implements the unified parser interface
type UnifiedParser struct{}

// GetDescription returns the parser description
func (p *UnifiedParser) GetDescription() string {
	return "Parses 'show system resources' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseSystemResources(content)
}
