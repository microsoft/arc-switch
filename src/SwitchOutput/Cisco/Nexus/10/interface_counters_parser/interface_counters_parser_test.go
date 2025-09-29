package interface_counters_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseInterfaceType(t *testing.T) {
	tests := []struct {
		interfaceName string
		expected      string
	}{
		{"Eth1/1", "ethernet"},
		{"Eth1/48", "ethernet"},
		{"Po50", "port-channel"},
		{"Po101", "port-channel"},
		{"Vlan1", "vlan"},
		{"Vlan125", "vlan"},
		{"mgmt0", "management"},
		{"Tunnel1", "tunnel"},
		{"Unknown123", "unknown"},
	}

	for _, test := range tests {
		result := parseInterfaceType(test.interfaceName)
		if result != test.expected {
			t.Errorf("parseInterfaceType(%s) = %s; expected %s", test.interfaceName, result, test.expected)
		}
	}
}

func TestParseCounterValue(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"123456", 123456},
		{"0", 0},
		{"--", -1},
		{"", -1},
		{"  123456  ", 123456},
		{"invalid", -1},
	}

	for _, test := range tests {
		result := parseCounterValue(test.input)
		if result != test.expected {
			t.Errorf("parseCounterValue(%s) = %d; expected %d", test.input, result, test.expected)
		}
	}
}

