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
		{"B, R", []string{"B", "R"}},
		{"not advertised", []string{}},
		{"", []string{}},
		{"B", []string{"B"}},
		{"B, R, T", []string{"B", "R", "T"}},
	}

	for _, test := range tests {
		result := parseCapabilities(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("parseCapabilities(%s) length = %d; expected %d", 
				test.input, len(result), len(test.expected))
			continue
		}
		for i, val := range result {
			if val != test.expected[i] {
				t.Errorf("parseCapabilities(%s)[%d] = %s; expected %s",
					test.input, i, val, test.expected[i])
			}
		}
	}
}

func TestParseTimeRemaining(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"40 seconds", 40},
		{"117 seconds", 117},
		{"0 seconds", 0},
		{"invalid", 0},
		{"", 0},
	}

	for _, test := range tests {
		result := parseTimeRemaining(test.input)
		if result != test.expected {
			t.Errorf("parseTimeRemaining(%s) = %d; expected %d", 
				test.input, result, test.expected)
		}
	}
}

func TestParseMaxFrameSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"9216", 9216},
		{"1500", 1500},
		{"not advertised", 0},
		{"", 0},
		{"invalid", 0},
	}

	for _, test := range tests {
		result := parseMaxFrameSize(test.input)
		if result != test.expected {
			t.Errorf("parseMaxFrameSize(%s) = %d; expected %d", 
				test.input, result, test.expected)
		}
	}
}

func TestCleanValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"null", ""},
		{"not advertised", ""},
		{"valid value", "valid value"},
		{"  trimmed  ", "trimmed"},
		{"", ""},
	}

	for _, test := range tests {
		result := cleanValue(test.input)
		if result != test.expected {
			t.Errorf("cleanValue(%s) = %s; expected %s", 
				test.input, result, test.expected)
		}
	}
}

