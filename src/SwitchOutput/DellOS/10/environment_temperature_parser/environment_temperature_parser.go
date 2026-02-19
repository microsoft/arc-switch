package environment_temperature_parser

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

type EnvironmentData struct {
	UnitID          int             `json:"unit_id"`
	UnitState       string          `json:"unit_state"`
	UnitTemperature int             `json:"unit_temperature"`
	ThermalSensors  []ThermalSensor `json:"thermal_sensors"`
}

type ThermalSensor struct {
	UnitID      int    `json:"unit_id"`
	SensorID    int    `json:"sensor_id"`
	SensorName  string `json:"sensor_name"`
	Temperature int    `json:"temperature"`
}

type EnvironmentParser struct{}

func (p *EnvironmentParser) GetDescription() string {
	return "Parses 'show environment' output (thermal sensors)"
}

func (p *EnvironmentParser) Parse(input []byte) (interface{}, error) {
	data := EnvironmentData{}
	lines := strings.Split(string(input), "\n")
	separatorRegex := regexp.MustCompile(`^-{10,}$`)
	unitSummaryRegex := regexp.MustCompile(`^(\d+)\s+(up|down)\s+(\d+)$`)
	sensorRegex := regexp.MustCompile(`^(\d+)\s+(\d+)\s+(.+?)\s{2,}(\d+)$`)
	inThermalSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if separatorRegex.MatchString(trimmed) {
			continue
		}
		if strings.HasPrefix(trimmed, "Thermal sensors") {
			inThermalSection = true
			continue
		}
		if strings.HasPrefix(trimmed, "Unit") && (strings.Contains(trimmed, "State") || strings.Contains(trimmed, "Sensor-Id")) {
			continue
		}
		if !inThermalSection {
			if match := unitSummaryRegex.FindStringSubmatch(trimmed); match != nil {
				data.UnitID, _ = strconv.Atoi(match[1])
				data.UnitState = match[2]
				data.UnitTemperature, _ = strconv.Atoi(match[3])
			}
		}
		if inThermalSection {
			if match := sensorRegex.FindStringSubmatch(trimmed); match != nil {
				sensor := ThermalSensor{SensorName: match[3]}
				sensor.UnitID, _ = strconv.Atoi(match[1])
				sensor.SensorID, _ = strconv.Atoi(match[2])
				sensor.Temperature, _ = strconv.Atoi(match[4])
				data.ThermalSensors = append(data.ThermalSensors, sensor)
			}
		}
	}
	now := time.Now().UTC()
	return []StandardizedEntry{{
		DataType: "dell_os10_environment", Timestamp: now.Format(time.RFC3339),
		Date: now.Format("2006-01-02"), Message: data,
	}}, nil
}
