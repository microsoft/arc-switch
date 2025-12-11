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
	DataType  string      `json:"data_type"`  // Always "cisco_nexus_version"
	Timestamp string      `json:"timestamp"`  // ISO 8601 timestamp
	Date      string      `json:"date"`       // Date in YYYY-MM-DD format
	Message   VersionData `json:"message"`    // Version-specific data
}

// VersionData represents the version data within the message field
type VersionData struct {
	BIOSVersion       string       `json:"bios_version"`
	NXOSVersion       string       `json:"nxos_version"`
	ReleaseType       string       `json:"release_type"`
	HostNXOSVersion   string       `json:"host_nxos_version"`
	BIOSCompileTime   string       `json:"bios_compile_time"`
	NXOSImageFile     string       `json:"nxos_image_file"`
	NXOSCompileTime   string       `json:"nxos_compile_time"`
	NXOSTimestamp     string       `json:"nxos_timestamp"`
	BootMode          string       `json:"boot_mode"`
	ChassisID         string       `json:"chassis_id"`
	CPUName           string       `json:"cpu_name"`
	MemoryKB          int64        `json:"memory_kb"`
	ProcessorBoardID  string       `json:"processor_board_id"`
	DeviceName        string       `json:"device_name"`
	BootflashKB       int64        `json:"bootflash_kb"`
	KernelUptime      KernelUptime `json:"kernel_uptime"`
	LastReset         LastReset    `json:"last_reset"`
	Plugins           []string     `json:"plugins"`
	ActivePackages    []string     `json:"active_packages"`
}

