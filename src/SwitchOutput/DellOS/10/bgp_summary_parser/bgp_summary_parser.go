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
	DataType  string         `json:"data_type"`
	Timestamp string         `json:"timestamp"`
	Date      string         `json:"date"`
	Message   BgpSummaryData `json:"message"`
}

// BgpSummaryData represents BGP summary data for a VRF
type BgpSummaryData struct {
	VRF            string        `json:"vrf"`
	RouterID       string        `json:"router_id"`
	LocalAS        int64         `json:"local_as"`
	ASNType        string        `json:"asn_type"`
	NeighborsCount int           `json:"neighbors_count"`
	Neighbors      []BgpNeighbor `json:"neighbors"`
}

// BgpNeighbor represents a single BGP neighbor
type BgpNeighbor struct {
	NeighborID     string `json:"neighbor_id"`
	NeighborAS     int64  `json:"neighbor_as"`
	MsgRecvd       int    `json:"msg_recvd"`
	MsgSent        int    `json:"msg_sent"`
	TableVersion   int    `json:"table_version"`
	InQ            int    `json:"inq"`
	OutQ           int    `json:"outq"`
	UpDownTime     string `json:"up_down_time"`
	State          string `json:"state"`
	PrefixReceived int    `json:"prefix_received"`
	SessionType    string `json:"session_type"`
}

type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

func classifyASN(asn int64) string {
	if (asn >= 64512 && asn <= 65534) || (asn >= 4200000000 && asn <= 4294967294) {
		return "private"
	}
	return "public"
}

// parseBgpSummary parses Dell OS10 show ip bgp summary output
func parseBgpSummary(content string) ([]StandardizedEntry, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return []StandardizedEntry{}, nil
	}

	var entries []StandardizedEntry
	lines := strings.Split(content, "\n")
	timestamp := time.Now().UTC()

	routerIDRegex := regexp.MustCompile(`BGP router identifier\s+([\d.]+),\s+local AS number\s+(\d+)`)
	neighborHeaderRegex := regexp.MustCompile(`^Neighbor\s+AS\s+MsgRcvd`)
	ipRegex := regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)

	var currentSummary *BgpSummaryData
	inNeighborTable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if match := routerIDRegex.FindStringSubmatch(trimmed); match != nil {
			if currentSummary != nil && len(currentSummary.Neighbors) > 0 {
				currentSummary.NeighborsCount = len(currentSummary.Neighbors)
				entry := StandardizedEntry{
					DataType:  "dell_os10_bgp_summary",
					Timestamp: timestamp.Format(time.RFC3339),
					Date:      timestamp.Format("2006-01-02"),
					Message:   *currentSummary,
				}
				entries = append(entries, entry)
			}
			currentSummary = &BgpSummaryData{VRF: "default"}
			currentSummary.RouterID = match[1]
			currentSummary.LocalAS, _ = strconv.ParseInt(match[2], 10, 64)
			currentSummary.ASNType = classifyASN(currentSummary.LocalAS)
			inNeighborTable = false
			continue
		}

		if neighborHeaderRegex.MatchString(trimmed) {
			inNeighborTable = true
			continue
		}

		if inNeighborTable && currentSummary != nil {
			fields := strings.Fields(trimmed)
			if len(fields) >= 9 && ipRegex.MatchString(fields[0]) {
				neighbor := BgpNeighbor{
					NeighborID: fields[0],
					UpDownTime: fields[7],
				}
				neighbor.NeighborAS, _ = strconv.ParseInt(fields[1], 10, 64)
				neighbor.MsgRecvd, _ = strconv.Atoi(fields[2])
				neighbor.MsgSent, _ = strconv.Atoi(fields[3])
				neighbor.TableVersion, _ = strconv.Atoi(fields[4])
				neighbor.InQ, _ = strconv.Atoi(fields[5])
				neighbor.OutQ, _ = strconv.Atoi(fields[6])

				if val, err := strconv.Atoi(fields[8]); err == nil {
					neighbor.State = "Established"
					neighbor.PrefixReceived = val
				} else {
					neighbor.State = fields[8]
				}

				if neighbor.NeighborAS == currentSummary.LocalAS {
					neighbor.SessionType = "iBGP"
				} else {
					neighbor.SessionType = "eBGP"
				}

				currentSummary.Neighbors = append(currentSummary.Neighbors, neighbor)
			}
		}
	}

	if currentSummary != nil {
		currentSummary.NeighborsCount = len(currentSummary.Neighbors)
		entry := StandardizedEntry{
			DataType:  "dell_os10_bgp_summary",
			Timestamp: timestamp.Format(time.RFC3339),
			Date:      timestamp.Format("2006-01-02"),
			Message:   *currentSummary,
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no BGP summary data found in input")
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
	inputFile := flag.String("input", "", "Input file containing 'show ip bgp summary' output")
	outputFile := flag.String("output", "", "Output file for JSON data (optional, defaults to stdout)")
	commandsFile := flag.String("commands", "", "Commands JSON file")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 BGP Summary Parser")
		fmt.Println("Parses 'show ip bgp summary' output and converts to JSON format.")
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
		command, err := findCommand(config, "bgp-summary")
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

	entries, err := parseBgpSummary(inputData)
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
	return "Parses 'show ip bgp summary' output for Dell OS10"
}

func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	return parseBgpSummary(string(input))
}
