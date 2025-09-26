package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// SFPData represents the structure of the JSON output for an SFP module.
type SFPData struct {
	SFP                string           `json:"sfp"`
	ID                 int              `json:"id"`
	ExtID              int              `json:"ext_id"`
	Connector          int              `json:"connector"`
	TransceiverCode    int              `json:"transceiver_code"`
	Encoding           int              `json:"encoding"`
	BRNominal          int              `json:"br_nominal"`
	Length             Length           `json:"length"`
	VendorRev          string           `json:"vendor_rev"`
	LaserWavelength    string           `json:"laser_wavelength"`
	TunableWavelength  string           `json:"tunable_wavelength"`
	CheckCodeBase      int              `json:"check_code_base"`
	SerialExtendedID   SerialExtendedID `json:"serial_extended_id"`
	Diagnostics        Diagnostics      `json:"diagnostics"`
}

// Length represents the length details of the SFP module.
type Length struct {
	SFMKm    int `json:"sfm_km"`
	OM32m    int `json:"om3_2m"`
	OM21m    int `json:"om2_1m"`
	OM11m    int `json:"om1_1m"`
	Copper1m int `json:"copper_1m"`
}

// SerialExtendedID represents the serial and extended ID details of the SFP module.
type SerialExtendedID struct {
	Options      int    `json:"options"`
	BRMax        int    `json:"br_max"`
	BRMin        int    `json:"br_min"`
	VendorSN     string `json:"vendor_sn"`
	Datecode     string `json:"datecode"`
	CheckCodeExt int    `json:"check_code_ext"`
}

// Diagnostics represents the diagnostic details of the SFP module.
type Diagnostics struct {
	RXPowerMeasurementType int       `json:"rx_power_measurement_type"`
	Thresholds             Thresholds `json:"thresholds"`
	Current                Current    `json:"current"`
}

// Thresholds represents the alarm and warning thresholds for the SFP module.
type Thresholds struct {
	TempHighAlarm     string `json:"temp_high_alarm"`
	VoltageHighAlarm  string `json:"voltage_high_alarm"`
	BiasHighAlarm     string `json:"bias_high_alarm"`
	TXPowerHighAlarm  string `json:"tx_power_high_alarm"`
	RXPowerHighAlarm  string `json:"rx_power_high_alarm"`
	TempLowAlarm      string `json:"temp_low_alarm"`
	VoltageLowAlarm   string `json:"voltage_low_alarm"`
	BiasLowAlarm      string `json:"bias_low_alarm"`
	TXPowerLowAlarm   string `json:"tx_power_low_alarm"`
	RXPowerLowAlarm   string `json:"rx_power_low_alarm"`
	TempHighWarning   string `json:"temp_high_warning"`
	VoltageHighWarning string `json:"voltage_high_warning"`
	BiasHighWarning   string `json:"bias_high_warning"`
	TXPowerHighWarning string `json:"tx_power_high_warning"`
	RXPowerHighWarning string `json:"rx_power_high_warning"`
	TempLowWarning    string `json:"temp_low_warning"`
	VoltageLowWarning string `json:"voltage_low_warning"`
	BiasLowWarning    string `json:"bias_low_warning"`
	TXPowerLowWarning string `json:"tx_power_low_warning"`
	RXPowerLowWarning string `json:"rx_power_low_warning"`
}

// Current represents the current diagnostic values of the SFP module.
type Current struct {
	Temperature      string `json:"temperature"`
	Voltage          string `json:"voltage"`
	RateSelectState  bool   `json:"rate_select_state"`
}

func main() {
	// Define command-line flags for input file and test mode.
	inputfile := flag.String("inputfile", "", "Path to the input file containing 'show interface phy-eth' data")
	test := flag.Bool("test", false, "Enable test mode (no OS commands will be executed, verbose output enabled)")
	flag.Parse()

	// Validate input parameters.
	if *test && *inputfile == "" {
		log.Fatal("Error: When test mode is enabled, an inputfile must be provided.")
	}

	// Fetch SFP data from the input file or CLI command.
	data, err := getSFPData(*inputfile, *test)
	if err != nil {
		log.Fatalf("Error fetching SFP data: %v", err)
	}

	// Process the raw data into JSON format.
	jsonData, err := processToJSON(data)
	if err != nil {
		log.Fatalf("Error processing data to JSON: %v", err)
	}

	// Log the JSON data to syslog.
	err = logEachInterfaceToSyslog(jsonData, *test)
	if err != nil {
		log.Fatalf("Error logging data to syslog: %v", err)
	}

	if *test {
		fmt.Println("Operation completed successfully in test mode.")
	}
}

// getSFPData fetches the SFP data from the specified source.
// If an input file is provided, it reads the data from the file.
// Otherwise, it executes the CLI command to fetch the data.
func getSFPData(inputfile string, test bool) (string, error) {
	if inputfile != "" {
		if test {
			fmt.Printf("Reading data from file: %s\n", inputfile)
		}
		data, err := os.ReadFile(inputfile)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return string(data), nil
	}

	if test {
		fmt.Println("Test mode enabled. Command to fetch data:")
		fmt.Println("clish -c show interface phy-eth")
		return "", errors.New("test mode is enabled, but no inputfile was provided")
	}

	// Use CLI command to fetch data.
	cmd := exec.Command("/opt/dell/os10/bin/clish", "-c", "show interface phy-eth")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute CLI command: %w", err)
	}
	return string(output), nil
}

