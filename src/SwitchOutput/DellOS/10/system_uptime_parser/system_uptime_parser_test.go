package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseSystemUptime(t *testing.T) {
	entries, err := parseSystemUptime("9 weeks 3 days 01:24:11")
	if err != nil {
		t.Fatalf("parseSystemUptime returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.DataType != "dell_os10_system_uptime" {
		t.Errorf("DataType: expected 'dell_os10_system_uptime', got '%s'", e.DataType)
	}
	if e.Message.Weeks != 9 {
		t.Errorf("Weeks: expected 9, got %d", e.Message.Weeks)
	}
	if e.Message.Days != 3 {
		t.Errorf("Days: expected 3, got %d", e.Message.Days)
	}
	if e.Message.Hours != 1 {
		t.Errorf("Hours: expected 1, got %d", e.Message.Hours)
	}
	if e.Message.Minutes != 24 {
		t.Errorf("Minutes: expected 24, got %d", e.Message.Minutes)
	}
	if e.Message.Seconds != 11 {
		t.Errorf("Seconds: expected 11, got %d", e.Message.Seconds)
	}

	// 9*7*24*3600 + 3*24*3600 + 1*3600 + 24*60 + 11 = 5707451
	expected := int64(9*7*24*3600 + 3*24*3600 + 1*3600 + 24*60 + 11)
	if e.Message.TotalSeconds != expected {
		t.Errorf("TotalSeconds: expected %d, got %d", expected, e.Message.TotalSeconds)
	}
}

func TestParseSystemUptimeNoDays(t *testing.T) {
	entries, err := parseSystemUptime("2 weeks 05:30:00")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	e := entries[0]
	if e.Message.Weeks != 2 {
		t.Errorf("Weeks: expected 2, got %d", e.Message.Weeks)
	}
	if e.Message.Days != 0 {
		t.Errorf("Days: expected 0, got %d", e.Message.Days)
	}
	if e.Message.Hours != 5 {
		t.Errorf("Hours: expected 5, got %d", e.Message.Hours)
	}
}

func TestParseSystemUptimeOnlyTime(t *testing.T) {
	entries, err := parseSystemUptime("12:34:56")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	e := entries[0]
	if e.Message.Hours != 12 {
		t.Errorf("Hours: expected 12, got %d", e.Message.Hours)
	}
	if e.Message.Minutes != 34 {
		t.Errorf("Minutes: expected 34, got %d", e.Message.Minutes)
	}
	expected := int64(12*3600 + 34*60 + 56)
	if e.Message.TotalSeconds != expected {
		t.Errorf("TotalSeconds: expected %d, got %d", expected, e.Message.TotalSeconds)
	}
}

func TestParseSystemUptimeSingularUnits(t *testing.T) {
	entries, err := parseSystemUptime("1 week 1 day 00:00:01")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	e := entries[0]
	if e.Message.Weeks != 1 {
		t.Errorf("Weeks: expected 1, got %d", e.Message.Weeks)
	}
	if e.Message.Days != 1 {
		t.Errorf("Days: expected 1, got %d", e.Message.Days)
	}
}

func TestParseSystemUptimeEmpty(t *testing.T) {
	entries, err := parseSystemUptime("")
	if err != nil {
		t.Fatalf("Error on empty: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry even for empty, got %d", len(entries))
	}
}

func TestParseSystemUptimeJSON(t *testing.T) {
	entries, err := parseSystemUptime("9 weeks 3 days 01:24:11")
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
	err := os.WriteFile(tempFile, []byte(`{"commands":[{"name":"system-uptime","command":"show uptime"}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load: %v", err)
	}
	command, err := findCommand(config, "system-uptime")
	if err != nil {
		t.Errorf("Failed to find: %v", err)
	}
	if command != "show uptime" {
		t.Errorf("Expected 'show uptime', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
