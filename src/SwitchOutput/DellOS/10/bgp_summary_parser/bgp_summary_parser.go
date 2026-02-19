package bgp_summary_parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type StandardizedEntry struct {
	DataType  string      `json:"data_type"`
	Timestamp string      `json:"timestamp"`
	Date      string      `json:"date"`
	Message   interface{} `json:"message"`
}

type BgpSummaryData struct {
	VRF            string        `json:"vrf"`
	RouterID       string        `json:"router_id"`
	LocalAS        int64         `json:"local_as"`
	ASNType        string        `json:"asn_type"`
	NeighborsCount int           `json:"neighbors_count"`
	Neighbors      []BgpNeighbor `json:"neighbors"`
}

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

type BgpSummaryParser struct{}

func (p *BgpSummaryParser) GetDescription() string {
	return "Parses 'show ip bgp summary' output"
}

func ClassifyASN(asn int64) string {
	if (asn >= 64512 && asn <= 65534) || (asn >= 4200000000 && asn <= 4294967294) {
		return "private"
	}
	return "public"
}

func (p *BgpSummaryParser) Parse(input []byte) (interface{}, error) {
	content := strings.TrimSpace(string(input))
	if content == "" {
		return []StandardizedEntry{}, nil
	}

	var entries []StandardizedEntry
	lines := strings.Split(content, "\n")
	now := time.Now().UTC()
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
				entries = append(entries, StandardizedEntry{
					DataType: "dell_os10_bgp_summary", Timestamp: now.Format(time.RFC3339),
					Date: now.Format("2006-01-02"), Message: *currentSummary,
				})
			}
			currentSummary = &BgpSummaryData{VRF: "default"}
			currentSummary.RouterID = match[1]
			currentSummary.LocalAS, _ = strconv.ParseInt(match[2], 10, 64)
			currentSummary.ASNType = ClassifyASN(currentSummary.LocalAS)
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
				neighbor := BgpNeighbor{NeighborID: fields[0], UpDownTime: fields[7]}
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
		entries = append(entries, StandardizedEntry{
			DataType: "dell_os10_bgp_summary", Timestamp: now.Format(time.RFC3339),
			Date: now.Format("2006-01-02"), Message: *currentSummary,
		})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no BGP summary data found in input")
	}
	return entries, nil
}
