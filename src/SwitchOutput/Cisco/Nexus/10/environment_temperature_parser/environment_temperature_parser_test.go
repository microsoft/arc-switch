package environment_temperature_parser

import (
	"os"
	"testing"
)

func TestParseTemperature(t *testing.T) {
	// Read the sample file
	content, err := os.ReadFile("../show-environment-temperature.txt")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	entries := parseTemperature(string(content))

	if len(entries) == 0 {
		t.Fatal("Expected at least one entry")
	}

	// We expect 4 temperature entries from the sample file
	expectedCount := 4
	if len(entries) != expectedCount {
		t.Errorf("Expected %d entries, got %d", expectedCount, len(entries))
	}

	// Check the first entry
	entry := entries[0]

	// Check data type
	if entry.DataType != "cisco_nexus_environment_temperature" {
		t.Errorf("Expected data_type 'cisco_nexus_environment_temperature', got '%s'", entry.DataType)
	}

	// Check timestamp and date are not empty
	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	if entry.Date == "" {
		t.Error("Date should not be empty")
	}

	// Check message fields
	if entry.Message.Module == "" {
		t.Error("Module should not be empty")
	}
	if entry.Message.Sensor == "" {
		t.Error("Sensor should not be empty")
	}
	if entry.Message.MajorThreshold == "" {
		t.Error("Major threshold should not be empty")
	}
	if entry.Message.MinorThreshold == "" {
		t.Error("Minor threshold should not be empty")
	}
	if entry.Message.CurrentTemp == "" {
		t.Error("Current temperature should not be empty")
	}
	if entry.Message.Status == "" {
		t.Error("Status should not be empty")
	}

	// Verify first entry specific values
	if entry.Message.Module != "1" {
		t.Errorf("Expected module '1', got '%s'", entry.Message.Module)
	}
	if entry.Message.Sensor != "FRONT" {
		t.Errorf("Expected sensor 'FRONT', got '%s'", entry.Message.Sensor)
	}
	if entry.Message.MajorThreshold != "80" {
		t.Errorf("Expected major threshold '80', got '%s'", entry.Message.MajorThreshold)
	}
	if entry.Message.MinorThreshold != "70" {
		t.Errorf("Expected minor threshold '70', got '%s'", entry.Message.MinorThreshold)
	}
	if entry.Message.CurrentTemp != "28" {
		t.Errorf("Expected current temp '28', got '%s'", entry.Message.CurrentTemp)
	}
	if entry.Message.Status != "Ok" {
		t.Errorf("Expected status 'Ok', got '%s'", entry.Message.Status)
	}

	// Check second entry (BACK sensor)
	if len(entries) > 1 {
		secondEntry := entries[1]
		if secondEntry.Message.Sensor != "BACK" {
			t.Errorf("Expected second entry sensor 'BACK', got '%s'", secondEntry.Message.Sensor)
		}
	}

	// Check third entry (CPU sensor)
	if len(entries) > 2 {
		thirdEntry := entries[2]
		if thirdEntry.Message.Sensor != "CPU" {
			t.Errorf("Expected third entry sensor 'CPU', got '%s'", thirdEntry.Message.Sensor)
		}
	}

	// Check fourth entry (Homewood sensor)
	if len(entries) > 3 {
		fourthEntry := entries[3]
		if fourthEntry.Message.Sensor != "Homewood" {
			t.Errorf("Expected fourth entry sensor 'Homewood', got '%s'", fourthEntry.Message.Sensor)
		}
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	// Create a temporary commands file
	content := `{
		"commands": [
			{
				"name": "environment-temperature",
				"command": "show environment temperature"
			},
			{
				"name": "other-command",
				"command": "show version"
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "commands-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading commands
	config, err := loadCommandsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load commands: %v", err)
	}

	if len(config.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(config.Commands))
	}

	// Test finding command
	cmd, err := findCommand(config, "environment-temperature")
	if err != nil {
		t.Fatalf("Failed to find command: %v", err)
	}

	if cmd != "show environment temperature" {
		t.Errorf("Expected command 'show environment temperature', got '%s'", cmd)
	}

	// Test non-existent command
	_, err = findCommand(config, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent command")
	}
}

func TestUnifiedParser(t *testing.T) {
	parser := &UnifiedParser{}

	// Test GetDescription
	desc := parser.GetDescription()
	if desc == "" {
		t.Error("Description should not be empty")
	}

	// Test Parse
	content, err := os.ReadFile("../show-environment-temperature.txt")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	result, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entries, ok := result.([]StandardizedEntry)
	if !ok {
		t.Fatal("Expected result to be []StandardizedEntry")
	}

	if len(entries) == 0 {
		t.Error("Expected at least one entry")
	}
}
