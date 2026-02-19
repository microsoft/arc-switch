package interface_status_parser

import (
	"regexp"
	"strings"
	"time"
)

type StandardizedEntry struct {
	DataType  string      `json:"data_type"`
	Timestamp string      `json:"timestamp"`
	Date      string      `json:"date"`
	Message   interface{} `json:"message"`
}

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

type InterfaceStatusParser struct{}

func (p *InterfaceStatusParser) GetDescription() string {
	return "Parses 'show interface status' output"
}

func (p *InterfaceStatusParser) Parse(input []byte) (interface{}, error) {
	var entries []StandardizedEntry
	lines := strings.Split(string(input), "\n")
	separatorRegex := regexp.MustCompile(`^-{10,}$`)
	portRegex := regexp.MustCompile(`^((?:Eth|Po|Vl|Lo|Ma)\s*\S+)\s+(.+)$`)
	now := time.Now().UTC()

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
			fields := strings.Fields(match[2])
			if len(fields) < 5 {
				continue
			}
			data := InterfaceStatusData{Port: port}

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
			entries = append(entries, StandardizedEntry{
				DataType: "dell_os10_interface_status", Timestamp: now.Format(time.RFC3339),
				Date: now.Format("2006-01-02"), Message: data,
			})
		}
	}
	return entries, nil
}
