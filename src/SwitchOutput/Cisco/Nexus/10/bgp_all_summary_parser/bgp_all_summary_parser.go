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
	DataType  string     `json:"data_type"`  // Always "cisco_nexus_bgp_summary"
	Timestamp string     `json:"timestamp"`  // ISO 8601 timestamp
	Date      string     `json:"date"`       // Date in YYYY-MM-DD format
	Message   BGPSummary `json:"message"`    // BGP summary-specific data
	Anomalies []string   `json:"anomalies,omitempty"` // List of detected anomalies
}

// BGPSummary represents the BGP summary data
type BGPSummary struct {
	VRFNameOut   string         `json:"vrf_name_out"`   // VRF context identification
	VRFRouterID  string         `json:"vrf_router_id"`  // Router ID stability and uniqueness
	VRFLocalAS   int            `json:"vrf_local_as"`   // ASN classification (private/public)
	ASNType      string         `json:"asn_type"`       // "private" or "public"
	AddressFamilies []AddressFamily `json:"address_families"` // Address family data
}

// AddressFamily represents data for a specific address family
type AddressFamily struct {
	AFID               int        `json:"af_id"`                // Address family identifier (1=IPv4, 2=IPv6)
	SAFI               int        `json:"safi"`                 // Subsequent Address Family Identifier
	AFName             string     `json:"af_name"`              // Human-readable AF name
	TableVersion       int        `json:"table_version"`        // RIB changes, convergence health
	ConfiguredPeers    int        `json:"configured_peers"`     // Total BGP peers configured
	CapablePeers       int        `json:"capable_peers"`        // Peers in Established state
	TotalNetworks      int        `json:"total_networks"`       // Total unique network prefixes
	TotalPaths         int        `json:"total_paths"`          // Total paths (including ECMP)
	MemoryUsed         int        `json:"memory_used"`          // Memory consumption in bytes
	NumberAttrs        int        `json:"number_attrs"`         // Route attribute count
	BytesAttrs         int        `json:"bytes_attrs"`          // Route attribute bytes
	NumberPaths        int        `json:"number_paths"`         // AS path count
	BytesPaths         int        `json:"bytes_paths"`          // AS path bytes
	NumberCommunities  int        `json:"number_communities"`   // Community count
	BytesCommunities   int        `json:"bytes_communities"`    // Community bytes
	NumberClusterList  int        `json:"number_clusterlist"`   // Cluster list count
	BytesClusterList   int        `json:"bytes_clusterlist"`    // Cluster list bytes
	Dampening          string     `json:"dampening"`            // Flap suppression status
	PathDiversityRatio float64    `json:"path_diversity_ratio"` // totalpaths/totalnetworks
	Neighbors          []Neighbor `json:"neighbors"`            // Per-neighbor data
}

// Neighbor represents per-neighbor BGP data
type Neighbor struct {
	NeighborID            string           `json:"neighbor_id"`             // BGP neighbor identifier
	NeighborVersion       int              `json:"neighbor_version"`        // BGP protocol version
	MsgRecvd              int              `json:"msg_recvd"`               // Messages received
	MsgSent               int              `json:"msg_sent"`                // Messages sent
	NeighborTableVersion  int              `json:"neighbor_table_version"`  // Peer's routing table version
	InQ                   int              `json:"inq"`                     // Input queue depth
	OutQ                  int              `json:"outq"`                    // Output queue depth
	NeighborAS            int              `json:"neighbor_as"`             // Peer's AS number
	Time                  string           `json:"time"`                    // Session uptime (ISO 8601 duration)
	TimeParsed            *ParsedDuration  `json:"time_parsed,omitempty"`   // Parsed duration breakdown
	State                 string           `json:"state"`                   // BGP session state
	PrefixReceived        int              `json:"prefix_received"`         // Prefixes received from neighbor
	SessionType           string           `json:"session_type"`            // "eBGP" or "iBGP"
	HealthStatus          string           `json:"health_status"`           // "healthy", "warning", "critical"
	HealthIssues          []string         `json:"health_issues,omitempty"` // List of health issues
}

// ParsedDuration represents a parsed ISO 8601 duration
type ParsedDuration struct {
	Weeks   int `json:"weeks"`
	Days    int `json:"days"`
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
	TotalSeconds int64 `json:"total_seconds"` // Total duration in seconds
}

// NexusBGPInput represents the input JSON structure from Cisco Nexus
type NexusBGPInput struct {
	TableVRF struct {
		RowVRF interface{} `json:"ROW_vrf"` // Can be single object or array
	} `json:"TABLE_vrf"`
}