// KernelUptime represents the kernel uptime breakdown
type KernelUptime struct {
	Days    int `json:"days"`
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

// LastReset represents the last reset information
type LastReset struct {
	Usecs         int64  `json:"usecs"`
	Time          string `json:"time"`
	Reason        string `json:"reason"`
	SystemVersion string `json:"system_version"`
	Service       string `json:"service"`
}

// parseVersion parses the show version output
func parseVersion(input string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))

	// Get current timestamp
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02")

	data := VersionData{
		Plugins:        make([]string, 0),
		ActivePackages: make([]string, 0),
	}

	// Regular expressions for parsing
	biosVerPattern := regexp.MustCompile(`^\s*BIOS:\s*version\s+(.+)$`)
	nxosVerPattern := regexp.MustCompile(`^\s*NXOS:\s*version\s+([^\s\[]+)(?:\s*\[([^\]]+)\])?`)
	hostNxosVerPattern := regexp.MustCompile(`^\s*Host NXOS:\s*version\s+(.+)$`)
	biosCompilePattern := regexp.MustCompile(`^\s*BIOS compile time:\s*(.+)$`)
	nxosImagePattern := regexp.MustCompile(`^\s*NXOS image file is:\s*(.+)$`)
	nxosCompilePattern := regexp.MustCompile(`^\s*NXOS compile time:\s*([^\[]+)(?:\s*\[([^\]]+)\])?`)
	bootModePattern := regexp.MustCompile(`^\s*NXOS boot mode:\s*(.+)$`)
	chassisPattern := regexp.MustCompile(`^\s*cisco\s+(.+Chassis.*)$`)
	cpuMemPattern := regexp.MustCompile(`^\s*(.+(?:CPU|Processor).+)\s+with\s+(\d+)\s+kB\s+of\s+memory`)
	procBoardPattern := regexp.MustCompile(`^\s*Processor Board ID\s+(.+)$`)
	deviceNamePattern := regexp.MustCompile(`^\s*Device name:\s*(.+)$`)
	bootflashPattern := regexp.MustCompile(`^\s*bootflash:\s*(\d+)\s+kB`)
	uptimePattern := regexp.MustCompile(`Kernel uptime is\s+(\d+)\s+day\(s\),\s+(\d+)\s+hour\(s\),\s+(\d+)\s+minute\(s\),\s+(\d+)\s+second\(s\)`)
	lastResetAtPattern := regexp.MustCompile(`Last reset at\s+(\d+)\s+usecs after\s+(.+)$`)
	resetReasonPattern := regexp.MustCompile(`^\s*Reason:\s*(.+)$`)
	sysVersionPattern := regexp.MustCompile(`^\s*System version:\s*(.+)$`)
	servicePattern := regexp.MustCompile(`^\s*Service:\s*(.*)$`)
	pluginLinePattern := regexp.MustCompile(`^\s*(\w+\s+Plugin(?:,\s*\w+\s+Plugin)*)`)

	inPluginSection := false
	inActivePackageSection := false
	expectingService := false

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines in certain contexts
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for section markers
		if strings.HasPrefix(strings.TrimSpace(line), "plugin") {
			inPluginSection = true
			inActivePackageSection = false
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "Active Package") {
			inActivePackageSection = true
			inPluginSection = false
			continue
		}

		// Parse BIOS version
		if matches := biosVerPattern.FindStringSubmatch(line); matches != nil {
			data.BIOSVersion = strings.TrimSpace(matches[1])
			continue
		}

		// Parse NXOS version
		if matches := nxosVerPattern.FindStringSubmatch(line); matches != nil {
			data.NXOSVersion = strings.TrimSpace(matches[1])
			if len(matches) > 2 && matches[2] != "" {
				data.ReleaseType = strings.TrimSpace(matches[2])
			}
			continue
		}

		// Parse Host NXOS version
		if matches := hostNxosVerPattern.FindStringSubmatch(line); matches != nil {
			data.HostNXOSVersion = strings.TrimSpace(matches[1])
			continue
		}

		// Parse BIOS compile time
		if matches := biosCompilePattern.FindStringSubmatch(line); matches != nil {
			data.BIOSCompileTime = strings.TrimSpace(matches[1])
			continue
		}

		// Parse NXOS image file
		if matches := nxosImagePattern.FindStringSubmatch(line); matches != nil {
			data.NXOSImageFile = strings.TrimSpace(matches[1])
			continue
		}

		// Parse NXOS compile time
		if matches := nxosCompilePattern.FindStringSubmatch(line); matches != nil {
			data.NXOSCompileTime = strings.TrimSpace(matches[1])
			if len(matches) > 2 && matches[2] != "" {
				data.NXOSTimestamp = strings.TrimSpace(matches[2])
			}
			continue
		}

		// Parse boot mode
		if matches := bootModePattern.FindStringSubmatch(line); matches != nil {
			data.BootMode = strings.TrimSpace(matches[1])
			continue
		}

		// Parse chassis ID
		if matches := chassisPattern.FindStringSubmatch(line); matches != nil {
			data.ChassisID = "cisco " + strings.TrimSpace(matches[1])
			continue
		}

		// Parse CPU and memory
		if matches := cpuMemPattern.FindStringSubmatch(line); matches != nil {
			data.CPUName = strings.TrimSpace(matches[1])
			mem, _ := strconv.ParseInt(matches[2], 10, 64)
			data.MemoryKB = mem
			continue
		}

		// Parse processor board ID
		if matches := procBoardPattern.FindStringSubmatch(line); matches != nil {
			data.ProcessorBoardID = strings.TrimSpace(matches[1])
			continue
		}

		// Parse device name
		if matches := deviceNamePattern.FindStringSubmatch(line); matches != nil {
			data.DeviceName = strings.TrimSpace(matches[1])
			continue
		}

		// Parse bootflash size
		if matches := bootflashPattern.FindStringSubmatch(line); matches != nil {
			size, _ := strconv.ParseInt(matches[1], 10, 64)
			data.BootflashKB = size
			continue
		}

		// Parse kernel uptime
		if matches := uptimePattern.FindStringSubmatch(line); matches != nil {
			days, _ := strconv.Atoi(matches[1])
			hours, _ := strconv.Atoi(matches[2])
			minutes, _ := strconv.Atoi(matches[3])
			seconds, _ := strconv.Atoi(matches[4])
			data.KernelUptime = KernelUptime{
				Days:    days,
				Hours:   hours,
				Minutes: minutes,
				Seconds: seconds,
			}
			continue
		}

		// Parse last reset at
		if matches := lastResetAtPattern.FindStringSubmatch(line); matches != nil {
			usecs, _ := strconv.ParseInt(matches[1], 10, 64)
			data.LastReset.Usecs = usecs
			data.LastReset.Time = strings.TrimSpace(matches[2])
			continue
		}

		// Parse reset reason
		if matches := resetReasonPattern.FindStringSubmatch(line); matches != nil {
			data.LastReset.Reason = strings.TrimSpace(matches[1])
			continue
		}

		// Parse system version
		if matches := sysVersionPattern.FindStringSubmatch(line); matches != nil {
			data.LastReset.SystemVersion = strings.TrimSpace(matches[1])
			expectingService = true
			continue
		}

		// Parse service (comes after System version)
		if expectingService {
			if matches := servicePattern.FindStringSubmatch(line); matches != nil {
				data.LastReset.Service = strings.TrimSpace(matches[1])
				expectingService = false
				continue
			}
		}

		// Parse plugins
		if inPluginSection {
			if matches := pluginLinePattern.FindStringSubmatch(line); matches != nil {
				pluginStr := strings.TrimSpace(matches[1])
				plugins := strings.Split(pluginStr, ",")
				for _, p := range plugins {
					p = strings.TrimSpace(p)
					if p != "" {
						data.Plugins = append(data.Plugins, p)
					}
				}
				inPluginSection = false
				continue
			}
		}

		// Parse active packages (if any listed)
		if inActivePackageSection {
			trimmed := strings.TrimSpace(line)
			// Skip empty lines, prompts (ending with #), and section headers
			if trimmed != "" && !strings.HasSuffix(trimmed, "#") && !strings.HasPrefix(trimmed, "Active Package") {
				data.ActivePackages = append(data.ActivePackages, trimmed)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	entry := StandardizedEntry{
		DataType:  "cisco_nexus_version",
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
	inputFile := flag.String("input", "", "Input file containing Cisco Nexus show version output")
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
		// Run the command using vsh
		vshOut, err := runVsh(versionCmd)
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
	return "Parses 'show version' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseVersion(content)
}
