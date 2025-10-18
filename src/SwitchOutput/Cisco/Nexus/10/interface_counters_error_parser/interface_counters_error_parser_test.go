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
		{"4303", 4303},
	}

	for _, test := range tests {
		result := parseCounterValue(test.input)
		if result != test.expected {
			t.Errorf("parseCounterValue(%s) = %d; expected %d", test.input, result, test.expected)
		}
	}
}

func TestParseInterfaceErrorCounters(t *testing.T) {
	// Sample input data
	sampleInput := `show interface counters errors

--------------------------------------------------------------------------------
Port          Align-Err    FCS-Err   Xmit-Err    Rcv-Err  UnderSize OutDiscards
--------------------------------------------------------------------------------
mgmt0                   0          0         --         --         --          --
Eth1/1                  0          0          0          0          0           0
Eth1/2                  0          0          0          0          0           0
Po700                   0          0          0          0          0        4303

----------------------------------------------------------------------------------
Port           Single-Col  Multi-Col   Late-Col  Exces-Col  Carri-Sen       Runts
----------------------------------------------------------------------------------
mgmt0                   0          0          0          0         --          --
Eth1/1                  0          0          0          0          0           0
Eth1/2                  0          0          0          0          0           0
Po700                   0          0          0          0          0           0

----------------------------------------------------------------------------------
Port            Giants SQETest-Err Deferred-Tx IntMacTx-Er IntMacRx-Er Symbol-Err
----------------------------------------------------------------------------------
mgmt0                0           0           0           0           0          0
Eth1/1               0          --           0           0           0          0
Eth1/2               0          --           0           0           0          0
Po700                0          --           0           0           0          0

----------------------------------------------------------------------------------
Port           InDiscards
----------------------------------------------------------------------------------
mgmt0                  --
Eth1/1                  0
Eth1/2                  0
Po700                   0

--------------------------------------------------------------------------------
Port         Stomped-CRC
--------------------------------------------------------------------------------
Eth1/1                0
Eth1/2                0`

	// Parse the data
	interfaces := parseInterfaceErrorCounters(sampleInput)

	// Should have 4 interfaces: mgmt0, Eth1/1, Eth1/2, Po700
	expectedCount := 4
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
		if eth1Entry.DataType != "cisco_nexus_interface_error_counters" {
			t.Errorf("Eth1/1 data_type: expected 'cisco_nexus_interface_error_counters', got '%s'", eth1Entry.DataType)
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
		if eth1.AlignErr != 0 {
			t.Errorf("Eth1/1 AlignErr: expected 0, got %d", eth1.AlignErr)
		}
		if eth1.FCSErr != 0 {
			t.Errorf("Eth1/1 FCSErr: expected 0, got %d", eth1.FCSErr)
		}
		if eth1.OutDiscards != 0 {
			t.Errorf("Eth1/1 OutDiscards: expected 0, got %d", eth1.OutDiscards)
		}
		if eth1.SQETestErr != -1 {
			t.Errorf("Eth1/1 SQETestErr: expected -1 (unavailable), got %d", eth1.SQETestErr)
		}
		if eth1.StompedCRC != 0 {
			t.Errorf("Eth1/1 StompedCRC: expected 0, got %d", eth1.StompedCRC)
		}
		if !eth1.HasErrorData {
			t.Errorf("Eth1/1 should have error data")
		}
	} else {
		t.Error("Eth1/1 interface not found in parsed data")
	}

	// Test Po700 (port-channel with OutDiscards=4303)
	if po700Entry, exists := interfaceMap["Po700"]; exists {
		po700 := po700Entry.Message
		if po700.InterfaceType != "port-channel" {
			t.Errorf("Po700 interface type: expected 'port-channel', got '%s'", po700.InterfaceType)
		}
		if po700.OutDiscards != 4303 {
			t.Errorf("Po700 OutDiscards: expected 4303, got %d", po700.OutDiscards)
		}
		if po700.AlignErr != 0 {
			t.Errorf("Po700 AlignErr: expected 0, got %d", po700.AlignErr)
		}
		if po700.SQETestErr != -1 {
			t.Errorf("Po700 SQETestErr: expected -1 (unavailable), got %d", po700.SQETestErr)
		}
	} else {
		t.Error("Po700 interface not found in parsed data")
	}

	// Test mgmt0 (management interface with -- values)
	if mgmt0Entry, exists := interfaceMap["mgmt0"]; exists {
		mgmt0 := mgmt0Entry.Message
		if mgmt0.InterfaceType != "management" {
			t.Errorf("mgmt0 interface type: expected 'management', got '%s'", mgmt0.InterfaceType)
		}
		if mgmt0.RcvErr != -1 {
			t.Errorf("mgmt0 RcvErr: expected -1 (unavailable), got %d", mgmt0.RcvErr)
		}
		if mgmt0.UnderSize != -1 {
			t.Errorf("mgmt0 UnderSize: expected -1 (unavailable), got %d", mgmt0.UnderSize)
		}
		if mgmt0.OutDiscards != -1 {
			t.Errorf("mgmt0 OutDiscards: expected -1 (unavailable), got %d", mgmt0.OutDiscards)
		}
		if mgmt0.InDiscards != -1 {
			t.Errorf("mgmt0 InDiscards: expected -1 (unavailable), got %d", mgmt0.InDiscards)
		}
		if mgmt0.CarriSen != -1 {
			t.Errorf("mgmt0 CarriSen: expected -1 (unavailable), got %d", mgmt0.CarriSen)
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
				"name": "interface-error-counter",
				"command": "show interface counters errors"
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

	// Test finding interface-error-counter command
	command, err := findInterfaceErrorCountersCommand(config)
	if err != nil {
		t.Errorf("Failed to find interface-error-counter command: %v", err)
	}

	expectedCommand := "show interface counters errors"
	if command != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, command)
	}

	// Test with missing interface-error-counter command
	configMissing := &CommandConfig{
		Commands: []Command{
			{Name: "other-command", Command: "show other"},
		},
	}

	_, err = findInterfaceErrorCountersCommand(configMissing)
	if err == nil {
		t.Error("Expected error when interface-error-counter command is missing")
	}
}

