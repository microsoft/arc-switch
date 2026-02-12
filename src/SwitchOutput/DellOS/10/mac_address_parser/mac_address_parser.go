package mac_address_parser

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
	DataType  string       `json:"data_type"`
	Timestamp string       `json:"timestamp"`
	Date      string       `json:"date"`
	Message   MacTableData `json:"message"`
}

// MacTableData represents the MAC table data within the message field
type MacTableData struct {
	VLAN         string `json:"vlan"`
	MACAddress   string `json:"mac_address"`
	Type         string `json:"type"`
	Interface    string `json:"interface"`
	PrivateVLAN  string `json:"private_vlan,omitempty"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseMAC parses the MAC address table output and emits each entry as JSON
func parseMAC(input string) ([]StandardizedEntry, error) {
	scanner := bufio.NewScanner(strings.NewReader(input))
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02")

	// Find the table header
	foundHeader := false
	for scanner.Scan() {
		line := scanner.Text()
		// Dell OS10 header format: "VlanId  Mac Address        Type     Interface"
		if strings.Contains(line, "VlanId") && strings.Contains(line, "Mac Address") {
			foundHeader = true
			break
		}
	}

	if !foundHeader {
		return nil, fmt.Errorf("could not find MAC address table header")
	}

	// MAC table entry line pattern for Dell OS10
	// Format: VlanId  Mac Address        Type     Interface
	// Example: 1       90:b1:1c:f4:a6:8f  dynamic  ethernet1/1/3
	// Can also have "pv <vlan-id>" for private VLAN
	entryPattern := regexp.MustCompile(`^\s*(\d+)\s+([0-9a-f:]+)\s+(\S+)\s+(\S+)(?:\s+pv\s+(\d+))?$`)

	var entries []StandardizedEntry

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Skip separator lines
		if strings.HasPrefix(strings.TrimSpace(line), "-") {
			continue
		}

		matches := entryPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		entry := StandardizedEntry{
			DataType:  "dell_os10_mac_table",
			Timestamp: timestamp,
			Date:      date,
			Message: MacTableData{
				VLAN:       matches[1],
				MACAddress: matches[2],
				Type:       matches[3],
				Interface:  matches[4],
			},
		}

		// Check for private VLAN
		if len(matches) > 5 && matches[5] != "" {
			entry.Message.PrivateVLAN = matches[5]
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// runCommand executes a command on the Dell OS10 switch using clish
func runCommand(command string) (string, error) {
	cmd := exec.Command("/opt/dell/os10/bin/clish", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("clish error: %v, output: %s", err, string(output))
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

// findMacAddressCommand finds the mac-address-table command in the commands.json
func findMacAddressCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "mac-address-table" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("mac-address-table command not found in commands file")
}

func main() {
	inputFile := flag.String("input", "", "Input file containing Dell OS10 MAC address table output")
	outputFile := flag.String("output", "", "Output file to write JSON results (default: stdout)")
	commandsFile := flag.String("commands", "", "Path to JSON file containing CLI commands")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 MAC Address Parser")
		fmt.Println("Parses 'show mac address-table' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  mac_address_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show mac address-table' output")
		fmt.Println("  -output <file>    Output file for JSON data (optional, defaults to stdout)")
		fmt.Println("  -commands <file>  Commands JSON file (used when no input file is specified)")
		fmt.Println("  -help             Show this help message")
		return
	}

	if (*inputFile != "" && *commandsFile != "") || (*inputFile == "" && *commandsFile == "") {
		fmt.Fprintln(os.Stderr, "Error: You must specify exactly one of -input or -commands.")
		os.Exit(1)
	}

	var inputData string

	if *commandsFile != "" {
		config, err := loadCommandsFromFile(*commandsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading commands file: %v\n", err)
			os.Exit(1)
		}

		macCmd, err := findMacAddressCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		output, err := runCommand(macCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running command: %v\n", err)
			os.Exit(1)
		}
		inputData = output
	} else if *inputFile != "" {
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
			os.Exit(1)
		}
		inputData = string(data)
	}

	// Parse the MAC address table
	entries, err := parseMAC(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing MAC address table: %v\n", err)
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
	return "Parses 'show mac address-table' output for Dell OS10"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseMAC(content)
}