// VRFData represents VRF-level data
type VRFData struct {
	VRFNameOut  string      `json:"vrf-name-out"`
	VRFRouterID string      `json:"vrf-router-id"`
	VRFLocalAS  json.Number `json:"vrf-local-as"`
	TableAF     struct {
		RowAF interface{} `json:"ROW_af"` // Can be single object or array
	} `json:"TABLE_af"`
}

// AFData represents address family data
type AFData struct {
	AFID              json.Number `json:"af-id"`
	SAFI              json.Number `json:"safi"`
	AFName            string      `json:"af-name"`
	TableVersion      json.Number `json:"tableversion"`
	ConfiguredPeers   json.Number `json:"configuredpeers"`
	CapablePeers      json.Number `json:"capablepeers"`
	TotalNetworks     json.Number `json:"totalnetworks"`
	TotalPaths        json.Number `json:"totalpaths"`
	MemoryUsed        json.Number `json:"memoryused"`
	NumberAttrs       json.Number `json:"numberattrs"`
	BytesAttrs        json.Number `json:"bytesattrs"`
	NumberPaths       json.Number `json:"numberpaths"`
	BytesPaths        json.Number `json:"bytespaths"`
	NumberCommunities json.Number `json:"numbercommunities"`
	BytesCommunities  json.Number `json:"bytescommunities"`
	NumberClusterList json.Number `json:"numberclusterlist"`
	BytesClusterList  json.Number `json:"bytesclusterlist"`
	Dampening         string      `json:"dampening"`
	TableNeighbor     struct {
		RowNeighbor interface{} `json:"ROW_neighbor"` // Can be single object or array
	} `json:"TABLE_neighbor"`
}

// NeighborData represents neighbor data
type NeighborData struct {
	NeighborID           string      `json:"neighborid"`
	NeighborVersion      json.Number `json:"neighborversion"`
	MsgRecvd             json.Number `json:"msgrecvd"`
	MsgSent              json.Number `json:"msgsent"`
	NeighborTableVersion json.Number `json:"neighbortableversion"`
	InQ                  json.Number `json:"inq"`
	OutQ                 json.Number `json:"outq"`
	NeighborAS           json.Number `json:"neighboras"`
	Time                 string      `json:"time"`
	State                string      `json:"state"`
	PrefixReceived       json.Number `json:"prefixreceived"`
}

