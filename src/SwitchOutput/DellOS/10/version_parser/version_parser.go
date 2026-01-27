package version_parser

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
	DataType  string      `json:"data_type"`  // Always "dell_os10_version"
	Timestamp string      `json:"timestamp"`  // ISO 8601 timestamp
	Date      string      `json:"date"`       // Date in YYYY-MM-DD format
	Message   VersionData `json:"message"`    // Version-specific data
}

// VersionData represents the version data within the message field
type VersionData struct {
	OSName        string       `json:"nxos_version"`       // Using nxos_version for OS name to align with Cisco
	OSVersion     string       `json:"bios_version"`       // Using bios_version for OS version to align with Cisco
	BuildVersion  string       `json:"nxos_compile_time"`  // Using nxos_compile_time for build version
	BuildTime     string       `json:"bios_compile_time"`  // Using bios_compile_time for build time
	SystemType    string       `json:"chassis_id"`         // Using chassis_id for system type to align with Cisco
	Architecture  string       `json:"cpu_name"`           // Using cpu_name for architecture to align with Cisco
	UpTime        string       `json:"boot_mode"`          // Using boot_mode for uptime string
	DeviceName    string       `json:"device_name"`        // Device hostname
	KernelUptime  KernelUptime `json:"kernel_uptime"`      // Parsed uptime breakdown
}

// KernelUptime represents the kernel uptime breakdown
type KernelUptime struct {
	Weeks   int `json:"weeks"`
	Days    int `json:"days"`
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

// parseVersion parses the show version output from Dell OS10
func parseVersion(input string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))

	// Get current timestamp
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02")

	data := VersionData{}

	// Regular expressions for parsing
	osNamePattern := regexp.MustCompile(`^(Dell SmartFabric OS10.*)$`)
	copyrightPattern := regexp.MustCompile(`^Copyright.*$`)
	osVersionPattern := regexp.MustCompile(`^OS Version:\s*(.+)$`)
	buildVersionPattern := regexp.MustCompile(`^Build Version:\s*(.+)$`)
	buildTimePattern := regexp.MustCompile(`^Build Time:\s*(.+)$`)
	systemTypePattern := regexp.MustCompile(`^System Type:\s*(.+)$`)
	architecturePattern := regexp.MustCompile(`^Architecture:\s*(.+)$`)
	upTimePattern := regexp.MustCompile(`^Up Time:\s*(.+)$`)
	deviceNamePattern := regexp.MustCompile(`^([a-zA-Z0-9\-]+)#\s*show version`)
	
	// Pattern for parsing uptime with weeks, days, hours, minutes, seconds
	// Format: "6 weeks 1 day 17:55:03" or "1 day 17:55:03" or "17:55:03"
	uptimeDetailPattern := regexp.MustCompile(`(?:(\d+)\s+weeks?)?\s*(?:(\d+)\s+days?)?\s*(\d+):(\d+):(\d+)`)

	var deviceName string

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse device name from prompt
		if matches := deviceNamePattern.FindStringSubmatch(line); matches != nil {
			deviceName = strings.TrimSpace(matches[1])
			continue
		}

		// Parse OS name
		if matches := osNamePattern.FindStringSubmatch(line); matches != nil {
			if !strings.HasPrefix(line, "Copyright") {
				data.OSName = strings.TrimSpace(matches[1])
			}
			continue
		}

		// Skip copyright line
		if matches := copyrightPattern.FindStringSubmatch(line); matches != nil {
			continue
		}

		// Parse OS version
		if matches := osVersionPattern.FindStringSubmatch(line); matches != nil {
			data.OSVersion = strings.TrimSpace(matches[1])
			continue
		}

		// Parse build version
		if matches := buildVersionPattern.FindStringSubmatch(line); matches != nil {
			data.BuildVersion = strings.TrimSpace(matches[1])
			continue
		}

		// Parse build time
		if matches := buildTimePattern.FindStringSubmatch(line); matches != nil {
			data.BuildTime = strings.TrimSpace(matches[1])
			continue
		}

		// Parse system type
		if matches := systemTypePattern.FindStringSubmatch(line); matches != nil {
			data.SystemType = strings.TrimSpace(matches[1])
			continue
		}

		// Parse architecture
		if matches := architecturePattern.FindStringSubmatch(line); matches != nil {
			data.Architecture = strings.TrimSpace(matches[1])
			continue
		}

		// Parse uptime
		if matches := upTimePattern.FindStringSubmatch(line); matches != nil {
			uptimeStr := strings.TrimSpace(matches[1])
			data.UpTime = uptimeStr
			
			// Parse detailed uptime components
			if detailMatches := uptimeDetailPattern.FindStringSubmatch(uptimeStr); detailMatches != nil {
				weeks := 0
				days := 0
				hours := 0
				minutes := 0
				seconds := 0
				
				if detailMatches[1] != "" {
					weeks, _ = strconv.Atoi(detailMatches[1])
				}
				if detailMatches[2] != "" {
					days, _ = strconv.Atoi(detailMatches[2])
				}
				if detailMatches[3] != "" {
					hours, _ = strconv.Atoi(detailMatches[3])
				}
				if detailMatches[4] != "" {
					minutes, _ = strconv.Atoi(detailMatches[4])
				}
				if detailMatches[5] != "" {
					seconds, _ = strconv.Atoi(detailMatches[5])
				}
				
				data.KernelUptime = KernelUptime{
					Weeks:   weeks,
					Days:    days,
					Hours:   hours,
					Minutes: minutes,
					Seconds: seconds,
				}
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Set device name
	data.DeviceName = deviceName

	entry := StandardizedEntry{
		DataType:  "dell_os10_version",
		Timestamp: timestamp,
		Date:      date,
		Message:   data,
	}

	return []StandardizedEntry{entry}, nil
}

// runClish runs the given command using the clish CLI and returns its output as a string
func runClish(command string) (string, error) {
	cmd := []string{"/opt/dell/os10/bin/clish", "-c", command}
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("clish error: %v, output: %s", err, string(out))
	}
	return string(out), nil
}

func main() {
	// Define command line flags
	inputFile := flag.String("input", "", "Input file containing Dell OS10 show version output")
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
		var versionCmd string
		for _, c := range cmdFile.Commands {
			if c.Name == "version" {
				versionCmd = c.Command
				break
			}
		}
		if versionCmd == "" {
			fmt.Fprintln(os.Stderr, "Error: No 'version' command found in commands JSON.")
			os.Exit(1)
		}
		// Run the command using clish
		clishOut, err := runClish(versionCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running clish: %v\n", err)
			os.Exit(1)
		}
		inputData = clishOut
	} else if *inputFile != "" {
		// Read from file
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
		inputData = string(data)
	}

	// Parse the version info
	entries, err := parseVersion(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing version: %v\n", err)
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
	return "Parses 'show version' output for Dell OS10"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseVersion(content)
}
