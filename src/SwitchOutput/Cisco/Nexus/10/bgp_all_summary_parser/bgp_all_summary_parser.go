package bgp_all_summary_parser

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
	VRFNameOut      string          `json:"vrf_name_out"`
	VRFRouterID     string          `json:"vrf_router_id"`
	VRFLocalAS      int             `json:"vrf_local_as"`
	ASNType         string          `json:"asn_type"`
	AddressFamilies []AddressFamily `json:"address_families"`
}

// AddressFamily represents data for a specific address family
type AddressFamily struct {
	AFID              int        `json:"af_id"`
	SAFI              int        `json:"safi"`
	AFName            string     `json:"af_name"`
	TableVersion      int        `json:"table_version"`
	ConfiguredPeers   int        `json:"configured_peers"`
	CapablePeers      int        `json:"capable_peers"`
	TotalNetworks     int        `json:"total_networks"`
	TotalPaths        int        `json:"total_paths"`
	MemoryUsed        int        `json:"memory_used"`
	NumberAttrs       int        `json:"number_attrs"`
	BytesAttrs        int        `json:"bytes_attrs"`
	NumberPaths       int        `json:"number_paths"`
	BytesPaths        int        `json:"bytes_paths"`
	NumberCommunities int        `json:"number_communities"`
	BytesCommunities  int        `json:"bytes_communities"`
	NumberClusterList int        `json:"number_clusterlist"`
	BytesClusterList  int        `json:"bytes_clusterlist"`
	Dampening         string     `json:"dampening"`
	Neighbors         []Neighbor `json:"neighbors"`
}

