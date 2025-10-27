package bgp_all_summary_parser

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// StandardizedEntry represents the standardized JSON structure
type StandardizedEntry struct {
	DataType  string     `json:"data_type"`  // Always "cisco_nexus_bgp_summary"
	Timestamp string     `json:"timestamp"`  // ISO 8601 timestamp
	Date      string     `json:"date"`       // Date in YYYY-MM-DD format
	Message   BGPSummary `json:"message"`    // BGP summary-specific data
}

// BGPSummary represents the BGP summary data
type BGPSummary struct {
	VRFNameOut      string          `json:"vrf_name_out"`   // VRF context identification
	VRFRouterID     string          `json:"vrf_router_id"`  // Router ID stability and uniqueness
	VRFLocalAS      int             `json:"vrf_local_as"`   // ASN classification (private/public)
	ASNType         string          `json:"asn_type"`       // "private" or "public"
	AddressFamilies []AddressFamily `json:"address_families"` // Address family data
}

// AddressFamily represents data for a specific address family
type AddressFamily struct {
	AFID              int        `json:"af_id"`                // Address family identifier (1=IPv4, 2=IPv6)
	SAFI              int        `json:"safi"`                 // Subsequent Address Family Identifier (1=Unicast)
	AFName            string     `json:"af_name"`              // Human-readable AF name
	TableVersion      int        `json:"table_version"`        // RIB changes, convergence health
	ConfiguredPeers   int        `json:"configured_peers"`     // Total BGP peers configured
	CapablePeers      int        `json:"capable_peers"`        // Peers in Established state
	TotalNetworks     int        `json:"total_networks"`       // Total unique network prefixes
	TotalPaths        int        `json:"total_paths"`          // Total paths (including ECMP)
	MemoryUsed        int        `json:"memory_used"`          // Memory consumption in bytes
	NumberAttrs       int        `json:"number_attrs"`         // Route attribute count
	BytesAttrs        int        `json:"bytes_attrs"`          // Route attribute bytes
	NumberPaths       int        `json:"number_paths"`         // AS path count
	BytesPaths        int        `json:"bytes_paths"`          // AS path bytes
	NumberCommunities int        `json:"number_communities"`   // Community count
	BytesCommunities  int        `json:"bytes_communities"`    // Community bytes
	NumberClusterList int        `json:"number_clusterlist"`   // Cluster list count
	BytesClusterList  int        `json:"bytes_clusterlist"`    // Cluster list bytes
	Dampening         string     `json:"dampening"`            // Flap suppression status
	Neighbors         []Neighbor `json:"neighbors"`            // Per-neighbor data
}

// Neighbor represents per-neighbor BGP data
type Neighbor struct {
	NeighborID           string          `json:"neighbor_id"`             // BGP neighbor identifier
	NeighborVersion      int             `json:"neighbor_version"`        // BGP protocol version
	MsgRecvd             int             `json:"msg_recvd"`               // Messages received
	MsgSent              int             `json:"msg_sent"`                // Messages sent
	NeighborTableVersion int             `json:"neighbor_table_version"`  // Peer's routing table version
	InQ                  int             `json:"inq"`                     // Input queue depth
	OutQ                 int             `json:"outq"`                    // Output queue depth
	NeighborAS           int             `json:"neighbor_as"`             // Peer's AS number
	Time                 string          `json:"time"`                    // Session uptime (ISO 8601 duration)
	TimeParsed           *ParsedDuration `json:"time_parsed,omitempty"`   // Parsed duration breakdown
	State                string          `json:"state"`                   // BGP session state
	PrefixReceived       int             `json:"prefix_received"`         // Prefixes received from neighbor
	SessionType          string          `json:"session_type"`            // "eBGP" or "iBGP"
}

// ParsedDuration represents a parsed duration
type ParsedDuration struct {
	Weeks        int   `json:"weeks"`
	Days         int   `json:"days"`
	Hours        int   `json:"hours"`
	Minutes      int   `json:"minutes"`
	Seconds      int   `json:"seconds"`
	TotalSeconds int64 `json:"total_seconds"` // Total duration in seconds
}