func TestParseInterfaceCounters(t *testing.T) {
	// Sample input data
	sampleInput := `CONTOSO-TOR1# show interface counters 

----------------------------------------------------------------------------------
Port                                     InOctets                      InUcastPkts
----------------------------------------------------------------------------------
mgmt0                                           0                                0

----------------------------------------------------------------------------------
Port                                     InOctets                      InUcastPkts
----------------------------------------------------------------------------------
Eth1/1                               205027653248                        650373664
Eth1/2                               144387970112                        277741204

----------------------------------------------------------------------------------
Port                                     InOctets                      InUcastPkts
----------------------------------------------------------------------------------
Po50                                 125621796156                       1648254276

----------------------------------------------------------------------------------
Port                                     InOctets                      InUcastPkts
----------------------------------------------------------------------------------
Vlan1                                           0                                0

----------------------------------------------------------------------------------
Port                                  InMcastPkts                      InBcastPkts
----------------------------------------------------------------------------------
mgmt0                                           0                                0

----------------------------------------------------------------------------------
Port                                  InMcastPkts                      InBcastPkts
----------------------------------------------------------------------------------
Eth1/1                                    2262324                            68097
Eth1/2                                    1760550                           278890

----------------------------------------------------------------------------------
Port                                  InMcastPkts                      InBcastPkts
----------------------------------------------------------------------------------
Po50                                      1525201                               10

----------------------------------------------------------------------------------
Port                                  InMcastPkts                      InBcastPkts
----------------------------------------------------------------------------------
Vlan1                                     --                                --

----------------------------------------------------------------------------------
Port                                    OutOctets                     OutUcastPkts
----------------------------------------------------------------------------------
mgmt0                                           0                                0

----------------------------------------------------------------------------------
Port                                    OutOctets                     OutUcastPkts
----------------------------------------------------------------------------------
Eth1/1                              3195383643785                       2314463086
Eth1/2                               565759281659                        488751519

----------------------------------------------------------------------------------
Port                                    OutOctets                     OutUcastPkts
----------------------------------------------------------------------------------
Po50                                   2804643442                         23654519

----------------------------------------------------------------------------------
Port                                    OutOctets                     OutUcastPkts
----------------------------------------------------------------------------------
Vlan1                                           0                                0

----------------------------------------------------------------------------------
Port                                 OutMcastPkts                     OutBcastPkts
----------------------------------------------------------------------------------
mgmt0                                           0                                0

----------------------------------------------------------------------------------
Port                                 OutMcastPkts                     OutBcastPkts
----------------------------------------------------------------------------------
Eth1/1                                  365931965                         53571839
Eth1/2                                  366238357                         53360933

----------------------------------------------------------------------------------
Port                                 OutMcastPkts                     OutBcastPkts
----------------------------------------------------------------------------------
Po50                                      1525193                               14

----------------------------------------------------------------------------------
Port                                 OutMcastPkts                     OutBcastPkts
----------------------------------------------------------------------------------
Vlan1                                     --                                --`

	// Parse the data
	interfaces := parseInterfaceCounters(sampleInput)

	// Should have 4 interfaces: mgmt0, Eth1/1, Eth1/2, Po50, Vlan1
	expectedCount := 5
	if len(interfaces) != expectedCount {
		t.Errorf("Expected %d interfaces, got %d", expectedCount, len(interfaces))
	}

	// Find specific interfaces and test their values
	interfaceMap := make(map[string]StandardizedEntry)
	for _, entry := range interfaces {
		interfaceMap[entry.Message.InterfaceName] = entry
	}

	// Test Eth1/1
	if eth1Entry, exists := interfaceMap["Eth1/1"]; exists {
		eth1 := eth1Entry.Message
		// Check standardized fields
		if eth1Entry.DataType != "cisco_nexus_interface_counters" {
			t.Errorf("Eth1/1 data_type: expected 'cisco_nexus_interface_counters', got '%s'", eth1Entry.DataType)
		}
		if eth1Entry.Timestamp == "" {
			t.Errorf("Eth1/1 timestamp should not be empty")
		}
		if eth1Entry.Date == "" {
			t.Errorf("Eth1/1 date should not be empty")
		}
		// Check message fields
		if eth1.InterfaceType != "ethernet" {
			t.Errorf("Eth1/1 interface type: expected 'ethernet', got '%s'", eth1.InterfaceType)
		}
		if eth1.InOctets != 205027653248 {
			t.Errorf("Eth1/1 InOctets: expected 205027653248, got %d", eth1.InOctets)
		}
		if eth1.InUcastPkts != 650373664 {
			t.Errorf("Eth1/1 InUcastPkts: expected 650373664, got %d", eth1.InUcastPkts)
		}
		if eth1.InMcastPkts != 2262324 {
			t.Errorf("Eth1/1 InMcastPkts: expected 2262324, got %d", eth1.InMcastPkts)
		}
		if !eth1.HasIngressData || !eth1.HasEgressData {
			t.Errorf("Eth1/1 should have both ingress and egress data")
		}
	} else {
		t.Error("Eth1/1 interface not found in parsed data")
	}

	// Test Po50 (port-channel)
	if po50Entry, exists := interfaceMap["Po50"]; exists {
		po50 := po50Entry.Message
		if po50.InterfaceType != "port-channel" {
			t.Errorf("Po50 interface type: expected 'port-channel', got '%s'", po50.InterfaceType)
		}
		if po50.InOctets != 125621796156 {
			t.Errorf("Po50 InOctets: expected 125621796156, got %d", po50.InOctets)
		}
	} else {
		t.Error("Po50 interface not found in parsed data")
	}

	// Test Vlan1 (should have -- values converted to -1)
	if vlan1Entry, exists := interfaceMap["Vlan1"]; exists {
		vlan1 := vlan1Entry.Message
		if vlan1.InterfaceType != "vlan" {
			t.Errorf("Vlan1 interface type: expected 'vlan', got '%s'", vlan1.InterfaceType)
		}
		if vlan1.InMcastPkts != -1 {
			t.Errorf("Vlan1 InMcastPkts: expected -1 (unavailable), got %d", vlan1.InMcastPkts)
		}
		if vlan1.InBcastPkts != -1 {
			t.Errorf("Vlan1 InBcastPkts: expected -1 (unavailable), got %d", vlan1.InBcastPkts)
		}
	} else {
		t.Error("Vlan1 interface not found in parsed data")
	}

	// Test mgmt0 (management interface)
	if mgmt0Entry, exists := interfaceMap["mgmt0"]; exists {
		mgmt0 := mgmt0Entry.Message
		if mgmt0.InterfaceType != "management" {
			t.Errorf("mgmt0 interface type: expected 'management', got '%s'", mgmt0.InterfaceType)
		}
	} else {
		t.Error("mgmt0 interface not found in parsed data")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(interfaces)
	if err != nil {
		t.Errorf("Failed to marshal interfaces to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledInterfaces []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledInterfaces)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if len(unmarshaledInterfaces) != len(interfaces) {
		t.Errorf("JSON round-trip failed: expected %d interfaces, got %d", len(interfaces), len(unmarshaledInterfaces))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	// Create a temporary commands file
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "interface-counter",
				"command": "show interface counters"
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

	// Test finding interface-counter command
	command, err := findInterfaceCountersCommand(config)
	if err != nil {
		t.Errorf("Failed to find interface-counter command: %v", err)
	}

	expectedCommand := "show interface counters"
	if command != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, command)
	}

	// Test with missing interface-counter command
	configMissing := &CommandConfig{
		Commands: []Command{
			{Name: "other-command", Command: "show other"},
		},
	}

	_, err = findInterfaceCountersCommand(configMissing)
	if err == nil {
		t.Error("Expected error when interface-counter command is missing")
	}
}
