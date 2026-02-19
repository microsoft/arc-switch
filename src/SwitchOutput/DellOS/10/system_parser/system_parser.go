package system_parser

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

type SystemData struct {
	NodeID        int           `json:"node_id"`
	MAC           string        `json:"mac"`
	NumberOfMACs  int           `json:"number_of_macs"`
	UpTime        string        `json:"up_time"`
	DiagOS        string        `json:"diag_os"`
	PCIeVersion   string        `json:"pcie_version"`
	Units         []SystemUnit  `json:"units"`
	PowerSupplies []PowerSupply `json:"power_supplies"`
	FanTrays      []FanTray     `json:"fan_trays"`
}

type SystemUnit struct {
	UnitID           int    `json:"unit_id"`
	Status           string `json:"status"`
	SystemIdentifier string `json:"system_identifier"`
	RequiredType     string `json:"required_type"`
	CurrentType      string `json:"current_type"`
	HardwareRevision string `json:"hardware_revision"`
	SoftwareVersion  string `json:"software_version"`
	PhysicalPorts    string `json:"physical_ports"`
}

type PowerSupply struct {
	PSUID     int    `json:"psu_id"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	Power     int    `json:"power_watts"`
	AvgPower  int    `json:"avg_power_watts"`
	AirFlow   string `json:"air_flow"`
	FanSpeed  int    `json:"fan_speed_rpm"`
	FanStatus string `json:"fan_status"`
}

type FanTray struct {
	TrayID  int       `json:"tray_id"`
	Status  string    `json:"status"`
	AirFlow string    `json:"air_flow"`
	Fans    []FanInfo `json:"fans"`
}

type FanInfo struct {
	FanID  int    `json:"fan_id"`
	Speed  int    `json:"speed_rpm"`
	Status string `json:"status"`
}

type SystemParser struct{}

func (p *SystemParser) GetDescription() string {
	return "Parses 'show system' output (hardware, power supplies, fans)"
}

func (p *SystemParser) Parse(input []byte) (interface{}, error) {
	data := SystemData{}
	lines := strings.Split(string(input), "\n")
	kvRegex := regexp.MustCompile(`^(.+?)\s*:\s+(.+)$`)
	sectionRegex := regexp.MustCompile(`^--\s+(.+)\s+--$`)
	separatorRegex := regexp.MustCompile(`^-{10,}$`)
	psuRegex := regexp.MustCompile(`^(\d+)\s+(up|down|absent)\s+(\S+)\s+(\d+)\s+(\d+)\s+\S+\s+(\S+)\s+(\d+)\s+(\d+)\s+(\S+)`)
	fanTrayRegex := regexp.MustCompile(`^(\d+)\s+(up|down|absent)\s+(\S+)\s+(\d+)\s+(\d+)\s+(\S+)`)
	fanContinueRegex := regexp.MustCompile(`^\s+(\d+)\s+(\d+)\s+(\S+)`)

	currentSection := "top"
	var currentUnit *SystemUnit
	var currentFanTray *FanTray

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || separatorRegex.MatchString(trimmed) {
			continue
		}
		if match := sectionRegex.FindStringSubmatch(trimmed); match != nil {
			sectionName := strings.TrimSpace(match[1])
			if currentUnit != nil {
				data.Units = append(data.Units, *currentUnit)
				currentUnit = nil
			}
			if currentFanTray != nil {
				data.FanTrays = append(data.FanTrays, *currentFanTray)
				currentFanTray = nil
			}
			if strings.HasPrefix(sectionName, "Unit") {
				currentSection = "unit"
				currentUnit = &SystemUnit{}
				parts := strings.Fields(sectionName)
				if len(parts) >= 2 {
					currentUnit.UnitID, _ = strconv.Atoi(parts[1])
				}
			} else if strings.Contains(sectionName, "Power") {
				currentSection = "power"
			} else if strings.Contains(sectionName, "Fan") {
				currentSection = "fan"
			}
			continue
		}
		if strings.HasPrefix(trimmed, "PSU-ID") || strings.HasPrefix(trimmed, "FanTray") {
			continue
		}

		switch currentSection {
		case "top":
			if match := kvRegex.FindStringSubmatch(trimmed); match != nil {
				key := strings.TrimSpace(match[1])
				value := strings.TrimSpace(match[2])
				switch key {
				case "Node Id":
					data.NodeID, _ = strconv.Atoi(value)
				case "MAC":
					data.MAC = value
				case "Number of MACs":
					data.NumberOfMACs, _ = strconv.Atoi(value)
				case "Up Time":
					data.UpTime = value
				case "DiagOS":
					data.DiagOS = value
				case "PCIe Version":
					data.PCIeVersion = value
				}
			}
		case "unit":
			if currentUnit != nil {
				if match := kvRegex.FindStringSubmatch(trimmed); match != nil {
					key := strings.TrimSpace(match[1])
					value := strings.TrimSpace(match[2])
					switch key {
					case "Status":
						currentUnit.Status = value
					case "System Identifier":
						currentUnit.SystemIdentifier = value
					case "Required Type":
						currentUnit.RequiredType = value
					case "Current Type":
						currentUnit.CurrentType = value
					case "Hardware Revision":
						currentUnit.HardwareRevision = value
					case "Software Version":
						currentUnit.SoftwareVersion = value
					case "Physical Ports":
						currentUnit.PhysicalPorts = value
					}
				}
			}
		case "power":
			if match := psuRegex.FindStringSubmatch(trimmed); match != nil {
				psu := PowerSupply{Status: match[2], Type: match[3], AirFlow: match[6], FanStatus: match[9]}
				psu.PSUID, _ = strconv.Atoi(match[1])
				psu.Power, _ = strconv.Atoi(match[4])
				psu.AvgPower, _ = strconv.Atoi(match[5])
				psu.FanSpeed, _ = strconv.Atoi(match[8])
				data.PowerSupplies = append(data.PowerSupplies, psu)
			}
		case "fan":
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				if match := fanContinueRegex.FindStringSubmatch(line); match != nil {
					if currentFanTray != nil {
						fan := FanInfo{Status: match[3]}
						fan.FanID, _ = strconv.Atoi(match[1])
						fan.Speed, _ = strconv.Atoi(match[2])
						currentFanTray.Fans = append(currentFanTray.Fans, fan)
					}
					continue
				}
			}
			if match := fanTrayRegex.FindStringSubmatch(trimmed); match != nil {
				if currentFanTray != nil {
					data.FanTrays = append(data.FanTrays, *currentFanTray)
				}
				currentFanTray = &FanTray{Status: match[2], AirFlow: match[3]}
				currentFanTray.TrayID, _ = strconv.Atoi(match[1])
				fan := FanInfo{Status: match[6]}
				fan.FanID, _ = strconv.Atoi(match[4])
				fan.Speed, _ = strconv.Atoi(match[5])
				currentFanTray.Fans = append(currentFanTray.Fans, fan)
			}
		}
	}
	if currentUnit != nil {
		data.Units = append(data.Units, *currentUnit)
	}
	if currentFanTray != nil {
		data.FanTrays = append(data.FanTrays, *currentFanTray)
	}
	now := time.Now().UTC()
	return []StandardizedEntry{{
		DataType: "dell_os10_system", Timestamp: now.Format(time.RFC3339),
		Date: now.Format("2006-01-02"), Message: data,
	}}, nil
}
