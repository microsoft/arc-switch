package lldp_neighbor_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseCapabilities(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Repeater, Bridge, Router", []string{"Repeater", "Bridge", "Router"}},
		{"Bridge", []string{"Bridge"}},
		{"", []string{}},
		{"Not advertised", []string{}},
		{"not advertised", []string{}},
		{"Router, Station", []string{"Router", "Station"}},
	}

	for _, test := range tests {
		result := parseCapabilities(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("parseCapabilities(%s) returned %d items; expected %d", test.input, len(result), len(test.expected))
			continue
		}
		for i, val := range result {
			if val != test.expected[i] {
				t.Errorf("parseCapabilities(%s)[%d] = %s; expected %s", test.input, i, val, test.expected[i])
			}
		}
	}
}

func TestParseLLDPNeighbors(t *testing.T) {
	// Sample Dell OS10 show lldp neighbors detail output
	sampleInput := `------------------------------------------------------------------------
Remote Chassis ID Subtype: Mac address
Remote Chassis ID: 90:b1:1c:f4:a6:00
Remote Port Subtype: Interface name
Remote Port ID: ethernet1/1/1
Remote Port Description: Uplink to Core
Local Port ID: ethernet1/1/3
Remote System Name: switch01.example.com
Remote System Desc: Dell OS10 Switch
Remote TTL: 120
Remote Max Frame Size: 9216
Remote Aggregation Status: Capable
Remote Management Address (IPv4): 10.1.1.1
Existing System Capabilities: Bridge, Router
Enabled System Capabilities: Bridge, Router
Time since last information change of this neighbor: 1d2h
  Auto-neg supported: 1
  Auto-neg enabled: 1
------------------------------------------------------------------------
Remote Chassis ID Subtype: Mac address
Remote Chassis ID: aa:bb:cc:dd:ee:ff
Remote Port ID: Ethernet0
Local Port ID: ethernet1/1/4
Remote System Name: server01
Remote TTL: 120
Remote Max Frame Size: 1500
------------------------------------------------------------------------`

	neighbors := parseLLDPNeighbors(sampleInput)

	if len(neighbors) != 2 {
		t.Errorf("Expected 2 neighbors, got %d", len(neighbors))
	}

	// Find the first neighbor and verify
	for _, entry := range neighbors {
		if entry.Message.LocalPortID == "ethernet1/1/3" {
			if entry.DataType != "dell_os10_lldp_neighbor" {
				t.Errorf("data_type: expected 'dell_os10_lldp_neighbor', got '%s'", entry.DataType)
			}
			if entry.Message.RemoteChassisID != "90:b1:1c:f4:a6:00" {
				t.Errorf("RemoteChassisID: expected '90:b1:1c:f4:a6:00', got '%s'", entry.Message.RemoteChassisID)
			}
			if entry.Message.RemoteChassisIDSubtype != "Mac address" {
				t.Errorf("RemoteChassisIDSubtype: expected 'Mac address', got '%s'", entry.Message.RemoteChassisIDSubtype)
			}
			if entry.Message.RemotePortID != "ethernet1/1/1" {
				t.Errorf("RemotePortID: expected 'ethernet1/1/1', got '%s'", entry.Message.RemotePortID)
			}
			if entry.Message.RemotePortDescription != "Uplink to Core" {
				t.Errorf("RemotePortDescription: expected 'Uplink to Core', got '%s'", entry.Message.RemotePortDescription)
			}
			if entry.Message.RemoteSystemName != "switch01.example.com" {
				t.Errorf("RemoteSystemName: expected 'switch01.example.com', got '%s'", entry.Message.RemoteSystemName)
			}
			if entry.Message.RemoteTTL != 120 {
				t.Errorf("RemoteTTL: expected 120, got %d", entry.Message.RemoteTTL)
			}
			if entry.Message.RemoteMaxFrameSize != 9216 {
				t.Errorf("RemoteMaxFrameSize: expected 9216, got %d", entry.Message.RemoteMaxFrameSize)
			}
			if entry.Message.ManagementAddressIPv4 != "10.1.1.1" {
				t.Errorf("ManagementAddressIPv4: expected '10.1.1.1', got '%s'", entry.Message.ManagementAddressIPv4)
			}
			if len(entry.Message.SystemCapabilities) != 2 {
				t.Errorf("SystemCapabilities: expected 2 capabilities, got %d", len(entry.Message.SystemCapabilities))
			}
			if !entry.Message.AutoNegSupported {
				t.Error("AutoNegSupported should be true")
			}
			if !entry.Message.AutoNegEnabled {
				t.Error("AutoNegEnabled should be true")
			}
			break
		}
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(neighbors)
	if err != nil {
		t.Errorf("Failed to marshal neighbors to JSON: %v", err)
	}

	var unmarshaledNeighbors []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledNeighbors)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
}

func TestParseLLDPNeighborsEmptyInput(t *testing.T) {
	neighbors := parseLLDPNeighbors("")
	if len(neighbors) != 0 {
		t.Errorf("Expected 0 neighbors for empty input, got %d", len(neighbors))
	}
}

func TestParseLLDPNeighborsWithIPv6(t *testing.T) {
	sampleInput := `------------------------------------------------------------------------
Remote Chassis ID: 90:b1:1c:f4:a6:00
Remote Port ID: ethernet1/1/1
Local Port ID: ethernet1/1/3
Remote TTL: 120
Remote Management Address (IPv4): 10.1.1.1
Remote Management Address (IPv6): 2001:db8::1
------------------------------------------------------------------------`

	neighbors := parseLLDPNeighbors(sampleInput)

	if len(neighbors) != 1 {
		t.Errorf("Expected 1 neighbor, got %d", len(neighbors))
	}

	if len(neighbors) > 0 {
		if neighbors[0].Message.ManagementAddressIPv4 != "10.1.1.1" {
			t.Errorf("ManagementAddressIPv4: expected '10.1.1.1', got '%s'", neighbors[0].Message.ManagementAddressIPv4)
		}
		if neighbors[0].Message.ManagementAddressIPv6 != "2001:db8::1" {
			t.Errorf("ManagementAddressIPv6: expected '2001:db8::1', got '%s'", neighbors[0].Message.ManagementAddressIPv6)
		}
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "lldp-neighbors",
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

	command, err := findLLDPCommand(config)
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
