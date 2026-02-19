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
	DataType  string              `json:"data_type"`
	Timestamp string              `json:"timestamp"`
	Date      string              `json:"date"`
	Message   InterfaceStatusData `json:"message"`
}

// InterfaceStatusData represents a single interface status entry
type InterfaceStatusData struct {
	Port        string `json:"port"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Speed       string `json:"speed"`
	Duplex      string `json:"duplex"`
	Mode        string `json:"mode"`
	Vlan        string `json:"vlan"`
	TaggedVlans string `json:"tagged_vlans"`
	IsUp        bool   `json:"is_up"`
}

type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// parseInterfaceStatus parses Dell OS10 show interface status output
func parseInterfaceStatus(content string) ([]StandardizedEntry, error) {
	var entries []StandardizedEntry
	lines := strings.Split(content, "\n")
	timestamp := time.Now().UTC()

	separatorRegex := regexp.MustCompile(`^-{10,}$`)
	portRegex := regexp.MustCompile(`^((?:Eth|Po|Vl|Lo|Ma)\s*\S+)\s+(.+)$`)

	headerSeen := false
	headerPassed := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if separatorRegex.MatchString(trimmed) {
			if headerSeen {
				headerPassed = true
			}
			continue
		}

		if strings.Contains(trimmed, "Port") && strings.Contains(trimmed, "Status") && strings.Contains(trimmed, "Speed") {
			headerSeen = true
			continue
		}

		if headerPassed {
			match := portRegex.FindStringSubmatch(trimmed)
			if match == nil {
				continue
			}

			port := strings.TrimSpace(match[1])
			rest := match[2]
			fields := strings.Fields(rest)
			if len(fields) < 5 {
				continue
			}

			data := InterfaceStatusData{Port: port}

			// Find status field (up/down/admin-down)
			statusIdx := -1
			for i, f := range fields {
				lower := strings.ToLower(f)
				if lower == "up" || lower == "down" || lower == "admin-down" {
					statusIdx = i
					break
				}
			}
			if statusIdx < 0 {
				continue
			}

			if statusIdx > 0 {
				data.Description = strings.Join(fields[:statusIdx], " ")
			}
			data.Status = fields[statusIdx]
			data.IsUp = strings.ToLower(data.Status) == "up"

			remaining := fields[statusIdx+1:]
			if len(remaining) >= 1 {
				data.Speed = remaining[0]
			}
			if len(remaining) >= 2 {
				data.Duplex = remaining[1]
			}
			if len(remaining) >= 3 {
				data.Mode = remaining[2]
			}
			if len(remaining) >= 4 {
				data.Vlan = remaining[3]
			}
			if len(remaining) >= 5 {
				data.TaggedVlans = strings.Join(remaining[4:], ",")
			}

			entry := StandardizedEntry{
				DataType:  "dell_os10_interface_status",
				Timestamp: timestamp.Format(time.RFC3339),
				Date:      timestamp.Format("2006-01-02"),
				Message:   data,
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
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
	inputFile := flag.String("input", "", "Input file containing 'show interface status' output")
	outputFile := flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	commandsFile := flag.String("commands", "", "Commands JSON file")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 Interface Status Parser")
		fmt.Println("Parses 'show interface status' output and converts to JSON format.")
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
		command, err := findCommand(config, "interface-status")
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

	entries, err := parseInterfaceStatus(inputData)
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
	return "Parses 'show interface status' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseInterfaceStatus(string(input))
}
