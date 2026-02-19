package main

import (
	"encoding/json"
	"os"
	"testing"
)

const testEnvironmentInput = `
Unit    State             Temperature
-------------------------------------
1       up                42

Thermal sensors
Unit   Sensor-Id        Sensor-name                               Temperature
------------------------------------------------------------------------------
1       1           CPU_temp                                          28
1       2           NPU_Near_temp                                     30
1       3           PT_Left_temp                                      30
1       4           PT_Right_temp                                     31
1       5           PT_Mid_temp                                       33
1       6           ILET_AF_temp                                      25
1       7           PSU1_temp                                         39
1       8           PSU1_AF_temp                                      25
1       9           PSU2_temp                                         42
1       10          PSU2_AF_temp                                      24
1       11          NPU temp sensor                                   41`

func TestParseEnvironmentTemperature(t *testing.T) {
	entries, err := parseEnvironmentTemperature(testEnvironmentInput)
	if err != nil {
		t.Fatalf("parseEnvironmentTemperature returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.DataType != "dell_os10_environment_temperature" {
		t.Errorf("DataType: expected 'dell_os10_environment_temperature', got '%s'", e.DataType)
	}
	if e.Message.UnitID != 1 {
		t.Errorf("UnitID: expected 1, got %d", e.Message.UnitID)
	}
	if e.Message.UnitState != "up" {
		t.Errorf("UnitState: expected 'up', got '%s'", e.Message.UnitState)
	}
	if e.Message.UnitTemperature != 42 {
		t.Errorf("UnitTemperature: expected 42, got %d", e.Message.UnitTemperature)
	}
	if len(e.Message.ThermalSensors) != 11 {
		t.Fatalf("Expected 11 thermal sensors, got %d", len(e.Message.ThermalSensors))
	}

	// Verify first sensor
	s0 := e.Message.ThermalSensors[0]
	if s0.SensorName != "CPU_temp" {
		t.Errorf("First sensor name: expected 'CPU_temp', got '%s'", s0.SensorName)
	}
	if s0.Temperature != 28 {
		t.Errorf("First sensor temp: expected 28, got %d", s0.Temperature)
	}
	if s0.SensorID != 1 {
		t.Errorf("First sensor ID: expected 1, got %d", s0.SensorID)
	}

	// Verify sensor with spaces in name (NPU temp sensor)
	s10 := e.Message.ThermalSensors[10]
	if s10.SensorName != "NPU temp sensor" {
		t.Errorf("Sensor 11 name: expected 'NPU temp sensor', got '%s'", s10.SensorName)
	}
	if s10.Temperature != 41 {
		t.Errorf("Sensor 11 temp: expected 41, got %d", s10.Temperature)
	}

	// Verify last underscore sensor
	s9 := e.Message.ThermalSensors[9]
	if s9.SensorName != "PSU2_AF_temp" {
		t.Errorf("Sensor 10 name: expected 'PSU2_AF_temp', got '%s'", s9.SensorName)
	}
	if s9.Temperature != 24 {
		t.Errorf("Sensor 10 temp: expected 24, got %d", s9.Temperature)
	}
}

func TestParseEnvironmentTemperatureEmpty(t *testing.T) {
	entries, err := parseEnvironmentTemperature("")
	if err != nil {
		t.Fatalf("Error on empty: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry even for empty, got %d", len(entries))
	}
}

func TestParseEnvironmentTemperatureJSON(t *testing.T) {
	entries, err := parseEnvironmentTemperature(testEnvironmentInput)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	jsonBytes, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	var unmarshaled []StandardizedEntry
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	err := os.WriteFile(tempFile, []byte(`{"commands":[{"name":"environment-temperature","command":"show environment"}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load: %v", err)
	}
	command, err := findCommand(config, "environment-temperature")
	if err != nil {
		t.Errorf("Failed to find: %v", err)
	}
	if command != "show environment" {
		t.Errorf("Expected 'show environment', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
