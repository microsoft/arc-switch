package system_uptime_parser

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

type UptimeData struct {
	RawUptime    string `json:"raw_uptime"`
	Weeks        int    `json:"weeks"`
	Days         int    `json:"days"`
	Hours        int    `json:"hours"`
	Minutes      int    `json:"minutes"`
	Seconds      int    `json:"seconds"`
	TotalSeconds int64  `json:"total_seconds"`
}

type UptimeParser struct{}

func (p *UptimeParser) GetDescription() string {
	return "Parses 'show uptime' output"
}

func (p *UptimeParser) Parse(input []byte) (interface{}, error) {
	raw := strings.TrimSpace(string(input))
	data := UptimeData{RawUptime: raw}

	if match := regexp.MustCompile(`(\d+)\s+weeks?`).FindStringSubmatch(raw); match != nil {
		data.Weeks, _ = strconv.Atoi(match[1])
	}
	if match := regexp.MustCompile(`(\d+)\s+days?`).FindStringSubmatch(raw); match != nil {
		data.Days, _ = strconv.Atoi(match[1])
	}
	if match := regexp.MustCompile(`(\d+):(\d+):(\d+)`).FindStringSubmatch(raw); match != nil {
		data.Hours, _ = strconv.Atoi(match[1])
		data.Minutes, _ = strconv.Atoi(match[2])
		data.Seconds, _ = strconv.Atoi(match[3])
	}
	data.TotalSeconds = int64(data.Weeks)*7*24*3600 +
		int64(data.Days)*24*3600 +
		int64(data.Hours)*3600 +
		int64(data.Minutes)*60 +
		int64(data.Seconds)

	now := time.Now().UTC()
	return []StandardizedEntry{{
		DataType: "dell_os10_uptime", Timestamp: now.Format(time.RFC3339),
		Date: now.Format("2006-01-02"), Message: data,
	}}, nil
}
