package main

import (
	"encoding/json"
	"os"
	"testing"
)

const testVersionInput = `Dell SmartFabric OS10 Enterprise
Copyright (c) 1999-2025 by Dell Inc. All Rights Reserved.
OS Version: 10.6.0.5
Build Version: 10.6.0.5.139
Build Time: 2025-07-02T19:13:52+0000
System Type: S5248F-ON
Architecture: x86_64
Up Time: 9 weeks 2 days 23:10:52`

func TestParseVersion(t *testing.T) {
	entries, err := parseVersion(testVersionInput)
	if err != nil {
		t.Fatalf("parseVersion returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.DataType != "dell_os10_version" {
		t.Errorf("DataType: expected 'dell_os10_version', got '%s'", entry.DataType)
	}
	if entry.Message.OSName != "Dell SmartFabric OS10 Enterprise" {
		t.Errorf("OSName: expected 'Dell SmartFabric OS10 Enterprise', got '%s'", entry.Message.OSName)
	}
	if entry.Message.OSVersion != "10.6.0.5" {
		t.Errorf("OSVersion: expected '10.6.0.5', got '%s'", entry.Message.OSVersion)
	}
	if entry.Message.BuildVersion != "10.6.0.5.139" {
		t.Errorf("BuildVersion: expected '10.6.0.5.139', got '%s'", entry.Message.BuildVersion)
	}
	if entry.Message.BuildTime != "2025-07-02T19:13:52+0000" {
		t.Errorf("BuildTime: expected '2025-07-02T19:13:52+0000', got '%s'", entry.Message.BuildTime)
	}
	if entry.Message.SystemType != "S5248F-ON" {
		t.Errorf("SystemType: expected 'S5248F-ON', got '%s'", entry.Message.SystemType)
	}
	if entry.Message.Architecture != "x86_64" {
		t.Errorf("Architecture: expected 'x86_64', got '%s'", entry.Message.Architecture)
	}
	if entry.Message.UpTime != "9 weeks 2 days 23:10:52" {
		t.Errorf("UpTime: expected '9 weeks 2 days 23:10:52', got '%s'", entry.Message.UpTime)
	}
}

func TestParseVersionEmpty(t *testing.T) {
	entries, err := parseVersion("")
	if err != nil {
		t.Fatalf("parseVersion returned error on empty input: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry even for empty input, got %d", len(entries))
	}
}

func TestParseVersionJSON(t *testing.T) {
	entries, err := parseVersion(testVersionInput)
	if err != nil {
		t.Fatalf("parseVersion returned error: %v", err)
	}
	jsonBytes, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	var unmarshaled []StandardizedEntry
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if len(unmarshaled) != 1 {
		t.Errorf("Expected 1 entry after round-trip, got %d", len(unmarshaled))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "version",
				"command": "show version"
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
	command, err := findCommand(config, "version")
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}
	if command != "show version" {
		t.Errorf("Expected command 'show version', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
