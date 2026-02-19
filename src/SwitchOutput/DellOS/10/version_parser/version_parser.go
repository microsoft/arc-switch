package version_parser

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

type VersionData struct {
	OSName       string `json:"os_name"`
	OSVersion    string `json:"os_version"`
	BuildVersion string `json:"build_version"`
	BuildTime    string `json:"build_time"`
	SystemType   string `json:"system_type"`
	Architecture string `json:"architecture"`
	UpTime       string `json:"up_time"`
}

type VersionParser struct{}

func (p *VersionParser) GetDescription() string {
	return "Parses 'show version' output"
}

func (p *VersionParser) Parse(input []byte) (interface{}, error) {
	data := VersionData{}
	lines := strings.Split(string(input), "\n")
	kvRegex := regexp.MustCompile(`^(.+?):\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Dell ") && data.OSName == "" {
			data.OSName = line
			continue
		}
		if match := kvRegex.FindStringSubmatch(line); match != nil {
			switch strings.TrimSpace(match[1]) {
			case "OS Version":
				data.OSVersion = strings.TrimSpace(match[2])
			case "Build Version":
				data.BuildVersion = strings.TrimSpace(match[2])
			case "Build Time":
				data.BuildTime = strings.TrimSpace(match[2])
			case "System Type":
				data.SystemType = strings.TrimSpace(match[2])
			case "Architecture":
				data.Architecture = strings.TrimSpace(match[2])
			case "Up Time":
				data.UpTime = strings.TrimSpace(match[2])
			}
		}
	}

	now := time.Now().UTC()
	return []StandardizedEntry{{
		DataType:  "dell_os10_version",
		Timestamp: now.Format(time.RFC3339),
		Date:      now.Format("2006-01-02"),
		Message:   data,
	}}, nil
}
