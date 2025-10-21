package environment_power_parser

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
	DataType  string               `json:"data_type"` // Always "cisco_nexus_environment_power"
	Timestamp string               `json:"timestamp"` // ISO 8601 timestamp
	Date      string               `json:"date"`      // Date in YYYY-MM-DD format
	Message   EnvironmentPowerData `json:"message"`   // Environment power-specific data
}

// EnvironmentPowerData represents the power data within the message field
type EnvironmentPowerData struct {
	Voltage       string              `json:"voltage,omitempty"`
	PowerSupplies []PowerSupply       `json:"power_supplies,omitempty"`
	PowerSummary  PowerSummary        `json:"power_summary,omitempty"`
	PowerDetails  PowerUsageDetails   `json:"power_usage_details,omitempty"`
	PSDetails     []PowerSupplyDetail `json:"power_supply_details,omitempty"`
}

// PowerSupply represents a single power supply in the table
type PowerSupply struct {
	PSNumber      string `json:"ps_number"`
	Model         string `json:"model"`
	ActualOutput  string `json:"actual_output"`
	ActualInput   string `json:"actual_input"`
	TotalCapacity string `json:"total_capacity"`
	Status        string `json:"status"`
}

// PowerSummary represents the power usage summary section
type PowerSummary struct {
	PSRedundancyModeConfigured  string `json:"ps_redundancy_mode_configured,omitempty"`
	PSRedundancyModeOperational string `json:"ps_redundancy_mode_operational,omitempty"`
	TotalPowerCapacity          string `json:"total_power_capacity,omitempty"`
	TotalGridAPowerCapacity     string `json:"total_grid_a_power_capacity,omitempty"`
	TotalGridBPowerCapacity     string `json:"total_grid_b_power_capacity,omitempty"`
	TotalPowerOfAllInputs       string `json:"total_power_of_all_inputs,omitempty"`
	TotalPowerOutputActualDraw  string `json:"total_power_output_actual_draw,omitempty"`
	TotalPowerInputActualDraw   string `json:"total_power_input_actual_draw,omitempty"`
	TotalPowerAllocatedBudget   string `json:"total_power_allocated_budget,omitempty"`
	TotalPowerAvailable         string `json:"total_power_available,omitempty"`
}

// PowerUsageDetails represents the power usage details section
type PowerUsageDetails struct {
	PowerReservedForSupervisors string `json:"power_reserved_for_supervisors,omitempty"`
	PowerReservedForFabricSC    string `json:"power_reserved_for_fabric_sc,omitempty"`
	PowerReservedForFanModules  string `json:"power_reserved_for_fan_modules,omitempty"`
	TotalPowerReserved          string `json:"total_power_reserved,omitempty"`
	AllInletCordsConnected      string `json:"all_inlet_cords_connected,omitempty"`
}

// PowerSupplyDetail represents detailed power supply information
type PowerSupplyDetail struct {
	Name          string `json:"name"`
	TotalCapacity string `json:"total_capacity"`
	Voltage       string `json:"voltage"`
	Pin           string `json:"pin,omitempty"`
	Vin           string `json:"vin,omitempty"`
	Iin           string `json:"iin,omitempty"`
	Pout          string `json:"pout,omitempty"`
	Vout          string `json:"vout,omitempty"`
	Iout          string `json:"iout,omitempty"`
	CordStatus    string `json:"cord_status,omitempty"`
	SoftwareAlarm string `json:"software_alarm,omitempty"`
}