// processToJSON processes the raw SFP data into JSON format.
// It parses the data line by line and populates the SFPData structure.
func processToJSON(data string) ([]byte, error) {
    sfps := []SFPData{}
    lines := strings.Split(data, "\n")
    var currentSFP *SFPData

    // Regular expressions for parsing
    reSFP := regexp.MustCompile(`^SFP(\S+) Id\s+=\s+(\d+)$`)
    reField := regexp.MustCompile(`^SFP\S+\s+(\S+)\s+=\s+(.+)$`)

    for _, line := range lines {
        line = strings.TrimSpace(line)

        if match := reSFP.FindStringSubmatch(line); match != nil {
            // Start of a new SFP
            if currentSFP != nil {
                sfps = append(sfps, *currentSFP)
            }
            id, _ := strconv.Atoi(match[2])
            currentSFP = &SFPData{
                SFP: match[1],
                ID:  id,
            }
        } else if currentSFP != nil {
            // Parse additional fields for the current SFP
            if match := reField.FindStringSubmatch(line); match != nil {
                field := strings.ToLower(strings.ReplaceAll(match[1], " ", "_"))
                value := match[2]
                switch field {
                case "ext_id":
                    currentSFP.ExtID, _ = strconv.Atoi(value)
                case "connector":
                    currentSFP.Connector, _ = strconv.Atoi(value)
                case "transceiver_code":
                    currentSFP.TransceiverCode, _ = strconv.Atoi(value)
                case "encoding":
                    currentSFP.Encoding, _ = strconv.Atoi(value)
                case "br_nominal":
                    currentSFP.BRNominal, _ = strconv.Atoi(value)
                case "length(sfm)_km":
                    currentSFP.Length.SFMKm, _ = strconv.Atoi(value)
                case "length(om3)_2m":
                    currentSFP.Length.OM32m, _ = strconv.Atoi(value)
                case "length(om2)_1m":
                    currentSFP.Length.OM21m, _ = strconv.Atoi(value)
                case "length(om1)_1m":
                    currentSFP.Length.OM11m, _ = strconv.Atoi(value)
                case "length(copper)_1m":
                    currentSFP.Length.Copper1m, _ = strconv.Atoi(value)
                case "vendor_rev":
                    currentSFP.VendorRev = value
                case "laser_wavelength":
                    currentSFP.LaserWavelength = value
                case "tunable_wavelength":
                    currentSFP.TunableWavelength = value
                case "check_code_base":
                    currentSFP.CheckCodeBase, _ = strconv.Atoi(value)
                case "options":
                    currentSFP.SerialExtendedID.Options, _ = strconv.Atoi(value)
                case "br_max":
                    currentSFP.SerialExtendedID.BRMax, _ = strconv.Atoi(value)
                case "br_min":
                    currentSFP.SerialExtendedID.BRMin, _ = strconv.Atoi(value)
                case "vendor_sn":
                    currentSFP.SerialExtendedID.VendorSN = value
                case "datecode":
                    currentSFP.SerialExtendedID.Datecode = value
                case "checkcodeext":
                    currentSFP.SerialExtendedID.CheckCodeExt, _ = strconv.Atoi(value)
                case "rx_power_measurement_type":
                    currentSFP.Diagnostics.RXPowerMeasurementType, _ = strconv.Atoi(value)
                }
            }
        }
    }

    // Add the last SFP
    if currentSFP != nil {
        sfps = append(sfps, *currentSFP)
    }

    // Convert to JSON
    jsonData, err := json.Marshal(sfps)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
    }
    return jsonData, nil
}

// parseBool converts a string to a boolean value.
func parseBool(value string) bool {
	return strings.ToLower(value) == "true"
}

// logEachInterfaceToSyslog logs each interface's JSON data as a unique logger entry.
// If test mode is enabled, it prints the logger commands to STDOUT.
func logEachInterfaceToSyslog(jsonData []byte, test bool) error {
    var sfps []SFPData
    err := json.Unmarshal(jsonData, &sfps)
    if err != nil {
        return fmt.Errorf("failed to unmarshal JSON data: %w", err)
    }

    for _, sfp := range sfps {
        interfaceJSON, err := json.Marshal(sfp)
        if err != nil {
            return fmt.Errorf("failed to marshal interface data to JSON: %w", err)
        }

        if test {
            fmt.Printf("Test mode enabled. Logger command for interface %s:\n", sfp.SFP)
            fmt.Printf("logger -p local0.info -t sfpdata '%s';\n", string(interfaceJSON))
        } else {
            // Prepare the logger command for each interface.
            loggerCmd := exec.Command("logger", "--size", "4096", "-p", "local0.info", "-t", "sfpdata", string(interfaceJSON)+";")
            err := loggerCmd.Run()
            if err != nil {
                return fmt.Errorf("failed to log data for interface %s to syslog: %w", sfp.SFP, err)
            }
        }
    }

    return nil
}