// parseISO8601Duration parses ISO 8601 duration format (e.g., "P14W1D", "P37W6D")
func parseISO8601Duration(duration string) *ParsedDuration {
	if duration == "" || duration == "never" {
		return nil
	}

	parsed := &ParsedDuration{}
	
	// Pattern for ISO 8601 duration: PnYnMnDTnHnMnS or simplified PnWnD
	// We'll handle common Cisco formats like P14W1D, P37W6D
	
	// Remove 'P' prefix
	if !strings.HasPrefix(duration, "P") {
		return nil
	}
	duration = strings.TrimPrefix(duration, "P")
	
	// Split by 'T' to separate date and time components
	parts := strings.Split(duration, "T")
	datePart := parts[0]
	timePart := ""
	if len(parts) > 1 {
		timePart = parts[1]
	}
	
	// Parse date part
	dateRegex := regexp.MustCompile(`(\d+)([YMWD])`)
	for _, match := range dateRegex.FindAllStringSubmatch(datePart, -1) {
		value, _ := strconv.Atoi(match[1])
		unit := match[2]
		switch unit {
		case "W":
			parsed.Weeks = value
		case "D":
			parsed.Days = value
		}
	}
	
	// Parse time part
	if timePart != "" {
		timeRegex := regexp.MustCompile(`(\d+)([HMS])`)
		for _, match := range timeRegex.FindAllStringSubmatch(timePart, -1) {
			value, _ := strconv.Atoi(match[1])
			unit := match[2]
			switch unit {
			case "H":
				parsed.Hours = value
			case "M":
				parsed.Minutes = value
			case "S":
				parsed.Seconds = value
			}
		}
	}
	
	// Calculate total seconds
	parsed.TotalSeconds = int64(parsed.Weeks*7*24*3600 + 
		parsed.Days*24*3600 + 
		parsed.Hours*3600 + 
		parsed.Minutes*60 + 
		parsed.Seconds)
	
	return parsed
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

// jsonNumberToInt converts json.Number to int, returns 0 if conversion fails
func jsonNumberToInt(n json.Number) int {
	if n == "" {
		return 0
	}
	val, err := n.Int64()
	if err != nil {
		return 0
	}
	return int(val)
}

// analyzeNeighborHealth determines neighbor health status and issues
func analyzeNeighborHealth(neighbor *Neighbor, tableVersion int, localAS int) {
	neighbor.HealthStatus = "healthy"
	neighbor.HealthIssues = []string{}
	
	// Determine session type
	if neighbor.NeighborAS == localAS {
		neighbor.SessionType = "iBGP"
	} else {
		neighbor.SessionType = "eBGP"
	}
	
	// Check for critical issues
	if neighbor.State != "Established" {
		neighbor.HealthStatus = "critical"
		neighbor.HealthIssues = append(neighbor.HealthIssues, fmt.Sprintf("session_state_%s", strings.ToLower(neighbor.State)))
	}
	
	if neighbor.InQ > 0 {
		neighbor.HealthStatus = "critical"
		neighbor.HealthIssues = append(neighbor.HealthIssues, fmt.Sprintf("input_queue_depth_%d", neighbor.InQ))
	}
	
	if neighbor.OutQ > 0 {
		if neighbor.HealthStatus == "healthy" {
			neighbor.HealthStatus = "warning"
		}
		neighbor.HealthIssues = append(neighbor.HealthIssues, fmt.Sprintf("output_queue_depth_%d", neighbor.OutQ))
	}
	
	if neighbor.State == "Established" && neighbor.PrefixReceived == 0 {
		if neighbor.HealthStatus == "healthy" {
			neighbor.HealthStatus = "warning"
		}
		neighbor.HealthIssues = append(neighbor.HealthIssues, "no_prefixes_received")
	}
	
	// Check for table version mismatch
	if neighbor.State == "Established" && neighbor.NeighborTableVersion != 0 && 
	   neighbor.NeighborTableVersion != tableVersion {
		if neighbor.HealthStatus == "healthy" {
			neighbor.HealthStatus = "warning"
		}
		neighbor.HealthIssues = append(neighbor.HealthIssues, 
			fmt.Sprintf("table_version_mismatch_local_%d_neighbor_%d", 
				tableVersion, neighbor.NeighborTableVersion))
	}
}

// detectAnomalies identifies system-level anomalies
func detectAnomalies(summary *BGPSummary) []string {
	anomalies := []string{}
	
	for _, af := range summary.AddressFamilies {
		// Check for peer status mismatch
		if af.CapablePeers < af.ConfiguredPeers {
			anomalies = append(anomalies, 
				fmt.Sprintf("%s: capable_peers(%d) < configured_peers(%d)", 
					af.AFName, af.CapablePeers, af.ConfiguredPeers))
		}
		
		// Check for excessive dependency on single peer
		if len(af.Neighbors) > 1 && af.TotalNetworks > 0 {
			for _, neighbor := range af.Neighbors {
				if neighbor.State == "Established" && neighbor.PrefixReceived > 0 {
					dependency := float64(neighbor.PrefixReceived) / float64(af.TotalNetworks) * 100
					if dependency > 50 {
						anomalies = append(anomalies, 
							fmt.Sprintf("%s: excessive_dependency_on_peer_%s_%.1f%%", 
								af.AFName, neighbor.NeighborID, dependency))
					}
				}
			}
		}
		
		// Check for any critical neighbor issues
		criticalCount := 0
		for _, neighbor := range af.Neighbors {
			if neighbor.HealthStatus == "critical" {
				criticalCount++
			}
		}
		if criticalCount > 0 {
			anomalies = append(anomalies, 
				fmt.Sprintf("%s: %d_neighbors_in_critical_state", af.AFName, criticalCount))
		}
	}
	
	return anomalies
}

// parseBGPSummary parses the BGP all summary JSON input
func parseBGPSummary(input string) ([]StandardizedEntry, error) {
	var nexusInput NexusBGPInput
	
	if err := json.Unmarshal([]byte(input), &nexusInput); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	
	// Get current timestamp
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	date := now.Format("2006-01-02")
	
	var entries []StandardizedEntry
	
	// Handle ROW_vrf as either single object or array
	vrfList := []VRFData{}
	switch v := nexusInput.TableVRF.RowVRF.(type) {
	case map[string]interface{}:
		var vrf VRFData
		data, _ := json.Marshal(v)
		if err := json.Unmarshal(data, &vrf); err == nil {
			vrfList = append(vrfList, vrf)
		}
	case []interface{}:
		for _, item := range v {
			var vrf VRFData
			data, _ := json.Marshal(item)
			if err := json.Unmarshal(data, &vrf); err == nil {
				vrfList = append(vrfList, vrf)
			}
		}
	}
	
	// Process each VRF
	for _, vrf := range vrfList {
		summary := BGPSummary{
			VRFNameOut:  vrf.VRFNameOut,
			VRFRouterID: vrf.VRFRouterID,
			VRFLocalAS:  jsonNumberToInt(vrf.VRFLocalAS),
		}
		summary.ASNType = classifyASN(summary.VRFLocalAS)
		
		// Handle ROW_af as either single object or array
		afList := []AFData{}
		switch v := vrf.TableAF.RowAF.(type) {
		case map[string]interface{}:
			var af AFData
			data, _ := json.Marshal(v)
			if err := json.Unmarshal(data, &af); err == nil {
				afList = append(afList, af)
			}
		case []interface{}:
			for _, item := range v {
				var af AFData
				data, _ := json.Marshal(item)
				if err := json.Unmarshal(data, &af); err == nil {
					afList = append(afList, af)
				}
			}
		}
		
		// Process each address family
		for _, afData := range afList {
			af := AddressFamily{
				AFID:              jsonNumberToInt(afData.AFID),
				SAFI:              jsonNumberToInt(afData.SAFI),
				AFName:            afData.AFName,
				TableVersion:      jsonNumberToInt(afData.TableVersion),
				ConfiguredPeers:   jsonNumberToInt(afData.ConfiguredPeers),
				CapablePeers:      jsonNumberToInt(afData.CapablePeers),
				TotalNetworks:     jsonNumberToInt(afData.TotalNetworks),
				TotalPaths:        jsonNumberToInt(afData.TotalPaths),
				MemoryUsed:        jsonNumberToInt(afData.MemoryUsed),
				NumberAttrs:       jsonNumberToInt(afData.NumberAttrs),
				BytesAttrs:        jsonNumberToInt(afData.BytesAttrs),
				NumberPaths:       jsonNumberToInt(afData.NumberPaths),
				BytesPaths:        jsonNumberToInt(afData.BytesPaths),
				NumberCommunities: jsonNumberToInt(afData.NumberCommunities),
				BytesCommunities:  jsonNumberToInt(afData.BytesCommunities),
				NumberClusterList: jsonNumberToInt(afData.NumberClusterList),
				BytesClusterList:  jsonNumberToInt(afData.BytesClusterList),
				Dampening:         afData.Dampening,
			}
			
			// Calculate path diversity ratio
			if af.TotalNetworks > 0 {
				af.PathDiversityRatio = float64(af.TotalPaths) / float64(af.TotalNetworks)
			}
			
			// Handle ROW_neighbor as either single object or array
			neighborList := []NeighborData{}
			switch v := afData.TableNeighbor.RowNeighbor.(type) {
			case map[string]interface{}:
				var nbr NeighborData
				data, _ := json.Marshal(v)
				if err := json.Unmarshal(data, &nbr); err == nil {
					neighborList = append(neighborList, nbr)
				}
			case []interface{}:
				for _, item := range v {
					var nbr NeighborData
					data, _ := json.Marshal(item)
					if err := json.Unmarshal(data, &nbr); err == nil {
						neighborList = append(neighborList, nbr)
					}
				}
			}
			
			// Process neighbors
			for _, nbrData := range neighborList {
				neighbor := Neighbor{
					NeighborID:           nbrData.NeighborID,
					NeighborVersion:      jsonNumberToInt(nbrData.NeighborVersion),
					MsgRecvd:             jsonNumberToInt(nbrData.MsgRecvd),
					MsgSent:              jsonNumberToInt(nbrData.MsgSent),
					NeighborTableVersion: jsonNumberToInt(nbrData.NeighborTableVersion),
					InQ:                  jsonNumberToInt(nbrData.InQ),
					OutQ:                 jsonNumberToInt(nbrData.OutQ),
					NeighborAS:           jsonNumberToInt(nbrData.NeighborAS),
					Time:                 nbrData.Time,
					State:                nbrData.State,
					PrefixReceived:       jsonNumberToInt(nbrData.PrefixReceived),
				}
				
				// Parse duration
				neighbor.TimeParsed = parseISO8601Duration(neighbor.Time)
				
				// Analyze neighbor health
				analyzeNeighborHealth(&neighbor, af.TableVersion, summary.VRFLocalAS)
				
				af.Neighbors = append(af.Neighbors, neighbor)
			}
			
			summary.AddressFamilies = append(summary.AddressFamilies, af)
		}
		
		// Detect anomalies
		anomalies := detectAnomalies(&summary)
		
		entry := StandardizedEntry{
			DataType:  "cisco_nexus_bgp_summary",
			Timestamp: timestamp,
			Date:      date,
			Message:   summary,
			Anomalies: anomalies,
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
	inputFile := flag.String("input", "", "Input file containing Cisco Nexus BGP summary JSON output")
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
	return "Parses 'show bgp all summary | json' output"
}

// Parse implements the Parser interface for unified binary
func (p *UnifiedParser) Parse(input []byte) (interface{}, error) {
	content := string(input)
	return parseBGPSummary(content)
}