// parsePowerEnvironment parses the show environment power detail output
func parsePowerEnvironment(content string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	timestamp := time.Now()

	var powerData EnvironmentPowerData
	var currentPSDetail *PowerSupplyDetail

	// Regular expressions
	voltageRegex := regexp.MustCompile(`^Voltage:\s*(\d+)\s*Volts`)
	psTableLineRegex := regexp.MustCompile(`^\s*(\d+)\s+(\S+)\s+(\d+\s+W)\s+(\d+\s+W)\s+(\d+\s+W)\s+(.+?)\s*$`)
	psDetailHeaderRegex := regexp.MustCompile(`^(PS_\d+)\s+total capacity:\s*(.+?)\s+Voltage:(.+)$`)
	psDetailDataRegex := regexp.MustCompile(`^Pin:(.+?)\s+Vin:(.+?)\s+Iin:(.+?)\s+Pout:(.+?)\s+Vout:(.+?)\s+Iout:(.+)$`)
	cordStatusRegex := regexp.MustCompile(`^Cord connected to (.+)$`)
	softwareAlarmRegex := regexp.MustCompile(`^Software-Alarm:\s*(.+)$`)

	inPowerSupplyTable := false
	inPowerSummary := false
	inPowerDetails := false
	inPSDetails := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines and command echoes
		if trimmedLine == "" || strings.Contains(line, "show environment power detail") {
			continue
		}

		// Parse voltage
		if matches := voltageRegex.FindStringSubmatch(line); matches != nil {
			powerData.Voltage = matches[1] + " Volts"
			continue
		}

		// Detect Power Supply table section
		if strings.Contains(line, "Power Supply:") {
			inPowerSupplyTable = true
			continue
		}

		// Detect Power Usage Summary section
		if strings.Contains(line, "Power Usage Summary:") {
			inPowerSupplyTable = false
			inPowerSummary = true
			continue
		}

		// Detect Power Usage details section
		if strings.Contains(line, "Power Usage details:") {
			inPowerSummary = false
			inPowerDetails = true
			continue
		}

		// Detect Power supply details section
		if strings.Contains(line, "Power supply details:") {
			inPowerDetails = false
			inPSDetails = true
			continue
		}

		// Parse power supply table entries
		if inPowerSupplyTable {
			// Skip header and separator lines
			if strings.Contains(line, "Power") && strings.Contains(line, "Supply") ||
				strings.Contains(line, "Model") || strings.Contains(line, "----") {
				continue
			}

			// Parse data line
			if matches := psTableLineRegex.FindStringSubmatch(line); matches != nil {
				ps := PowerSupply{
					PSNumber:      matches[1],
					Model:         strings.TrimSpace(matches[2]),
					ActualOutput:  strings.TrimSpace(matches[3]),
					ActualInput:   strings.TrimSpace(matches[4]),
					TotalCapacity: strings.TrimSpace(matches[5]),
					Status:        strings.TrimSpace(matches[6]),
				}
				powerData.PowerSupplies = append(powerData.PowerSupplies, ps)
			}
		}

		// Parse power summary
		if inPowerSummary {
			if strings.Contains(line, "Power Supply redundancy mode (configured)") {
				parts := strings.Split(line, ")")
				if len(parts) > 1 {
					powerData.PowerSummary.PSRedundancyModeConfigured = strings.TrimSpace(parts[len(parts)-1])
				}
			} else if strings.Contains(line, "Power Supply redundancy mode (operational)") {
				parts := strings.Split(line, ")")
				if len(parts) > 1 {
					powerData.PowerSummary.PSRedundancyModeOperational = strings.TrimSpace(parts[len(parts)-1])
				}
			} else if strings.Contains(line, "Total Power Capacity (based on configured mode)") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					powerData.PowerSummary.TotalPowerCapacity = strings.Join(parts[len(parts)-2:], " ")
				}
			} else if strings.Contains(line, "Total Grid-A") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					powerData.PowerSummary.TotalGridAPowerCapacity = strings.Join(parts[len(parts)-2:], " ")
				}
			} else if strings.Contains(line, "Total Grid-B") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					powerData.PowerSummary.TotalGridBPowerCapacity = strings.Join(parts[len(parts)-2:], " ")
				}
			} else if strings.Contains(line, "Total Power of all Inputs") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					powerData.PowerSummary.TotalPowerOfAllInputs = strings.Join(parts[len(parts)-2:], " ")
				}
			} else if strings.Contains(line, "Total Power Output (actual draw)") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					powerData.PowerSummary.TotalPowerOutputActualDraw = strings.Join(parts[len(parts)-2:], " ")
				}
			} else if strings.Contains(line, "Total Power Input (actual draw)") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					powerData.PowerSummary.TotalPowerInputActualDraw = strings.Join(parts[len(parts)-2:], " ")
				}
			} else if strings.Contains(line, "Total Power Allocated (budget)") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					powerData.PowerSummary.TotalPowerAllocatedBudget = parts[len(parts)-1]
				}
			} else if strings.Contains(line, "Total Power Available for additional modules") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					powerData.PowerSummary.TotalPowerAvailable = parts[len(parts)-1]
				}
			}
		}

		// Parse power usage details
		if inPowerDetails {
			if strings.Contains(line, "Power reserved for Supervisor") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					powerData.PowerDetails.PowerReservedForSupervisors = parts[len(parts)-1]
				}
			} else if strings.Contains(line, "Power reserved for Fabric, SC Module") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					powerData.PowerDetails.PowerReservedForFabricSC = parts[len(parts)-1]
				}
			} else if strings.Contains(line, "Power reserved for Fan Module") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					powerData.PowerDetails.PowerReservedForFanModules = parts[len(parts)-1]
				}
			} else if strings.Contains(line, "Total power reserved for Sups,SCs,Fabrics,Fans") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					powerData.PowerDetails.TotalPowerReserved = parts[len(parts)-1]
				}
			} else if strings.Contains(line, "Are all inlet cords connected") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					powerData.PowerDetails.AllInletCordsConnected = strings.TrimSpace(parts[1])
				}
			}
		}

		// Parse power supply details
		if inPSDetails {
			// PS header line
			if matches := psDetailHeaderRegex.FindStringSubmatch(line); matches != nil {
				// Save previous PS detail if exists
				if currentPSDetail != nil {
					powerData.PSDetails = append(powerData.PSDetails, *currentPSDetail)
				}

				currentPSDetail = &PowerSupplyDetail{
					Name:          matches[1],
					TotalCapacity: strings.TrimSpace(matches[2]),
					Voltage:       strings.TrimSpace(matches[3]),
				}
			} else if currentPSDetail != nil {
				// PS detail data line
				if matches := psDetailDataRegex.FindStringSubmatch(line); matches != nil {
					currentPSDetail.Pin = strings.TrimSpace(matches[1])
					currentPSDetail.Vin = strings.TrimSpace(matches[2])
					currentPSDetail.Iin = strings.TrimSpace(matches[3])
					currentPSDetail.Pout = strings.TrimSpace(matches[4])
					currentPSDetail.Vout = strings.TrimSpace(matches[5])
					currentPSDetail.Iout = strings.TrimSpace(matches[6])
				} else if matches := cordStatusRegex.FindStringSubmatch(line); matches != nil {
					currentPSDetail.CordStatus = "connected to " + matches[1]
				} else if matches := softwareAlarmRegex.FindStringSubmatch(line); matches != nil {
					currentPSDetail.SoftwareAlarm = strings.TrimSpace(matches[1])
				}
			}
		}
	}

	// Save last PS detail if exists
	if currentPSDetail != nil {
		powerData.PSDetails = append(powerData.PSDetails, *currentPSDetail)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Create single standardized entry
	entry := StandardizedEntry{
		DataType:  "cisco_nexus_environment_power",
		Timestamp: timestamp.Format(time.RFC3339),
		Date:      timestamp.Format("2006-01-02"),
		Message:   powerData,
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
	inputFile := flag.String("input", "", "Input file containing Cisco Nexus power environment output")
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
		var powerCmd string
		for _, c := range cmdFile.Commands {
			if c.Name == "power-supply" {
				powerCmd = c.Command
				break
			}
		}
		if powerCmd == "" {
			fmt.Fprintln(os.Stderr, "Error: No 'power-supply' command found in commands JSON.")
			os.Exit(1)
		}
		// Run the command using vsh
		vshOut, err := runVsh(powerCmd)
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

	// Parse the power environment data
	entries, err := parsePowerEnvironment(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing power environment data: %v\n", err)
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
	encoder.SetIndent("", "  ")
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
	return "Parses 'show environment power detail' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parsePowerEnvironment(content)
}