// classifyASN determines if an ASN is private or public
func classifyASN(asn int) string {
	// Private ASN ranges:
	// 16-bit: 64512-65534
	// 32-bit: 4200000000-4294967294
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
	
	// Parse patterns like: 4d22h, 1w2d, 00:01:23, 1d00h
	// Weeks
	weekRe := regexp.MustCompile(`(\d+)w`)
	if match := weekRe.FindStringSubmatch(uptime); len(match) > 1 {
		parsed.Weeks, _ = strconv.Atoi(match[1])
	}
	
	// Days
	dayRe := regexp.MustCompile(`(\d+)d`)
	if match := dayRe.FindStringSubmatch(uptime); len(match) > 1 {
		parsed.Days, _ = strconv.Atoi(match[1])
	}
	
	// Hours
	hourRe := regexp.MustCompile(`(\d+)h`)
	if match := hourRe.FindStringSubmatch(uptime); len(match) > 1 {
		parsed.Hours, _ = strconv.Atoi(match[1])
	}
	
	// Handle HH:MM:SS format
	timeRe := regexp.MustCompile(`(\d+):(\d+):(\d+)`)
	if match := timeRe.FindStringSubmatch(uptime); len(match) > 3 {
		parsed.Hours, _ = strconv.Atoi(match[1])
		parsed.Minutes, _ = strconv.Atoi(match[2])
		parsed.Seconds, _ = strconv.Atoi(match[3])
	}
	
	// Calculate total seconds
	parsed.TotalSeconds = int64(parsed.Weeks*7*24*3600 +
		parsed.Days*24*3600 +
		parsed.Hours*3600 +
		parsed.Minutes*60 +
		parsed.Seconds)
	
	// Convert to ISO 8601 duration format
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
	
	// Regex patterns
	vrfLineRe := regexp.MustCompile(`BGP summary information for VRF (\S+), address family (.+)`)
	routerIDRe := regexp.MustCompile(`BGP router identifier ([\d.]+), local AS number (\d+)`)
	tableVersionRe := regexp.MustCompile(`BGP table version is (\d+),.*config peers (\d+), capable peers (\d+)`)
	networkEntriesRe := regexp.MustCompile(`(\d+) network entries and (\d+) paths using (\d+) bytes of memory`)
	attrsRe := regexp.MustCompile(`BGP attribute entries \[(\d+)/(\d+)\], BGP AS path entries \[(\d+)/(\d+)\]`)
	communityRe := regexp.MustCompile(`BGP community entries \[(\d+)/(\d+)\], BGP clusterlist entries \[(\d+)/(\d+)\]`)
	neighborHeaderRe := regexp.MustCompile(`^Neighbor\s+V\s+AS\s+MsgRcvd\s+MsgSent`)
	
	inNeighborTable := false
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Empty lines
		if line == "" {
			if currentAF != nil && currentSummary != nil {
				currentSummary.AddressFamilies = append(currentSummary.AddressFamilies, *currentAF)
				currentAF = nil
			}
			if currentSummary != nil && len(currentSummary.AddressFamilies) > 0 {
				entry := StandardizedEntry{
					DataType:  "cisco_nexus_bgp_summary",
					Timestamp: timestamp,
					Date:      date,
					Message:   *currentSummary,
				}
				entries = append(entries, entry)
				currentSummary = nil
			}
			inNeighborTable = false
			continue
		}
		
		// VRF and address family line
		if match := vrfLineRe.FindStringSubmatch(line); len(match) > 2 {
			// Save previous AF if exists
			if currentAF != nil && currentSummary != nil {
				currentSummary.AddressFamilies = append(currentSummary.AddressFamilies, *currentAF)
			}
			
			// If we're starting a new VRF, save previous summary
			vrfName := match[1]
			if currentSummary != nil && currentSummary.VRFNameOut != vrfName {
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
			
			// Create new summary if needed
			if currentSummary == nil {
				currentSummary = &BGPSummary{
					VRFNameOut: vrfName,
				}
			}
			
			// Create new AF
			afName := strings.TrimSpace(match[2])
			currentAF = &AddressFamily{
				AFName:    afName,
				SAFI:      1, // Unicast
				Dampening: "false",
			}
			
			// Set AF ID based on name
			if strings.Contains(strings.ToLower(afName), "ipv6") {
				currentAF.AFID = 2
			} else {
				currentAF.AFID = 1
			}
			
			inNeighborTable = false
			continue
		}
		
		// Router ID and AS line
		if match := routerIDRe.FindStringSubmatch(line); len(match) > 2 && currentSummary != nil {
			currentSummary.VRFRouterID = match[1]
			currentSummary.VRFLocalAS = parseInt(match[2])
			currentSummary.ASNType = classifyASN(currentSummary.VRFLocalAS)
			continue
		}
		
		// Table version line
		if match := tableVersionRe.FindStringSubmatch(line); len(match) > 3 && currentAF != nil {
			currentAF.TableVersion = parseInt(match[1])
			currentAF.ConfiguredPeers = parseInt(match[2])
			currentAF.CapablePeers = parseInt(match[3])
			continue
		}
		
		// Network entries line
		if match := networkEntriesRe.FindStringSubmatch(line); len(match) > 3 && currentAF != nil {
			currentAF.TotalNetworks = parseInt(match[1])
			currentAF.TotalPaths = parseInt(match[2])
			currentAF.MemoryUsed = parseInt(match[3])
			continue
		}
		
		// Attributes line
		if match := attrsRe.FindStringSubmatch(line); len(match) > 4 && currentAF != nil {
			currentAF.NumberAttrs = parseInt(match[1])
			currentAF.BytesAttrs = parseInt(match[2])
			currentAF.NumberPaths = parseInt(match[3])
			currentAF.BytesPaths = parseInt(match[4])
			continue
		}
		
		// Community line
		if match := communityRe.FindStringSubmatch(line); len(match) > 4 && currentAF != nil {
			currentAF.NumberCommunities = parseInt(match[1])
			currentAF.BytesCommunities = parseInt(match[2])
			currentAF.NumberClusterList = parseInt(match[3])
			currentAF.BytesClusterList = parseInt(match[4])
			continue
		}
		
		// Neighbor header
		if neighborHeaderRe.MatchString(line) {
			inNeighborTable = true
			continue
		}
		
		// Parse neighbor entries
		if inNeighborTable && currentAF != nil && currentSummary != nil {
			// Expected format: IP V AS MsgRcvd MsgSent TblVer InQ OutQ Up/Down State/PfxRcd
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
				
				// Up/Down time
				uptime := fields[8]
				iso8601Time, parsedTime := parseUptime(uptime)
				neighbor.Time = iso8601Time
				neighbor.TimeParsed = parsedTime
				
				// State/PfxRcd - could be "Established" or number, or other states
				stateField := fields[9]
				if val, err := strconv.Atoi(stateField); err == nil {
					// It's a number - peer is established
					neighbor.State = "Established"
					neighbor.PrefixReceived = val
				} else {
					// It's a state string
					neighbor.State = stateField
					neighbor.PrefixReceived = 0
				}
				
				// Session type
				neighbor.SessionType = determineSessionType(neighbor.NeighborAS, currentSummary.VRFLocalAS)
				
				currentAF.Neighbors = append(currentAF.Neighbors, neighbor)
			}
		}
	}
	
	// Save any remaining data
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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input_file> [output_file]\n", os.Args[0])
		os.Exit(1)
	}
	
	inputFile := os.Args[1]
	outputFile := ""
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}
	
	// Read input
	data, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}
	
	// Parse the text output
	entries, err := parseBGPTextOutput(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing BGP text output: %v\n", err)
		os.Exit(1)
	}
	
	// Prepare output
	var output *os.File
	if outputFile == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer output.Close()
	}
	
	// Write as JSON array
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
	
	output.Write(jsonData)
	output.Write([]byte("\n"))
	
	fmt.Fprintf(os.Stderr, "Successfully parsed BGP text output\n")
}