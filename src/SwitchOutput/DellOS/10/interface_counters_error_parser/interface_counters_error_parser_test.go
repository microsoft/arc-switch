package interface_counters_error_parser

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
		{"ethernet1/1/1", "ethernet"},
		{"Ethernet1/1/1", "ethernet"},
		{"port-channel10", "port-channel"},
		{"Port-channel10", "port-channel"},
		{"vlan100", "vlan"},
		{"management1/1/1", "management"},
		{"loopback0", "loopback"},
		{"unknown", "unknown"},
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
		{"123", 123},
		{"0", 0},
		{"--", -1},
		{"N/A", -1},
		{"", -1},
		{"  456  ", 456},
		{"invalid", -1},
	}

	for _, test := range tests {
		result := parseCounterValue(test.input)
		if result != test.expected {
			t.Errorf("parseCounterValue(%s) = %d; expected %d", test.input, result, test.expected)
		}
	}
}

func TestParseInterfaceErrorCounters(t *testing.T) {
	sampleInput := `Interface         RX-Err   RX-Drop  RX-OVR   TX-Err   TX-Drop  TX-OVR
-----------       ------   -------  ------   ------   -------  ------
ethernet1/1/1     0        10       0        0        5        0
ethernet1/1/2     5        0        0        3        0        0
port-channel10    0        0        0        0        0        0`

	interfaces := parseInterfaceErrorCounters(sampleInput)

	if len(interfaces) != 3 {
		t.Errorf("Expected 3 interfaces, got %d", len(interfaces))
	}

	// Verify first interface
	if len(interfaces) > 0 {
		entry := interfaces[0]
		if entry.DataType != "dell_os10_interface_error_counters" {
			t.Errorf("data_type: expected 'dell_os10_interface_error_counters', got '%s'", entry.DataType)
		}
		if entry.Message.InterfaceName != "ethernet1/1/1" {
			t.Errorf("InterfaceName: expected 'ethernet1/1/1', got '%s'", entry.Message.InterfaceName)
		}
		if entry.Message.InterfaceType != "ethernet" {
			t.Errorf("InterfaceType: expected 'ethernet', got '%s'", entry.Message.InterfaceType)
		}
		if entry.Message.RxErr != 0 {
			t.Errorf("RxErr: expected 0, got %d", entry.Message.RxErr)
		}
		if entry.Message.RxDrop != 10 {
			t.Errorf("RxDrop: expected 10, got %d", entry.Message.RxDrop)
		}
		if entry.Message.TxDrop != 5 {
			t.Errorf("TxDrop: expected 5, got %d", entry.Message.TxDrop)
		}
		if !entry.Message.HasErrorData {
			t.Error("HasErrorData should be true")
		}
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(interfaces)
	if err != nil {
		t.Errorf("Failed to marshal interfaces to JSON: %v", err)
	}

	var unmarshaledInterfaces []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledInterfaces)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
}

func TestParseInterfaceErrorCountersEmptyInput(t *testing.T) {
	interfaces := parseInterfaceErrorCounters("")
	if len(interfaces) != 0 {
		t.Errorf("Expected 0 interfaces for empty input, got %d", len(interfaces))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "interface-error-counter",
				"command": "show interface counters errors"
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

	command, err := findInterfaceErrorCountersCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show interface counters errors" {
		t.Errorf("Expected command 'show interface counters errors', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
