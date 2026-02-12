package inventory_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDetermineComponentType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Chassis", "chassis"},
		{"Unit 1", "unit"},
		{"Power Supply 1", "power_supply"},
		{"PSU 1", "power_supply"},
		{"Fan Tray 1", "fan"},
		{"Ethernet1/1/1", "transceiver"},
		{"SFP+ 10G", "transceiver"},
		{"Module 1", "module"},
		{"Unknown Component", "unknown"},
	}

	for _, test := range tests {
		result := determineComponentType(test.name)
		if result != test.expected {
			t.Errorf("determineComponentType(%s) = %s; expected %s", test.name, result, test.expected)
		}
	}
}

func TestCleanQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"Chassis"`, "Chassis"},
		{`"Power Supply 1"`, "Power Supply 1"},
		{"NoQuotes", "NoQuotes"},
		{`  "Trimmed"  `, "Trimmed"},
		{`""`, ""},
	}

	for _, test := range tests {
		result := cleanQuotes(test.input)
		if result != test.expected {
			t.Errorf("cleanQuotes(%s) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestParseInventory(t *testing.T) {
	sampleInput := `NAME: "Chassis", DESCR: "Dell EMC Networking S4148-ON"
PID: S4148-ON          , VID: 01 , SN: ABC1234567

NAME: "Unit 1", DESCR: "Dell EMC Networking S4148-ON"
PID: S4148-ON          , VID: 01 , SN: ABC1234568

NAME: "Power Supply 1", DESCR: "Dell EMC AC Power Supply"
PID: DPS-550AB-39 A    , VID: 01 , SN: XYZ9876543`

	entries := parseInventory(sampleInput)

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// Verify first entry (Chassis)
	if len(entries) > 0 {
		entry := entries[0]
		if entry.DataType != "dell_os10_inventory" {
			t.Errorf("data_type: expected 'dell_os10_inventory', got '%s'", entry.DataType)
		}
		if entry.Message.Name != "Chassis" {
			t.Errorf("Name: expected 'Chassis', got '%s'", entry.Message.Name)
		}
		if entry.Message.Description != "Dell EMC Networking S4148-ON" {
			t.Errorf("Description: expected 'Dell EMC Networking S4148-ON', got '%s'", entry.Message.Description)
		}
		if entry.Message.ProductID != "S4148-ON" {
			t.Errorf("ProductID: expected 'S4148-ON', got '%s'", entry.Message.ProductID)
		}
		if entry.Message.VersionID != "01" {
			t.Errorf("VersionID: expected '01', got '%s'", entry.Message.VersionID)
		}
		if entry.Message.SerialNumber != "ABC1234567" {
			t.Errorf("SerialNumber: expected 'ABC1234567', got '%s'", entry.Message.SerialNumber)
		}
		if entry.Message.ComponentType != "chassis" {
			t.Errorf("ComponentType: expected 'chassis', got '%s'", entry.Message.ComponentType)
		}
	}

	// Verify second entry (Unit)
	if len(entries) > 1 {
		entry := entries[1]
		if entry.Message.Name != "Unit 1" {
			t.Errorf("Name: expected 'Unit 1', got '%s'", entry.Message.Name)
		}
		if entry.Message.ComponentType != "unit" {
			t.Errorf("ComponentType: expected 'unit', got '%s'", entry.Message.ComponentType)
		}
	}

	// Verify third entry (Power Supply)
	if len(entries) > 2 {
		entry := entries[2]
		if entry.Message.Name != "Power Supply 1" {
			t.Errorf("Name: expected 'Power Supply 1', got '%s'", entry.Message.Name)
		}
		if entry.Message.ComponentType != "power_supply" {
			t.Errorf("ComponentType: expected 'power_supply', got '%s'", entry.Message.ComponentType)
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

func TestParseInventoryEmptyInput(t *testing.T) {
	entries := parseInventory("")
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseInventoryFanTray(t *testing.T) {
	sampleInput := `NAME: "Fan Tray 1", DESCR: "Dell EMC Fan Tray"
PID: FAN-S4148         , VID: 01 , SN: FAN1234567

NAME: "Fan Tray 2", DESCR: "Dell EMC Fan Tray"
PID: FAN-S4148         , VID: 01 , SN: FAN1234568`

	entries := parseInventory(sampleInput)

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if entry.Message.ComponentType != "fan" {
			t.Errorf("ComponentType: expected 'fan', got '%s'", entry.Message.ComponentType)
		}
	}
}

func TestParseInventoryTransceivers(t *testing.T) {
	sampleInput := `NAME: "Ethernet1/1/1", DESCR: "SFP+ 10GBASE-SR"
PID: SFP-10G-SR        , VID: V02, SN: TRANS123456

NAME: "Ethernet1/1/2", DESCR: "SFP+ 10GBASE-LR"
PID: SFP-10G-LR        , VID: V01, SN: TRANS654321`

	entries := parseInventory(sampleInput)

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if entry.Message.ComponentType != "transceiver" {
			t.Errorf("ComponentType: expected 'transceiver', got '%s'", entry.Message.ComponentType)
		}
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "inventory",
				"command": "show inventory"
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

	command, err := findInventoryCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show inventory" {
		t.Errorf("Expected command 'show inventory', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
