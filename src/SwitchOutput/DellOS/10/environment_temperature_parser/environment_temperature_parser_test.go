package environment_temperature_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseEnvironmentFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_environment.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&EnvironmentParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg EnvironmentData
	json.Unmarshal(b, &msg)

	if msg.UnitID != 1 {
		t.Errorf("UnitID: got %d", msg.UnitID)
	}
	if msg.UnitState != "up" {
		t.Errorf("UnitState: got '%s'", msg.UnitState)
	}
	if msg.UnitTemperature != 42 {
		t.Errorf("UnitTemperature: got %d", msg.UnitTemperature)
	}
	if len(msg.ThermalSensors) != 11 {
		t.Fatalf("Expected 11 sensors, got %d", len(msg.ThermalSensors))
	}
	if msg.ThermalSensors[0].SensorName != "CPU_temp" {
		t.Errorf("First sensor: got '%s'", msg.ThermalSensors[0].SensorName)
	}
	if msg.ThermalSensors[0].Temperature != 28 {
		t.Errorf("First sensor temp: got %d", msg.ThermalSensors[0].Temperature)
	}
	if msg.ThermalSensors[10].SensorName != "NPU temp sensor" {
		t.Errorf("Last sensor: got '%s'", msg.ThermalSensors[10].SensorName)
	}
	if msg.ThermalSensors[10].Temperature != 41 {
		t.Errorf("Last sensor temp: got %d", msg.ThermalSensors[10].Temperature)
	}
}

func TestParseEnvironmentEmpty(t *testing.T) {
	result, err := (&EnvironmentParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
}
