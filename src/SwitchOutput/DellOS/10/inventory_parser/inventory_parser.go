package main

import (
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
	DataType  string        `json:"data_type"`
	Timestamp string        `json:"timestamp"`
	Date      string        `json:"date"`
	Message   InventoryData `json:"message"`
}

// InventoryData represents parsed show inventory output
type InventoryData struct {
	Product             string          `json:"product"`
	Description         string          `json:"description"`
	SoftwareVersion     string          `json:"software_version"`
	ProductBase         string          `json:"product_base"`
	ProductSerialNumber string          `json:"product_serial_number"`
	ProductPartNumber   string          `json:"product_part_number"`
	Units               []InventoryUnit `json:"units"`
}

// InventoryUnit represents a hardware unit in the inventory
type InventoryUnit struct {
	UnitID      string `json:"unit_id"`
	Type        string `json:"type"`
	PartNumber  string `json:"part_number"`
	Revision    string `json:"revision"`
	PiecePartID string `json:"piece_part_id"`
	ServiceTag  string `json:"service_tag"`
	ExpressCode string `json:"express_code"`
}

type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseInventory parses Dell OS10 show inventory output
func parseInventory(content string) ([]StandardizedEntry, error) {
	data := InventoryData{}
	lines := strings.Split(content, "\n")
	timestamp := time.Now().UTC()

	kvRegex := regexp.MustCompile(`^(.+?)\s*:\s*(.*)$`)
	separatorRegex := regexp.MustCompile(`^-{10,}$`)
	// Unit line: "* 1  S5248F-ON                006Y6V       A03  TH-006Y6V-CET00-332-60OZ  5M44SR3  122 211 099 03"
	unitRegex := regexp.MustCompile(`^\*?\s*(\d+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.+)$`)

	inUnitTable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if separatorRegex.MatchString(trimmed) {
			inUnitTable = true
			continue
		}

		if inUnitTable {
			if match := unitRegex.FindStringSubmatch(trimmed); match != nil {
				unit := InventoryUnit{
					UnitID:      match[1],
					Type:        match[2],
					PartNumber:  match[3],
					Revision:    match[4],
					PiecePartID: match[5],
					ServiceTag:  match[6],
					ExpressCode: strings.TrimSpace(match[7]),
				}
				data.Units = append(data.Units, unit)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "Unit Type") {
			continue
		}

		if match := kvRegex.FindStringSubmatch(trimmed); match != nil {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			switch key {
			case "Product":
				data.Product = value
			case "Description":
				data.Description = value
			case "Software version":
				data.SoftwareVersion = value
			case "Product Base":
				data.ProductBase = value
			case "Product Serial Number":
				data.ProductSerialNumber = value
			case "Product Part Number":
				data.ProductPartNumber = value
			}
		}
	}

	entry := StandardizedEntry{
		DataType:  "dell_os10_inventory",
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
	inputFile := flag.String("input", "", "Input file containing 'show inventory' output")
	outputFile := flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	commandsFile := flag.String("commands", "", "Commands JSON file")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Inventory Parser")
		fmt.Println("Parses 'show inventory' output and converts to JSON format.")
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
		command, err := findCommand(config, "inventory")
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

	entries, err := parseInventory(inputData)
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
	return "Parses 'show inventory' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseInventory(string(input))
}
