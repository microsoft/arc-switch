package main

import (
	"encoding/json"
	"os"
	"testing"
)

const testLldpInput = `
Remote Chassis ID Subtype: Mac address (4)
Remote Chassis ID: c8:4b:d6:90:c7:27
Remote Port Subtype: Mac address (3)
Remote Port ID: c8:4b:d6:90:c7:27
Remote Port Description: NIC 25Gb QSFP
Local Port ID: ethernet1/1/1:1
Locally assigned remote Neighbor Index: 4064
Remote TTL: 121
Information valid for next 98 seconds
Time since last information change of this neighbor: 3 weeks 4 days 21:30:16
Remote System Name: Not Advertised
Remote System Desc: Not Advertised
Remote Max Frame Size: 0
Remote Aggregation Status: false
MAC PHY Configuration:
    Auto-neg supported: 0
    Auto-neg enabled: 0
---------------------------------------------------------------------------

Remote Chassis ID Subtype: Mac address (4)
Remote Chassis ID: c8:4b:d6:90:c8:0d
Remote Port Subtype: Mac address (3)
Remote Port ID: c8:4b:d6:90:c8:0d
Remote Port Description: NIC 25Gb QSFP
Local Port ID: ethernet1/1/2:1
Locally assigned remote Neighbor Index: 4051
Remote TTL: 121
Information valid for next 119 seconds
Time since last information change of this neighbor: 3 weeks 4 days 21:43:04
Remote System Name: Not Advertised
Remote System Desc: Not Advertised
Remote Max Frame Size: 0
Remote Aggregation Status: false
MAC PHY Configuration:
    Auto-neg supported: 0
    Auto-neg enabled: 0
---------------------------------------------------------------------------`

func TestParseLldpNeighbor(t *testing.T) {
	entries, err := parseLldpNeighbor(testLldpInput)
	if err != nil {
		t.Fatalf("parseLldpNeighbor returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}

	entry := entries[0]
	if entry.DataType != "dell_os10_lldp_neighbor" {
		t.Errorf("DataType: expected 'dell_os10_lldp_neighbor', got '%s'", entry.DataType)
	}
	if entry.Message.RemoteChassisID != "c8:4b:d6:90:c7:27" {
		t.Errorf("RemoteChassisID: expected 'c8:4b:d6:90:c7:27', got '%s'", entry.Message.RemoteChassisID)
	}
	if entry.Message.RemotePortDescription != "NIC 25Gb QSFP" {
		t.Errorf("RemotePortDescription: expected 'NIC 25Gb QSFP', got '%s'", entry.Message.RemotePortDescription)
	}
	if entry.Message.LocalPortID != "ethernet1/1/1:1" {
		t.Errorf("LocalPortID: expected 'ethernet1/1/1:1', got '%s'", entry.Message.LocalPortID)
	}
	if entry.Message.RemoteNeighborIndex != 4064 {
		t.Errorf("RemoteNeighborIndex: expected 4064, got %d", entry.Message.RemoteNeighborIndex)
	}
	if entry.Message.RemoteTTL != 121 {
		t.Errorf("RemoteTTL: expected 121, got %d", entry.Message.RemoteTTL)
	}
	if entry.Message.RemoteSystemName != "Not Advertised" {
		t.Errorf("RemoteSystemName: expected 'Not Advertised', got '%s'", entry.Message.RemoteSystemName)
	}
	if entry.Message.AutoNegSupported != 0 {
		t.Errorf("AutoNegSupported: expected 0, got %d", entry.Message.AutoNegSupported)
	}

	// Verify second entry
	if entries[1].Message.LocalPortID != "ethernet1/1/2:1" {
		t.Errorf("Second entry LocalPortID: expected 'ethernet1/1/2:1', got '%s'", entries[1].Message.LocalPortID)
	}
}

func TestParseLldpNeighborEmpty(t *testing.T) {
	entries, err := parseLldpNeighbor("")
	if err != nil {
		t.Fatalf("parseLldpNeighbor returned error on empty input: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseLldpNeighborNoTrailingSeparator(t *testing.T) {
	input := `Remote Chassis ID Subtype: Mac address (4)
Remote Chassis ID: aa:bb:cc:dd:ee:ff
Local Port ID: ethernet1/1/5:1
Remote TTL: 60`

	entries, err := parseLldpNeighbor(input)
	if err != nil {
		t.Fatalf("parseLldpNeighbor returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry without trailing separator, got %d", len(entries))
	}
}

func TestParseLldpNeighborJSON(t *testing.T) {
	entries, err := parseLldpNeighbor(testLldpInput)
	if err != nil {
		t.Fatalf("parseLldpNeighbor returned error: %v", err)
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
	if len(unmarshaled) != 2 {
		t.Errorf("Expected 2 entries after round-trip, got %d", len(unmarshaled))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "lldp-neighbor",
				"command": "show lldp neighbors detail"
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
	command, err := findCommand(config, "lldp-neighbor")
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}
	if command != "show lldp neighbors detail" {
		t.Errorf("Expected command 'show lldp neighbors detail', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