func TestParseInterfaceErrorCountersComprehensive(t *testing.T) {
	// More comprehensive test with various counter values
	sampleInput := `show interface counters errors

--------------------------------------------------------------------------------
Port          Align-Err    FCS-Err   Xmit-Err    Rcv-Err  UnderSize OutDiscards
--------------------------------------------------------------------------------
Eth1/10                 5         10         15         20         25          30
Eth1/11                 0          0          0          0          0           0

----------------------------------------------------------------------------------
Port           Single-Col  Multi-Col   Late-Col  Exces-Col  Carri-Sen       Runts
----------------------------------------------------------------------------------
Eth1/10                 1          2          3          4          5           6
Eth1/11                 0          0          0          0          0           0

----------------------------------------------------------------------------------
Port            Giants SQETest-Err Deferred-Tx IntMacTx-Er IntMacRx-Er Symbol-Err
----------------------------------------------------------------------------------
Eth1/10              7          8           9         10          11          12
Eth1/11              0         --           0          0           0           0

----------------------------------------------------------------------------------
Port           InDiscards
----------------------------------------------------------------------------------
Eth1/10                13
Eth1/11                 0

--------------------------------------------------------------------------------
Port         Stomped-CRC
--------------------------------------------------------------------------------
Eth1/10              14
Eth1/11               0`

	interfaces := parseInterfaceErrorCounters(sampleInput)

	// Should have 2 interfaces
	expectedCount := 2
	if len(interfaces) != expectedCount {
		t.Errorf("Expected %d interfaces, got %d", expectedCount, len(interfaces))
	}

	// Find Eth1/10 and verify all counters
	interfaceMap := make(map[string]StandardizedEntry)
	for _, entry := range interfaces {
		interfaceMap[entry.Message.InterfaceName] = entry
	}

	if eth10Entry, exists := interfaceMap["Eth1/10"]; exists {
		eth10 := eth10Entry.Message
		
		// Verify all counter values
		expectedValues := map[string]int64{
			"AlignErr":    5,
			"FCSErr":      10,
			"XmitErr":     15,
			"RcvErr":      20,
			"UnderSize":   25,
			"OutDiscards": 30,
			"SingleCol":   1,
			"MultiCol":    2,
			"LateCol":     3,
			"ExcesCol":    4,
			"CarriSen":    5,
			"Runts":       6,
			"Giants":      7,
			"SQETestErr":  8,
			"DeferredTx":  9,
			"IntMacTxEr":  10,
			"IntMacRxEr":  11,
			"SymbolErr":   12,
			"InDiscards":  13,
			"StompedCRC":  14,
		}

		actualValues := map[string]int64{
			"AlignErr":    eth10.AlignErr,
			"FCSErr":      eth10.FCSErr,
			"XmitErr":     eth10.XmitErr,
			"RcvErr":      eth10.RcvErr,
			"UnderSize":   eth10.UnderSize,
			"OutDiscards": eth10.OutDiscards,
			"SingleCol":   eth10.SingleCol,
			"MultiCol":    eth10.MultiCol,
			"LateCol":     eth10.LateCol,
			"ExcesCol":    eth10.ExcesCol,
			"CarriSen":    eth10.CarriSen,
			"Runts":       eth10.Runts,
			"Giants":      eth10.Giants,
			"SQETestErr":  eth10.SQETestErr,
			"DeferredTx":  eth10.DeferredTx,
			"IntMacTxEr":  eth10.IntMacTxEr,
			"IntMacRxEr":  eth10.IntMacRxEr,
			"SymbolErr":   eth10.SymbolErr,
			"InDiscards":  eth10.InDiscards,
			"StompedCRC":  eth10.StompedCRC,
		}

		for field, expected := range expectedValues {
			if actual := actualValues[field]; actual != expected {
				t.Errorf("Eth1/10 %s: expected %d, got %d", field, expected, actual)
			}
		}

		if !eth10.HasErrorData {
			t.Errorf("Eth1/10 should have error data")
		}
	} else {
		t.Error("Eth1/10 interface not found in parsed data")
	}
}
