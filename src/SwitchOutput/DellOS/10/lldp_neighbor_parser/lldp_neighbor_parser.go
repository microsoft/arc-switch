package lldp_neighbor_parser

import (
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

type LldpNeighborData struct {
	RemoteChassisIDSubtype string `json:"remote_chassis_id_subtype"`
	RemoteChassisID        string `json:"remote_chassis_id"`
	RemotePortSubtype      string `json:"remote_port_subtype"`
	RemotePortID           string `json:"remote_port_id"`
	RemotePortDescription  string `json:"remote_port_description"`
	LocalPortID            string `json:"local_port_id"`
	RemoteNeighborIndex    int    `json:"remote_neighbor_index"`
	RemoteTTL              int    `json:"remote_ttl"`
	RemoteSystemName       string `json:"remote_system_name"`
	RemoteSystemDesc       string `json:"remote_system_desc"`
	RemoteMaxFrameSize     int    `json:"remote_max_frame_size"`
	AutoNegSupported       int    `json:"auto_neg_supported"`
	AutoNegEnabled         int    `json:"auto_neg_enabled"`
}

type LldpParser struct{}

func (p *LldpParser) GetDescription() string {
	return "Parses 'show lldp neighbors detail' output"
}

func (p *LldpParser) Parse(input []byte) (interface{}, error) {
	var entries []StandardizedEntry
	lines := strings.Split(string(input), "\n")
	kvRegex := regexp.MustCompile(`^(.+?):\s+(.+)$`)
	separatorRegex := regexp.MustCompile(`^-{10,}$`)
	now := time.Now().UTC()
	var current *LldpNeighborData

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if separatorRegex.MatchString(trimmed) {
			if current != nil {
				entries = append(entries, StandardizedEntry{
					DataType: "dell_os10_lldp_neighbor", Timestamp: now.Format(time.RFC3339),
					Date: now.Format("2006-01-02"), Message: *current,
				})
			}
			current = nil
			continue
		}
		if match := kvRegex.FindStringSubmatch(trimmed); match != nil {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])
			if current == nil {
				current = &LldpNeighborData{}
			}
			switch key {
			case "Remote Chassis ID Subtype":
				current.RemoteChassisIDSubtype = value
			case "Remote Chassis ID":
				current.RemoteChassisID = value
			case "Remote Port Subtype":
				current.RemotePortSubtype = value
			case "Remote Port ID":
				current.RemotePortID = value
			case "Remote Port Description":
				current.RemotePortDescription = value
			case "Local Port ID":
				current.LocalPortID = value
			case "Locally assigned remote Neighbor Index":
				current.RemoteNeighborIndex, _ = strconv.Atoi(value)
			case "Remote TTL":
				current.RemoteTTL, _ = strconv.Atoi(value)
			case "Remote System Name":
				current.RemoteSystemName = value
			case "Remote System Desc":
				current.RemoteSystemDesc = value
			case "Remote Max Frame Size":
				current.RemoteMaxFrameSize, _ = strconv.Atoi(value)
			case "Auto-neg supported":
				current.AutoNegSupported, _ = strconv.Atoi(value)
			case "Auto-neg enabled":
				current.AutoNegEnabled, _ = strconv.Atoi(value)
			}
		}
	}
	if current != nil {
		entries = append(entries, StandardizedEntry{
			DataType: "dell_os10_lldp_neighbor", Timestamp: now.Format(time.RFC3339),
			Date: now.Format("2006-01-02"), Message: *current,
		})
	}
	return entries, nil
}
