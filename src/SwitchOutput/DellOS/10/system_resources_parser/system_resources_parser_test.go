package system_resources_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseSystemResourcesFreeStyle(t *testing.T) {
	sampleInput := `              total        used        free      shared  buff/cache   available
Mem:          16000        8000        2000         500        6000       10000
Swap:          4000        1000        3000`

	entries, err := parseSystemResources(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse system resources: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	if len(entries) > 0 {
		entry := entries[0]
		if entry.DataType != "dell_os10_system_resources" {
			t.Errorf("data_type: expected 'dell_os10_system_resources', got '%s'", entry.DataType)
		}

		msg := entry.Message
		if msg.MemoryTotal != 16000 {
			t.Errorf("MemoryTotal: expected 16000, got %d", msg.MemoryTotal)
		}
		if msg.MemoryUsed != 8000 {
			t.Errorf("MemoryUsed: expected 8000, got %d", msg.MemoryUsed)
		}
		if msg.MemoryFree != 2000 {
			t.Errorf("MemoryFree: expected 2000, got %d", msg.MemoryFree)
		}
		if msg.SwapTotal != 4000 {
			t.Errorf("SwapTotal: expected 4000, got %d", msg.SwapTotal)
		}
		if msg.SwapUsed != 1000 {
			t.Errorf("SwapUsed: expected 1000, got %d", msg.SwapUsed)
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

func TestParseSystemResourcesWithCPU(t *testing.T) {
	sampleInput := `CPU usage: 5.5% user, 3.2% system, 91.3% idle
load average: 0.15, 0.10, 0.05
Mem:          16000        8000        2000         500        6000       10000
Swap:          4000        1000        3000`

	entries, err := parseSystemResources(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse system resources: %v", err)
	}

	if len(entries) > 0 {
		msg := entries[0].Message
		if msg.CPUUsageUser != 5.5 {
			t.Errorf("CPUUsageUser: expected 5.5, got %f", msg.CPUUsageUser)
		}
		if msg.CPUUsageSystem != 3.2 {
			t.Errorf("CPUUsageSystem: expected 3.2, got %f", msg.CPUUsageSystem)
		}
		if msg.CPUUsageIdle != 91.3 {
			t.Errorf("CPUUsageIdle: expected 91.3, got %f", msg.CPUUsageIdle)
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
	}
}

func TestParseSystemResourcesEmptyInput(t *testing.T) {
	entries, err := parseSystemResources("")
	if err != nil {
		t.Errorf("Unexpected error for empty input: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (with zero values), got %d", len(entries))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "system-resources",
				"command": "show system-resources"
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

	command, err := findSystemResourcesCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show system-resources" {
		t.Errorf("Expected command 'show system-resources', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
