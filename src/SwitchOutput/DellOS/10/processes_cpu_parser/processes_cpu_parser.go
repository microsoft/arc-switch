package processes_cpu_parser

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

type ProcessesCpuData struct {
	UnitID         int           `json:"unit_id"`
	OverallCPU5Sec float64      `json:"overall_cpu_5sec_pct"`
	OverallCPU1Min float64      `json:"overall_cpu_1min_pct"`
	OverallCPU5Min float64      `json:"overall_cpu_5min_pct"`
	Processes      []ProcessInfo `json:"processes"`
}

type ProcessInfo struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	RuntimeSec int64   `json:"runtime_seconds"`
	CPU5Sec    float64 `json:"cpu_5sec_pct"`
	CPU1Min    float64 `json:"cpu_1min_pct"`
	CPU5Min    float64 `json:"cpu_5min_pct"`
}

type ProcessesCpuParser struct{}

func (p *ProcessesCpuParser) GetDescription() string {
	return "Parses 'show processes cpu' output"
}

func (p *ProcessesCpuParser) Parse(input []byte) (interface{}, error) {
	data := ProcessesCpuData{}
	lines := strings.Split(string(input), "\n")
	unitRegex := regexp.MustCompile(`CPU Statistics of Unit (\d+)`)
	overallRegex := regexp.MustCompile(`^Overall\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)`)
	processRegex := regexp.MustCompile(`^(\d+)\s+(\S+)\s+(\d+)\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)`)
	inProcessTable := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if match := unitRegex.FindStringSubmatch(trimmed); match != nil {
			data.UnitID, _ = strconv.Atoi(match[1])
			continue
		}
		if match := overallRegex.FindStringSubmatch(trimmed); match != nil {
			data.OverallCPU5Sec, _ = strconv.ParseFloat(match[1], 64)
			data.OverallCPU1Min, _ = strconv.ParseFloat(match[2], 64)
			data.OverallCPU5Min, _ = strconv.ParseFloat(match[3], 64)
			continue
		}
		if strings.HasPrefix(trimmed, "PID") && strings.Contains(trimmed, "Process") {
			inProcessTable = true
			continue
		}
		if inProcessTable {
			if match := processRegex.FindStringSubmatch(trimmed); match != nil {
				proc := ProcessInfo{Name: match[2]}
				proc.PID, _ = strconv.Atoi(match[1])
				proc.RuntimeSec, _ = strconv.ParseInt(match[3], 10, 64)
				proc.CPU5Sec, _ = strconv.ParseFloat(match[4], 64)
				proc.CPU1Min, _ = strconv.ParseFloat(match[5], 64)
				proc.CPU5Min, _ = strconv.ParseFloat(match[6], 64)
				data.Processes = append(data.Processes, proc)
			}
		}
	}
	now := time.Now().UTC()
	return []StandardizedEntry{{
		DataType: "dell_os10_processes_cpu", Timestamp: now.Format(time.RFC3339),
		Date: now.Format("2006-01-02"), Message: data,
	}}, nil
}
