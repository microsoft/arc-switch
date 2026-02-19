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
	Message   SystemData `json:"message"`
}

// SystemData represents parsed show system output
type SystemData struct {
	NodeID        int           `json:"node_id"`
	MAC           string        `json:"mac"`
	NumberOfMACs  int           `json:"number_of_macs"`
	UpTime        string        `json:"up_time"`
	DiagOS        string        `json:"diag_os"`
	PCIeVersion   string        `json:"pcie_version"`
	Units         []SystemUnit  `json:"units"`
	PowerSupplies []PowerSupply `json:"power_supplies"`
	FanTrays      []FanTray     `json:"fan_trays"`
}

// SystemUnit represents a unit in the system
type SystemUnit struct {
	UnitID           int    `json:"unit_id"`
	Status           string `json:"status"`
	SystemIdentifier string `json:"system_identifier"`
	RequiredType     string `json:"required_type"`
	CurrentType      string `json:"current_type"`
	HardwareRevision string `json:"hardware_revision"`
	SoftwareVersion  string `json:"software_version"`
	PhysicalPorts    string `json:"physical_ports"`
}

// PowerSupply represents a PSU entry
type PowerSupply struct {
	PSUID     int    `json:"psu_id"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	Power     int    `json:"power_watts"`
	AvgPower  int    `json:"avg_power_watts"`
	AirFlow   string `json:"air_flow"`
	FanSpeed  int    `json:"fan_speed_rpm"`
	FanStatus string `json:"fan_status"`
}

// FanTray represents a fan tray entry
type FanTray struct {
	TrayID  int       `json:"tray_id"`
	Status  string    `json:"status"`
	AirFlow string    `json:"air_flow"`
	Fans    []FanInfo `json:"fans"`
}

// FanInfo represents an individual fan within a tray
type FanInfo struct {
	FanID  int    `json:"fan_id"`
	Speed  int    `json:"speed_rpm"`
	Status string `json:"status"`
}

type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseSystem parses Dell OS10 show system output
func parseSystem(content string) ([]StandardizedEntry, error) {
	data := SystemData{}
	lines := strings.Split(content, "\n")
	timestamp := time.Now().UTC()

	kvRegex := regexp.MustCompile(`^(.+?)\s*:\s+(.+)$`)
	sectionRegex := regexp.MustCompile(`^--\s+(.+)\s+--$`)
	separatorRegex := regexp.MustCompile(`^-{10,}$`)
	psuRegex := regexp.MustCompile(`^(\d+)\s+(up|down|absent)\s+(\S+)\s+(\d+)\s+(\d+)\s+\S+\s+(\S+)\s+(\d+)\s+(\d+)\s+(\S+)`)
	fanTrayRegex := regexp.MustCompile(`^(\d+)\s+(up|down|absent)\s+(\S+)\s+(\d+)\s+(\d+)\s+(\S+)`)
	fanContinueRegex := regexp.MustCompile(`^\s+(\d+)\s+(\d+)\s+(\S+)`)

	currentSection := "top"
	var currentUnit *SystemUnit
	var currentFanTray *FanTray

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if separatorRegex.MatchString(trimmed) {
			continue
		}

		// Detect section headers
		if match := sectionRegex.FindStringSubmatch(trimmed); match != nil {
			sectionName := strings.TrimSpace(match[1])

			if currentUnit != nil {
				data.Units = append(data.Units, *currentUnit)
				currentUnit = nil
			}
			if currentFanTray != nil {
				data.FanTrays = append(data.FanTrays, *currentFanTray)
				currentFanTray = nil
			}

			if strings.HasPrefix(sectionName, "Unit") {
				currentSection = "unit"
				currentUnit = &SystemUnit{}
				parts := strings.Fields(sectionName)
				if len(parts) >= 2 {
					currentUnit.UnitID, _ = strconv.Atoi(parts[1])
				}
			} else if strings.Contains(sectionName, "Power") {
				currentSection = "power"
			} else if strings.Contains(sectionName, "Fan") {
				currentSection = "fan"
			}
			continue
		}

		// Skip table header lines
		if strings.HasPrefix(trimmed, "PSU-ID") || strings.HasPrefix(trimmed, "FanTray") {
			continue
		}

		switch currentSection {
		case "top":
			if match := kvRegex.FindStringSubmatch(trimmed); match != nil {
				key := strings.TrimSpace(match[1])
				value := strings.TrimSpace(match[2])
				switch key {
				case "Node Id":
					data.NodeID, _ = strconv.Atoi(value)
				case "MAC":
					data.MAC = value
				case "Number of MACs":
					data.NumberOfMACs, _ = strconv.Atoi(value)
				case "Up Time":
					data.UpTime = value
				case "DiagOS":
					data.DiagOS = value
				case "PCIe Version":
					data.PCIeVersion = value
				}
			}

		case "unit":
			if currentUnit != nil {
				if match := kvRegex.FindStringSubmatch(trimmed); match != nil {
					key := strings.TrimSpace(match[1])
					value := strings.TrimSpace(match[2])
					switch key {
					case "Status":
						currentUnit.Status = value
					case "System Identifier":
						currentUnit.SystemIdentifier = value
					case "Required Type":
						currentUnit.RequiredType = value
					case "Current Type":
						currentUnit.CurrentType = value
					case "Hardware Revision":
						currentUnit.HardwareRevision = value
					case "Software Version":
						currentUnit.SoftwareVersion = value
					case "Physical Ports":
						currentUnit.PhysicalPorts = value
					}
				}
			}

		case "power":
			if match := psuRegex.FindStringSubmatch(trimmed); match != nil {
				psu := PowerSupply{
					Status:  match[2],
					Type:    match[3],
					AirFlow: match[6],
				}
				psu.PSUID, _ = strconv.Atoi(match[1])
				psu.Power, _ = strconv.Atoi(match[4])
				psu.AvgPower, _ = strconv.Atoi(match[5])
				psu.FanSpeed, _ = strconv.Atoi(match[8])
				psu.FanStatus = match[9]
				data.PowerSupplies = append(data.PowerSupplies, psu)
			}

		case "fan":
			// Check for fan continuation line first (indented)
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				if match := fanContinueRegex.FindStringSubmatch(line); match != nil {
					if currentFanTray != nil {
						fan := FanInfo{Status: match[3]}
						fan.FanID, _ = strconv.Atoi(match[1])
						fan.Speed, _ = strconv.Atoi(match[2])
						currentFanTray.Fans = append(currentFanTray.Fans, fan)
					}
					continue
				}
			}
			// Fan tray header line
			if match := fanTrayRegex.FindStringSubmatch(trimmed); match != nil {
				if currentFanTray != nil {
					data.FanTrays = append(data.FanTrays, *currentFanTray)
				}
				currentFanTray = &FanTray{
					Status:  match[2],
					AirFlow: match[3],
				}
				currentFanTray.TrayID, _ = strconv.Atoi(match[1])
				fan := FanInfo{Status: match[6]}
				fan.FanID, _ = strconv.Atoi(match[4])
				fan.Speed, _ = strconv.Atoi(match[5])
				currentFanTray.Fans = append(currentFanTray.Fans, fan)
			}
		}
	}

	if currentUnit != nil {
		data.Units = append(data.Units, *currentUnit)
	}
	if currentFanTray != nil {
		data.FanTrays = append(data.FanTrays, *currentFanTray)
	}

	entry := StandardizedEntry{
		DataType:  "dell_os10_system",
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
	inputFile := flag.String("input", "", "Input file containing 'show system' output")
	outputFile := flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	commandsFile := flag.String("commands", "", "Commands JSON file")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 System Parser")
		fmt.Println("Parses 'show system' output and converts to JSON format.")
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
		command, err := findCommand(config, "system")
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

	entries, err := parseSystem(inputData)
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
	return "Parses 'show system' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseSystem(string(input))
}