func TestParseLLDPNeighbors(t *testing.T) {
	// Sample input data
	sampleInput := `CONTOSO-TOR1# show lldp neighbors detail
Capability codes:
  (R) Router, (B) Bridge, (T) Telephone, (C) DOCSIS Cable Device
  (W) WLAN Access Point, (P) Repeater, (S) Station, (O) Other
Device ID            Local Intf      Hold-time  Capability  Port ID  

Chassis id: 1111.2222.3301
Port id: 1111.2222.3301
Local Port id: Eth1/1
Port Description: ConnectX-4 Lx, 25G/10G/1G SFP
System Name: null
System Description: null
Time remaining: 40 seconds
System Capabilities: not advertised
Enabled Capabilities: not advertised
Management Address: not advertised
Management Address IPV6: not advertised
Vlan ID: not advertised
Max Frame Size: not advertised
Vlan Name TLV:
[Vlan ID: Vlan Name]  not advertised
Link Aggregation TLV: 
Capability: not advertised
Status : not advertised
Link agg ID : 0


Chassis id: 2416.9d9f.08b0
Port id: Ethernet1/41
Local Port id: Eth1/41
Port Description: MLAG Heartbeat and iBGP TOR1-TOR2
System Name: CONTOSO-DC1-TOR-01.contoso.local
System Description: Cisco Nexus Operating System (NX-OS) Software 10.3(4a)
TAC support: http://www.cisco.com/tac
Copyright (c) 2002-2023, Cisco Systems, Inc. All rights reserved.
Time remaining: 117 seconds
System Capabilities: B, R
Enabled Capabilities: B, R
Management Address: 1234.5678.90ab
Management Address IPV6: not advertised
Vlan ID: not advertised
Max Frame Size: 9216
Vlan Name TLV:
[Vlan ID: Vlan Name]  not advertised
Link Aggregation TLV: 
Capability: enabled
Status : aggregated
Link agg ID : 50


Chassis id: 689e.0baf.916a
Port id: Ethernet1/12/3
Local Port id: Eth1/47
Port Description: INFRA:PTP:Uplink-to-Spine:Eth1/47:HL
System Name: CONTOSO-DC1-SPINE-01.contoso.com
System Description: Cisco Nexus Operating System (NX-OS) Software 10.3(4a)
TAC support: http://www.cisco.com/tac
Copyright (c) 2002-2023, Cisco Systems, Inc. All rights reserved.
Time remaining: 91 seconds
System Capabilities: B, R
Enabled Capabilities: B, R
Management Address: not advertised
Management Address IPV6: 2001:db8:1234:5678::1001
Vlan ID: not advertised
Max Frame Size: 9100
Vlan Name TLV:
[Vlan ID: Vlan Name]  not advertised
Link Aggregation TLV: 
Capability: enabled
Status : not aggregated
Link agg ID : 0


Chassis id: 2416.9d9f.08b8
Port id: Ethernet1/49
Local Port id: Eth1/49
Port Description: L2 MLAG_PEER
System Name: CONTOSO-DC1-TOR-01.contoso.local
System Description: Cisco Nexus Operating System (NX-OS) Software 10.3(4a)
TAC support: http://www.cisco.com/tac
Copyright (c) 2002-2023, Cisco Systems, Inc. All rights reserved.
Time remaining: 117 seconds
System Capabilities: B, R
Enabled Capabilities: B, R
Management Address: 1234.5678.90ae
Management Address IPV6: not advertised
Vlan ID: 99
Max Frame Size: 1500
Vlan Name TLV:
[Vlan ID: Vlan Name]  1: default, 2: Unused_Ports, 6: HNV_PA, 7: Management
Link Aggregation TLV: 
Capability: enabled
Status : aggregated
Link agg ID : 101

Total entries displayed: 4`

	// Parse the data
	neighbors := parseLLDPNeighbors(sampleInput)

	// Should have 4 neighbors
	expectedCount := 4
	if len(neighbors) != expectedCount {
		t.Errorf("Expected %d neighbors, got %d", expectedCount, len(neighbors))
	}

	// Create map for easier testing
	neighborMap := make(map[string]StandardizedEntry)
	for _, entry := range neighbors {
		neighborMap[entry.Message.ChassisID] = entry
	}

	// Test first neighbor (minimal data)
	if neighbor1Entry, exists := neighborMap["1111.2222.3301"]; exists {
		neighbor1 := neighbor1Entry.Message
		// Check standardized fields
		if neighbor1Entry.DataType != "cisco_nexus_lldp_neighbor" {
			t.Errorf("Neighbor1 data_type: expected 'cisco_nexus_lldp_neighbor', got '%s'", 
				neighbor1Entry.DataType)
		}
		if neighbor1Entry.Timestamp == "" {
			t.Errorf("Neighbor1 timestamp should not be empty")
		}
		if neighbor1Entry.Date == "" {
			t.Errorf("Neighbor1 date should not be empty")
		}
		// Check message fields
		if neighbor1.PortID != "1111.2222.3301" {
			t.Errorf("Neighbor1 port ID: expected '1111.2222.3301', got '%s'", neighbor1.PortID)
		}
		if neighbor1.LocalPortID != "Eth1/1" {
			t.Errorf("Neighbor1 local port ID: expected 'Eth1/1', got '%s'", neighbor1.LocalPortID)
		}
		if neighbor1.TimeRemaining != 40 {
			t.Errorf("Neighbor1 time remaining: expected 40, got %d", neighbor1.TimeRemaining)
		}
		if neighbor1.SystemName != "" {
			t.Errorf("Neighbor1 system name: expected empty (null), got '%s'", neighbor1.SystemName)
		}
		if neighbor1.LinkAggregation.LinkAggID != 0 {
			t.Errorf("Neighbor1 link agg ID: expected 0, got %d", neighbor1.LinkAggregation.LinkAggID)
		}
	} else {
		t.Error("Neighbor 1111.2222.3301 not found in parsed data")
	}

	// Test second neighbor (full data with capabilities)
	if neighbor2Entry, exists := neighborMap["2416.9d9f.08b0"]; exists {
		neighbor2 := neighbor2Entry.Message
		if neighbor2.SystemName != "CONTOSO-DC1-TOR-01.contoso.local" {
			t.Errorf("Neighbor2 system name: expected 'CONTOSO-DC1-TOR-01.contoso.local', got '%s'", 
				neighbor2.SystemName)
		}
		if len(neighbor2.SystemCapabilities) != 2 {
			t.Errorf("Neighbor2: expected 2 system capabilities, got %d", 
				len(neighbor2.SystemCapabilities))
		} else {
			if neighbor2.SystemCapabilities[0] != "B" || neighbor2.SystemCapabilities[1] != "R" {
				t.Errorf("Neighbor2 system capabilities: expected [B, R], got %v", 
					neighbor2.SystemCapabilities)
			}
		}
		if neighbor2.MaxFrameSize != 9216 {
			t.Errorf("Neighbor2 max frame size: expected 9216, got %d", neighbor2.MaxFrameSize)
		}
		if neighbor2.LinkAggregation.Capability != "enabled" {
			t.Errorf("Neighbor2 link agg capability: expected 'enabled', got '%s'", 
				neighbor2.LinkAggregation.Capability)
		}
		if neighbor2.LinkAggregation.Status != "aggregated" {
			t.Errorf("Neighbor2 link agg status: expected 'aggregated', got '%s'", 
				neighbor2.LinkAggregation.Status)
		}
		if neighbor2.LinkAggregation.LinkAggID != 50 {
			t.Errorf("Neighbor2 link agg ID: expected 50, got %d", 
				neighbor2.LinkAggregation.LinkAggID)
		}
	} else {
		t.Error("Neighbor 2416.9d9f.08b0 not found in parsed data")
	}

	// Test third neighbor (IPv6 management address)
	if neighbor3Entry, exists := neighborMap["689e.0baf.916a"]; exists {
		neighbor3 := neighbor3Entry.Message
		if neighbor3.ManagementAddressIPv6 != "2001:db8:1234:5678::1001" {
			t.Errorf("Neighbor3 IPv6 mgmt address: expected '2001:db8:1234:5678::1001', got '%s'", 
				neighbor3.ManagementAddressIPv6)
		}
		if neighbor3.ManagementAddress != "" {
			t.Errorf("Neighbor3 IPv4 mgmt address: expected empty, got '%s'", 
				neighbor3.ManagementAddress)
		}
	} else {
		t.Error("Neighbor 689e.0baf.916a not found in parsed data")
	}

	// Test fourth neighbor (VLAN names)
	if neighbor4Entry, exists := neighborMap["2416.9d9f.08b8"]; exists {
		neighbor4 := neighbor4Entry.Message
		if neighbor4.VlanID != "99" {
			t.Errorf("Neighbor4 VLAN ID: expected '99', got '%s'", neighbor4.VlanID)
		}
		if neighbor4.VlanNames == nil || len(neighbor4.VlanNames) == 0 {
			t.Error("Neighbor4 should have VLAN names")
		} else {
			if vlanName, exists := neighbor4.VlanNames["1"]; !exists || vlanName != "default" {
				t.Errorf("Neighbor4 VLAN 1: expected 'default', got '%s'", vlanName)
			}
			if vlanName, exists := neighbor4.VlanNames["7"]; !exists || vlanName != "Management" {
				t.Errorf("Neighbor4 VLAN 7: expected 'Management', got '%s'", vlanName)
			}
		}
	} else {
		t.Error("Neighbor 2416.9d9f.08b8 not found in parsed data")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(neighbors)
	if err != nil {
		t.Errorf("Failed to marshal neighbors to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledNeighbors []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledNeighbors)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if len(unmarshaledNeighbors) != len(neighbors) {
		t.Errorf("JSON round-trip failed: expected %d neighbors, got %d", 
			len(neighbors), len(unmarshaledNeighbors))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	// Create a temporary commands file
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "lldp-neighbor",
				"command": "show lldp neighbors detail"
			},
			{
				"name": "test-command",
				"command": "show test"
			}
		]
	}`

	err := os.WriteFile(tempFile, []byte(commandsData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test commands file: %v", err)
	}
	defer os.Remove(tempFile)

	// Test loading commands
	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load commands from file: %v", err)
	}

	if len(config.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(config.Commands))
	}

	// Test finding lldp-neighbor command
	command, err := findLLDPCommand(config)
	if err != nil {
		t.Errorf("Failed to find lldp-neighbor command: %v", err)
	}

	expectedCommand := "show lldp neighbors detail"
	if command != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, command)
	}
}