// Neighbor represents per-neighbor BGP data
type Neighbor struct {
	NeighborID           string          `json:"neighbor_id"`
	NeighborVersion      int             `json:"neighbor_version"`
	MsgRecvd             int             `json:"msg_recvd"`
	MsgSent              int             `json:"msg_sent"`
	NeighborTableVersion int             `json:"neighbor_table_version"`
	InQ                  int             `json:"inq"`
	OutQ                 int             `json:"outq"`
	NeighborAS           int             `json:"neighbor_as"`
	Time                 string          `json:"time"`
	TimeParsed           *ParsedDuration `json:"time_parsed,omitempty"`
	State                string          `json:"state"`
	PrefixReceived       int             `json:"prefix_received"`
	SessionType          string          `json:"session_type"`
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

// classifyASN determines if an ASN is private or public
func classifyASN(asn int) string {
	if (asn >= 64512 && asn <= 65534) || (asn >= 4200000000 && asn <= 4294967294) {
		return "private"
	}
	return "public"
}

// determineSessionType determines if the session is eBGP or iBGP
func determineSessionType(neighborAS, localAS int) string {
	if neighborAS == localAS {
		return "iBGP"
	}
	return "eBGP"
}

// parseUptime converts text uptime like "4d22h" to ISO 8601 duration
func parseUptime(uptime string) (string, *ParsedDuration) {
	if uptime == "" || uptime == "never" {
		return "", nil
	}

	parsed := &ParsedDuration{}

	weekRe := regexp.MustCompile(`(\d+)w`)
	if match := weekRe.FindStringSubmatch(uptime); len(match) > 1 {
		parsed.Weeks, _ = strconv.Atoi(match[1])
	}

	dayRe := regexp.MustCompile(`(\d+)d`)
	if match := dayRe.FindStringSubmatch(uptime); len(match) > 1 {
		parsed.Days, _ = strconv.Atoi(match[1])
	}

	hourRe := regexp.MustCompile(`(\d+)h`)
	if match := hourRe.FindStringSubmatch(uptime); len(match) > 1 {
		parsed.Hours, _ = strconv.Atoi(match[1])
	}

	timeRe := regexp.MustCompile(`(\d+):(\d+):(\d+)`)
	if match := timeRe.FindStringSubmatch(uptime); len(match) > 3 {
		parsed.Hours, _ = strconv.Atoi(match[1])
		parsed.Minutes, _ = strconv.Atoi(match[2])
		parsed.Seconds, _ = strconv.Atoi(match[3])
	}

	parsed.TotalSeconds = int64(parsed.Weeks*7*24*3600 +
		parsed.Days*24*3600 +
		parsed.Hours*3600 +
		parsed.Minutes*60 +
		parsed.Seconds)

	iso8601 := "P"
	if parsed.Weeks > 0 {
		iso8601 += fmt.Sprintf("%dW", parsed.Weeks)
	}
	if parsed.Days > 0 {
		iso8601 += fmt.Sprintf("%dD", parsed.Days)
	}
	if parsed.Hours > 0 || parsed.Minutes > 0 || parsed.Seconds > 0 {
		iso8601 += "T"
		if parsed.Hours > 0 {
			iso8601 += fmt.Sprintf("%dH", parsed.Hours)
		}
		if parsed.Minutes > 0 {
			iso8601 += fmt.Sprintf("%dM", parsed.Minutes)
		}
		if parsed.Seconds > 0 {
			iso8601 += fmt.Sprintf("%dS", parsed.Seconds)
		}
	}

	return iso8601, parsed
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

// parseBGPTextOutput parses the text output from "show bgp all summary"
func parseBGPTextOutput(input string) ([]StandardizedEntry, error) {
	var entries []StandardizedEntry
	lines := strings.Split(input, "\n")

	timestamp := time.Now().UTC().Format(time.RFC3339)
	date := time.Now().UTC().Format("2006-01-02")

	var currentSummary *BGPSummary
	var currentAF *AddressFamily

	vrfLineRe := regexp.MustCompile(`BGP summary information for VRF (\S+), address family (.+)`)
	routerIDRe := regexp.MustCompile(`BGP router identifier ([\d.]+), local AS number (\d+)`)
	tableVersionRe := regexp.MustCompile(`BGP table version is (\d+),.*config peers (\d+), capable peers (\d+)`)
	networkEntriesRe := regexp.MustCompile(`(\d+) network entries and (\d+) paths using (\d+) bytes of memory`)
	attrsRe := regexp.MustCompile(`BGP attribute entries \[(\d+)/(\d+)\], BGP AS path entries \[(\d+)/(\d+)\]`)
	communityRe := regexp.MustCompile(`BGP community entries \[(\d+)/(\d+)\], BGP clusterlist entries \[(\d+)/(\d+)\]`)
	neighborHeaderRe := regexp.MustCompile(`^Neighbor\s+V\s+AS\s+MsgRcvd\s+MsgSent`)

	inNeighborTable := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// ✅ FIXED: Don't finalize on empty lines - neighbors come after empty lines!
		if line == "" {
			// Just continue parsing - don't reset currentAF or finalize entries yet
			continue
		}

		if match := vrfLineRe.FindStringSubmatch(line); len(match) > 2 {
			// Before processing new AF, finalize the PREVIOUS AF (which now has its neighbors!)
			if currentAF != nil && currentSummary != nil {
				currentSummary.AddressFamilies = append(currentSummary.AddressFamilies, *currentAF)
				currentAF = nil
			}

			vrfName := match[1]
			// Check if we're starting a new VRF
			if currentSummary != nil && currentSummary.VRFNameOut != vrfName {
				// Finalize previous VRF entry
				if len(currentSummary.AddressFamilies) > 0 {
					entry := StandardizedEntry{
						DataType:  "cisco_nexus_bgp_summary",
						Timestamp: timestamp,
						Date:      date,
						Message:   *currentSummary,
					}
					entries = append(entries, entry)
				}
				currentSummary = nil
			}

			// Initialize new VRF if needed
			if currentSummary == nil {
				currentSummary = &BGPSummary{
					VRFNameOut: vrfName,
				}
			}

			// Initialize new address family
			afName := strings.TrimSpace(match[2])
			currentAF = &AddressFamily{
				AFName:    afName,
				SAFI:      1,
				Dampening: "false",
				Neighbors: []Neighbor{}, // ✅ Initialize empty neighbors slice
			}

			if strings.Contains(strings.ToLower(afName), "ipv6") {
				currentAF.AFID = 2
			} else {
				currentAF.AFID = 1
			}

			inNeighborTable = false
			continue
		}

		if match := routerIDRe.FindStringSubmatch(line); len(match) > 2 && currentSummary != nil {
			currentSummary.VRFRouterID = match[1]
			currentSummary.VRFLocalAS = parseInt(match[2])
			currentSummary.ASNType = classifyASN(currentSummary.VRFLocalAS)
			continue
		}

		if match := tableVersionRe.FindStringSubmatch(line); len(match) > 3 && currentAF != nil {
			currentAF.TableVersion = parseInt(match[1])
			currentAF.ConfiguredPeers = parseInt(match[2])
			currentAF.CapablePeers = parseInt(match[3])
			continue
		}

		if match := networkEntriesRe.FindStringSubmatch(line); len(match) > 3 && currentAF != nil {
			currentAF.TotalNetworks = parseInt(match[1])
			currentAF.TotalPaths = parseInt(match[2])
			currentAF.MemoryUsed = parseInt(match[3])
			continue
		}

		if match := attrsRe.FindStringSubmatch(line); len(match) > 4 && currentAF != nil {
			currentAF.NumberAttrs = parseInt(match[1])
			currentAF.BytesAttrs = parseInt(match[2])
			currentAF.NumberPaths = parseInt(match[3])
			currentAF.BytesPaths = parseInt(match[4])
			continue
		}

		if match := communityRe.FindStringSubmatch(line); len(match) > 4 && currentAF != nil {
			currentAF.NumberCommunities = parseInt(match[1])
			currentAF.BytesCommunities = parseInt(match[2])
			currentAF.NumberClusterList = parseInt(match[3])
			currentAF.BytesClusterList = parseInt(match[4])
			continue
		}

		if neighborHeaderRe.MatchString(line) {
			inNeighborTable = true
			continue
		}

		// ✅ FIXED: This now works because currentAF is NOT nil!
		if inNeighborTable && currentAF != nil && currentSummary != nil {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				neighbor := Neighbor{
					NeighborID:           fields[0],
					NeighborVersion:      parseInt(fields[1]),
					NeighborAS:           parseInt(fields[2]),
					MsgRecvd:             parseInt(fields[3]),
					MsgSent:              parseInt(fields[4]),
					NeighborTableVersion: parseInt(fields[5]),
					InQ:                  parseInt(fields[6]),
					OutQ:                 parseInt(fields[7]),
				}

				uptime := fields[8]
				iso8601Time, parsedTime := parseUptime(uptime)
				neighbor.Time = iso8601Time
				neighbor.TimeParsed = parsedTime

				stateField := fields[9]
				if val, err := strconv.Atoi(stateField); err == nil {
					neighbor.State = "Established"
					neighbor.PrefixReceived = val
				} else {
					neighbor.State = stateField
					neighbor.PrefixReceived = 0
				}

				neighbor.SessionType = determineSessionType(neighbor.NeighborAS, currentSummary.VRFLocalAS)

				currentAF.Neighbors = append(currentAF.Neighbors, neighbor)
			}
		}
	}

	if currentAF != nil && currentSummary != nil {
		currentSummary.AddressFamilies = append(currentSummary.AddressFamilies, *currentAF)
	}
	if currentSummary != nil && len(currentSummary.AddressFamilies) > 0 {
		entry := StandardizedEntry{
			DataType:  "cisco_nexus_bgp_summary",
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
	inputFile := flag.String("input", "", "Input file containing Cisco Nexus BGP summary output")
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
		var bgpCmd string
		for _, c := range cmdFile.Commands {
			if c.Name == "bgp-all-summary" {
				bgpCmd = c.Command
				break
			}
		}
		if bgpCmd == "" {
			fmt.Fprintln(os.Stderr, "Error: No 'bgp-all-summary' command found in commands JSON.")
			os.Exit(1)
		}
		// Run the command using vsh
		vshOut, err := runVsh(bgpCmd)
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

	// Parse the BGP summary
	entries, err := parseBGPTextOutput(inputData)
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
	return "Parses 'show bgp all summary' output (TEXT format)"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseBGPTextOutput(content)
}