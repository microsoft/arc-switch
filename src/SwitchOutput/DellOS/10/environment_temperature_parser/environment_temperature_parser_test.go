package environment_temperature_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseTemperature(t *testing.T) {
	sampleInput := `Unit  Sensor        Current   Minor     Major     Status
----  ------        -------   -----     -----     ------
1     CPU           45        70        80        Ok
1     FRONT         28        55        65        Ok
1     BACK          32        60        70        Ok`

	entries := parseTemperature(sampleInput)

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// Verify first entry
	if len(entries) > 0 {
		entry := entries[0]
		if entry.DataType != "dell_os10_environment_temperature" {
			t.Errorf("data_type: expected 'dell_os10_environment_temperature', got '%s'", entry.DataType)
		}
		if entry.Message.Unit != "1" {
			t.Errorf("Unit: expected '1', got '%s'", entry.Message.Unit)
		}
		if entry.Message.Sensor != "CPU" {
			t.Errorf("Sensor: expected 'CPU', got '%s'", entry.Message.Sensor)
		}
		if entry.Message.CurrentTemp != 45 {
			t.Errorf("CurrentTemp: expected 45, got %f", entry.Message.CurrentTemp)
		}
		if entry.Message.MinorThreshold != 70 {
			t.Errorf("MinorThreshold: expected 70, got %f", entry.Message.MinorThreshold)
		}
		if entry.Message.MajorThreshold != 80 {
			t.Errorf("MajorThreshold: expected 80, got %f", entry.Message.MajorThreshold)
		}
		if entry.Message.Status != "Ok" {
			t.Errorf("Status: expected 'Ok', got '%s'", entry.Message.Status)
		}
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(entries)
	if err != nil {
		t.Errorf("Failed to marshal entries to JSON: %v", err)
	}

	var unmarshaledEntries []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledEntries)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
}

func TestParseTemperatureEmptyInput(t *testing.T) {
	entries := parseTemperature("")
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseTemperatureAlertStatus(t *testing.T) {
	sampleInput := `Unit  Sensor        Current   Minor     Major     Status
----  ------        -------   -----     -----     ------
1     CPU           75        70        80        Alert`

	entries := parseTemperature(sampleInput)

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].Message.Status != "Alert" {
		t.Errorf("Status: expected 'Alert', got '%s'", entries[0].Message.Status)
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "environment-temperature",
				"command": "show environment temperature"
			}
		]
	}`

	err := os.WriteFile(tempFile, []byte(commandsData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test commands file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load commands from file: %v", err)
	}

	command, err := findTemperatureCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show environment temperature" {
		t.Errorf("Expected command 'show environment temperature', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
