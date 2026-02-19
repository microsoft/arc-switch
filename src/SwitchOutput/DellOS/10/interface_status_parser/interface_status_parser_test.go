package main

import (
	"encoding/json"
	"os"
	"testing"
)

const testInterfaceStatusInput = `--------------------------------------------------------------------------------------------------
Port            Description     Status   Speed    Duplex   Mode Vlan Tagged-Vlans
--------------------------------------------------------------------------------------------------
Eth 1/1/1:1     Switched-Comp.. up       10G      full     T    7    6,201,301,401,501-516,3939
Eth 1/1/2:1     Switched-Comp.. up       10G      full     T    7    6,201,301,401,501-516,3939
Eth 1/1/3:1     Switched-Comp.. down     --       --       T    7    6,201,301,401,501-516,3939
Po 128          Uplink-Po..     up       200G     full     T    1    6,7,201,301,401,501-516,3939
Vl 7                            up       --       --       --   --`

func TestParseInterfaceStatus(t *testing.T) {
	entries, err := parseInterfaceStatus(testInterfaceStatusInput)
	if err != nil {
		t.Fatalf("parseInterfaceStatus returned error: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("Expected 5 entries, got %d", len(entries))
	}

	e0 := entries[0]
	if e0.DataType != "dell_os10_interface_status" {
		t.Errorf("DataType: expected 'dell_os10_interface_status', got '%s'", e0.DataType)
	}
	if e0.Message.Port != "Eth 1/1/1:1" {
		t.Errorf("Port: expected 'Eth 1/1/1:1', got '%s'", e0.Message.Port)
	}
	if e0.Message.Status != "up" {
		t.Errorf("Status: expected 'up', got '%s'", e0.Message.Status)
	}
	if !e0.Message.IsUp {
		t.Error("IsUp: expected true for 'up' status")
	}
	if e0.Message.Speed != "10G" {
		t.Errorf("Speed: expected '10G', got '%s'", e0.Message.Speed)
	}

	// Verify down interface
	if entries[2].Message.Status != "down" {
		t.Errorf("Third entry Status: expected 'down', got '%s'", entries[2].Message.Status)
	}
	if entries[2].Message.IsUp {
		t.Error("Third entry IsUp: expected false for 'down' status")
	}

	// Verify port-channel
	if entries[3].Message.Port != "Po 128" {
		t.Errorf("Fourth entry Port: expected 'Po 128', got '%s'", entries[3].Message.Port)
	}
	if entries[3].Message.Speed != "200G" {
		t.Errorf("Fourth entry Speed: expected '200G', got '%s'", entries[3].Message.Speed)
	}
}

func TestParseInterfaceStatusEmpty(t *testing.T) {
	entries, err := parseInterfaceStatus("")
	if err != nil {
		t.Fatalf("Error on empty: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseInterfaceStatusJSON(t *testing.T) {
	entries, err := parseInterfaceStatus(testInterfaceStatusInput)
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
	err := os.WriteFile(tempFile, []byte(`{"commands":[{"name":"interface-status","command":"show interface status"}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load: %v", err)
	}
	command, err := findCommand(config, "interface-status")
	if err != nil {
		t.Errorf("Failed to find: %v", err)
	}
	if command != "show interface status" {
		t.Errorf("Expected 'show interface status', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
