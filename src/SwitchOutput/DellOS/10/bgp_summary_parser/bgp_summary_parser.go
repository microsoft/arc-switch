package bgp_summary_parser

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
	Message   BGPSummary `json:"message"`
}

// BGPSummary represents the BGP summary data
type BGPSummary struct {
	VRF             string     `json:"vrf"`
	RouterID        string     `json:"router_id"`
	LocalAS         int64      `json:"local_as"`
	ASNType         string     `json:"asn_type"`
	NeighborsCount  int        `json:"neighbors_count"`
	MemoryUsed      int        `json:"memory_used"`
	RoutesToAdd     int        `json:"routes_to_add"`
	RoutesToReplace int        `json:"routes_to_replace"`
	RoutesWithdrawn int        `json:"routes_withdrawn"`
	Neighbors       []Neighbor `json:"neighbors"`
}

// Neighbor represents per-neighbor BGP data
type Neighbor struct {
	NeighborID      string          `json:"neighbor_id"`
	NeighborAS      int64           `json:"neighbor_as"`
	MsgRecvd        int             `json:"msg_recvd"`
	MsgSent         int             `json:"msg_sent"`
	TableVersion    int             `json:"table_version"`
	InQ             int             `json:"inq"`
	OutQ            int             `json:"outq"`
	UpDownTime      string          `json:"up_down_time"`
	TimeParsed      *ParsedDuration `json:"time_parsed,omitempty"`
	State           string          `json:"state"`
	PrefixReceived  int             `json:"prefix_received"`
	SessionType     string          `json:"session_type"`
}

// ParsedDuration represents a parsed duration
type ParsedDuration struct {
	Weeks        int   `json:"weeks"`
	Days         int   `json:"days"`
	Hours        int   `json:"hours"`
	Minutes      int   `json:"minutes"`
	Seconds      int   `json:"seconds"`
	TotalSeconds int64 `json:"total_seconds"`
}

// CommandConfig represents the structure of the commands.json file
type CommandConfig struct {
	Commands []Command `json:"commands"`
}

type Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// classifyASN determines if an ASN is private or public
func classifyASN(asn int64) string {
	if (asn >= 64512 && asn <= 65534) || (asn >= 4200000000 && asn <= 4294967294) {
		return "private"
	}
	return "public"
}

// determineSessionType determines if the session is eBGP or iBGP
func determineSessionType(neighborAS, localAS int64) string {
	if neighborAS == localAS {
		return "iBGP"
	}
	return "eBGP"
}

// parseUpDownTime parses uptime strings like "00:00:00", "1d21h", "11w2d"
func parseUpDownTime(uptime string) (string, *ParsedDuration) {
	if uptime == "" || uptime == "never" {
		return "", nil
	}

	parsed := &ParsedDuration{}

	// Week-day format: "11w2d"
	weekDayRe := regexp.MustCompile(`(\d+)w(\d+)d`)
	if match := weekDayRe.FindStringSubmatch(uptime); len(match) > 2 {
		parsed.Weeks, _ = strconv.Atoi(match[1])
		parsed.Days, _ = strconv.Atoi(match[2])
	}

	// Day-hour format: "1d21h"
	dayHourRe := regexp.MustCompile(`(\d+)d(\d+)h`)
	if match := dayHourRe.FindStringSubmatch(uptime); len(match) > 2 {
		parsed.Days, _ = strconv.Atoi(match[1])
		parsed.Hours, _ = strconv.Atoi(match[2])
	}

	// Hour:minute:second format: "00:00:00"
	hmsRe := regexp.MustCompile(`(\d+):(\d+):(\d+)`)
	if match := hmsRe.FindStringSubmatch(uptime); len(match) > 3 {
		parsed.Hours, _ = strconv.Atoi(match[1])
		parsed.Minutes, _ = strconv.Atoi(match[2])
		parsed.Seconds, _ = strconv.Atoi(match[3])
	}

	parsed.TotalSeconds = int64(parsed.Weeks*7*24*3600 +
		parsed.Days*24*3600 +
		parsed.Hours*3600 +
		parsed.Minutes*60 +
		parsed.Seconds)

	return uptime, parsed
}

