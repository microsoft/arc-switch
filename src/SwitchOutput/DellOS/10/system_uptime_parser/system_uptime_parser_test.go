package system_uptime_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseSystemUptimeWithDays(t *testing.T) {
	sampleInput := ` 10:23:45 up 45 days, 3:12, 2 users, load average: 0.15, 0.10, 0.05`

	entry, err := parseSystemUptime(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse system uptime: %v", err)
	}

	if entry == nil {
		t.Fatal("Expected non-nil entry")
	}

	if entry.DataType != "dell_os10_system_uptime" {
		t.Errorf("data_type: expected 'dell_os10_system_uptime', got '%s'", entry.DataType)
	}

	msg := entry.Message
	if msg.UptimeDays != 45 {
		t.Errorf("UptimeDays: expected 45, got %d", msg.UptimeDays)
	}
	if msg.UptimeHours != 3 {
		t.Errorf("UptimeHours: expected 3, got %d", msg.UptimeHours)
	}
	if msg.UptimeMinutes != 12 {
		t.Errorf("UptimeMinutes: expected 12, got %d", msg.UptimeMinutes)
	}
	if msg.Users != 2 {
		t.Errorf("Users: expected 2, got %d", msg.Users)
	}
	if msg.LoadAvg1Min != "0.15" {
		t.Errorf("LoadAvg1Min: expected '0.15', got '%s'", msg.LoadAvg1Min)
	}
	if msg.LoadAvg5Min != "0.10" {
		t.Errorf("LoadAvg5Min: expected '0.10', got '%s'", msg.LoadAvg5Min)
	}
	if msg.LoadAvg15Min != "0.05" {
		t.Errorf("LoadAvg15Min: expected '0.05', got '%s'", msg.LoadAvg15Min)
	}
	if msg.UptimeTotalHours != 45*24+3 {
		t.Errorf("UptimeTotalHours: expected %d, got %d", 45*24+3, msg.UptimeTotalHours)
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(entry)
	if err != nil {
		t.Errorf("Failed to marshal entry to JSON: %v", err)
	}

	var unmarshaledEntry StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledEntry)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
}

func TestParseSystemUptimeNoDays(t *testing.T) {
	sampleInput := ` 10:23:45 up 3:12, 2 users, load average: 0.15, 0.10, 0.05`

	entry, err := parseSystemUptime(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse system uptime: %v", err)
	}

	if entry == nil {
		t.Fatal("Expected non-nil entry")
	}

	msg := entry.Message
	if msg.UptimeDays != 0 {
		t.Errorf("UptimeDays: expected 0, got %d", msg.UptimeDays)
	}
	if msg.UptimeHours != 3 {
		t.Errorf("UptimeHours: expected 3, got %d", msg.UptimeHours)
	}
	if msg.UptimeMinutes != 12 {
		t.Errorf("UptimeMinutes: expected 12, got %d", msg.UptimeMinutes)
	}
}

func TestParseSystemUptimeMinutesOnly(t *testing.T) {
	sampleInput := ` 10:23:45 up 45 min, 1 user, load average: 0.15, 0.10, 0.05`

	entry, err := parseSystemUptime(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse system uptime: %v", err)
	}

	if entry == nil {
		t.Fatal("Expected non-nil entry")
	}

	msg := entry.Message
	if msg.UptimeMinutes != 45 {
		t.Errorf("UptimeMinutes: expected 45, got %d", msg.UptimeMinutes)
	}
	if msg.Users != 1 {
		t.Errorf("Users: expected 1, got %d", msg.Users)
	}
}

func TestParseSystemUptimeEmptyInput(t *testing.T) {
	_, err := parseSystemUptime("")
	if err == nil {
		t.Error("Expected error for empty input")
	}
}

func TestParseSystemUptimeInvalidInput(t *testing.T) {
	_, err := parseSystemUptime("some random text")
	if err == nil {
		t.Error("Expected error for invalid input")
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "system-uptime",
				"command": "show uptime"
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

	command, err := findSystemUptimeCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show uptime" {
		t.Errorf("Expected command 'show uptime', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
