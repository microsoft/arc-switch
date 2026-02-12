package mac_address_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseMAC(t *testing.T) {
	// Sample Dell OS10 show mac address-table output
	sampleInput := `VlanId  Mac Address        Type     Interface
------  -----------------  -------  ---------
1       90:b1:1c:f4:a6:8f  dynamic  ethernet1/1/3
1       00:1a:2b:3c:4d:5e  static   ethernet1/1/4
100     aa:bb:cc:dd:ee:ff  dynamic  port-channel10`

	entries, err := parseMAC(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse MAC address table: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// Verify first entry
	if len(entries) > 0 {
		entry := entries[0]
		if entry.DataType != "dell_os10_mac_table" {
			t.Errorf("data_type: expected 'dell_os10_mac_table', got '%s'", entry.DataType)
		}
		if entry.Message.VLAN != "1" {
			t.Errorf("VLAN: expected '1', got '%s'", entry.Message.VLAN)
		}
		if entry.Message.MACAddress != "90:b1:1c:f4:a6:8f" {
			t.Errorf("MACAddress: expected '90:b1:1c:f4:a6:8f', got '%s'", entry.Message.MACAddress)
		}
		if entry.Message.Type != "dynamic" {
			t.Errorf("Type: expected 'dynamic', got '%s'", entry.Message.Type)
		}
		if entry.Message.Interface != "ethernet1/1/3" {
			t.Errorf("Interface: expected 'ethernet1/1/3', got '%s'", entry.Message.Interface)
		}
	}

	// Verify static entry
	if len(entries) > 1 {
		entry := entries[1]
		if entry.Message.Type != "static" {
			t.Errorf("Type: expected 'static', got '%s'", entry.Message.Type)
		}
	}

	// Verify port-channel entry
	if len(entries) > 2 {
		entry := entries[2]
		if entry.Message.VLAN != "100" {
			t.Errorf("VLAN: expected '100', got '%s'", entry.Message.VLAN)
		}
		if entry.Message.Interface != "port-channel10" {
			t.Errorf("Interface: expected 'port-channel10', got '%s'", entry.Message.Interface)
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

func TestParseMACWithPrivateVLAN(t *testing.T) {
	// Sample with private VLAN
	sampleInput := `VlanId  Mac Address        Type     Interface        Private VLAN
------  -----------------  -------  ---------        ------------
1       90:b1:1c:f4:a6:8f  dynamic  ethernet1/1/3    pv 200`

	entries, err := parseMAC(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse MAC address table: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	if len(entries) > 0 && entries[0].Message.PrivateVLAN != "200" {
		t.Errorf("PrivateVLAN: expected '200', got '%s'", entries[0].Message.PrivateVLAN)
	}
}

func TestParseMACNoHeader(t *testing.T) {
	// Input without header
	sampleInput := `Some random text
without a proper header`

	_, err := parseMAC(sampleInput)
	if err == nil {
		t.Error("Expected error for input without header")
	}
}

func TestParseMACEmptyTable(t *testing.T) {
	// Input with header but no entries
	sampleInput := `VlanId  Mac Address        Type     Interface
------  -----------------  -------  ---------`

	entries, err := parseMAC(sampleInput)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty table, got %d", len(entries))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "mac-address-table",
				"command": "show mac address-table"
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

	command, err := findMacAddressCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show mac address-table" {
		t.Errorf("Expected command 'show mac address-table', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