// parseInt safely parses an integer from a string
func parseInt(s string) int {
	s = strings.TrimSpace(s)
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// parseInt64 safely parses an int64 from a string
func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// parseBGPSummary parses the show ip bgp summary output for Dell OS10
func parseBGPSummary(input string) ([]StandardizedEntry, error) {
	var entries []StandardizedEntry
	lines := strings.Split(input, "\n")

	timestamp := time.Now().UTC().Format(time.RFC3339)
	date := time.Now().UTC().Format("2006-01-02")

	var currentSummary *BGPSummary
	currentVRF := "default"
	inNeighborTable := false

	// Regular expressions for parsing Dell OS10 BGP output
	vrfRegex := regexp.MustCompile(`VRF:\s+(\S+)`)
	routerIDRegex := regexp.MustCompile(`BGP router identifier\s+([\d.]+),\s+local AS number\s+(\d+)`)
	ribRegex := regexp.MustCompile(`BGP local RIB\s*:\s*Routes to be Added\s+(\d+),\s*Replaced\s+(\d+),\s*Withdrawn\s+(\d+)`)
	neighborsCountRegex := regexp.MustCompile(`(\d+)\s+neighbor\(s\)\s+using\s+(\d+)\s+bytes`)
	neighborHeaderRegex := regexp.MustCompile(`^Neighbor\s+AS\s+MsgRcvd`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Check for VRF
		if match := vrfRegex.FindStringSubmatch(line); len(match) > 1 {
			// Save previous summary if exists
			if currentSummary != nil && len(currentSummary.Neighbors) > 0 {
				entry := StandardizedEntry{
					DataType:  "dell_os10_bgp_summary",
					Timestamp: timestamp,
					Date:      date,
					Message:   *currentSummary,
				}
				entries = append(entries, entry)
			}
			currentVRF = match[1]
			currentSummary = nil
			inNeighborTable = false
			continue
		}

		// Parse router identifier and local AS
		if match := routerIDRegex.FindStringSubmatch(line); len(match) > 2 {
			if currentSummary == nil {
				currentSummary = &BGPSummary{
					VRF:       currentVRF,
					Neighbors: []Neighbor{},
				}
			}
			currentSummary.RouterID = match[1]
			currentSummary.LocalAS = parseInt64(match[2])
			currentSummary.ASNType = classifyASN(currentSummary.LocalAS)
			continue
		}

		// Parse local RIB info
		if match := ribRegex.FindStringSubmatch(line); len(match) > 3 && currentSummary != nil {
			currentSummary.RoutesToAdd = parseInt(match[1])
			currentSummary.RoutesToReplace = parseInt(match[2])
			currentSummary.RoutesWithdrawn = parseInt(match[3])
			continue
		}

		// Parse neighbors count and memory
		if match := neighborsCountRegex.FindStringSubmatch(line); len(match) > 2 && currentSummary != nil {
			currentSummary.NeighborsCount = parseInt(match[1])
			currentSummary.MemoryUsed = parseInt(match[2])
			continue
		}

		// Detect neighbor table header
		if neighborHeaderRegex.MatchString(line) {
			inNeighborTable = true
			continue
		}

		// Parse neighbor lines
		if inNeighborTable && currentSummary != nil {
			fields := strings.Fields(line)
			if len(fields) >= 9 {
				// Validate first field is an IP address
				if !regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(fields[0]) {
					continue
				}

				neighbor := Neighbor{
					NeighborID:   fields[0],
					NeighborAS:   parseInt64(fields[1]),
					MsgRecvd:     parseInt(fields[2]),
					MsgSent:      parseInt(fields[3]),
					TableVersion: parseInt(fields[4]),
					InQ:          parseInt(fields[5]),
					OutQ:         parseInt(fields[6]),
				}

				upDownTime, parsedTime := parseUpDownTime(fields[7])
				neighbor.UpDownTime = upDownTime
				neighbor.TimeParsed = parsedTime

				// State/PfxRcd is the last field
				stateField := fields[8]
				if val, err := strconv.Atoi(stateField); err == nil {
					// It's a number, so the neighbor is Established
					neighbor.State = "Established"
					neighbor.PrefixReceived = val
				} else {
					// It's a state string
					neighbor.State = stateField
					neighbor.PrefixReceived = 0
				}

				neighbor.SessionType = determineSessionType(neighbor.NeighborAS, currentSummary.LocalAS)
				currentSummary.Neighbors = append(currentSummary.Neighbors, neighbor)
			}
		}
	}

	// Save last summary if exists
	if currentSummary != nil {
		entry := StandardizedEntry{
			DataType:  "dell_os10_bgp_summary",
			Timestamp: timestamp,
			Date:      date,
			Message:   *currentSummary,
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no BGP summary data found in input")
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

// findBGPSummaryCommand finds the bgp-summary command in the commands.json
func findBGPSummaryCommand(config *CommandConfig) (string, error) {
	for _, cmd := range config.Commands {
		if cmd.Name == "bgp-summary" {
			return cmd.Command, nil
		}
	}
	return "", fmt.Errorf("bgp-summary command not found in commands file")
}

func main() {
	inputFile := flag.String("input", "", "Input file containing Dell OS10 BGP summary output")
	outputFile := flag.String("output", "", "Output file to write JSON results (default: stdout)")
	commandsFile := flag.String("commands", "", "Path to JSON file containing CLI commands")
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		fmt.Println("Dell OS10 BGP Summary Parser")
		fmt.Println("Parses 'show ip bgp summary' output and converts to JSON format.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  bgp_summary_parser [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input <file>     Input file containing 'show ip bgp summary' output")
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

		bgpCmd, err := findBGPSummaryCommand(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		output, err := runCommand(bgpCmd)
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

	// Parse the BGP summary
	entries, err := parseBGPSummary(inputData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing BGP summary: %v\n", err)
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
	return "Parses 'show ip bgp summary' output for Dell OS10"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseBGPSummary(content)
}
