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
		{"Slot 1", "slot"},
		{"Power Supply 1", "power_supply"},
		{"Fan 1", "fan"},
		{"Ethernet1/1", "transceiver"},
		{"Ethernet1/49", "transceiver"},
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
		{`Chassis`, "Chassis"},
		{`"Nexus9000 C93180YC-FX Chassis"`, "Nexus9000 C93180YC-FX Chassis"},
		{`  "Chassis"  `, "Chassis"},
		{`""`, ""},
		{``, ""},
	}

	for _, test := range tests {
		result := cleanQuotes(test.input)
		if result != test.expected {
			t.Errorf("cleanQuotes(%s) = %s; expected %s", test.input, result, test.expected)
		}
	}
}

func TestParseInventory(t *testing.T) {
	// Sample input data
	sampleInput := `CONTOSO-TOR1# show inventory all 
NAME: "Chassis",  DESCR: "Nexus9000 C93180YC-FX Chassis"         
PID: N9K-C93180YC-FX     ,  VID: V04 ,  SN: FKE24000X1A          

NAME: "Slot 1",  DESCR: "48x10/25G/32G + 6x40/100G Ethernet/FC Module"
PID: N9K-C93180YC-FX     ,  VID: V04 ,  SN: FKE24000X1A          

NAME: "Power Supply 1",  DESCR: "Nexus9000 C93180YC-FX Chassis Power Supply"
PID: NXA-PAC-500W-PE     ,  VID: V01 ,  SN: ABC24001X2B          

NAME: "Power Supply 2",  DESCR: "Nexus9000 C93180YC-FX Chassis Power Supply"
PID: NXA-PAC-500W-PE     ,  VID: V01 ,  SN: ABC24001X3C          

NAME: "Fan 1",  DESCR: "Nexus9000 C93180YC-FX Chassis Fan Module"
PID: NXA-FAN-30CFM-F     ,  VID: V01 ,  SN: N/A                  

NAME: "Fan 2",  DESCR: "Nexus9000 C93180YC-FX Chassis Fan Module"
PID: NXA-FAN-30CFM-F     ,  VID: V01 ,  SN: N/A                  

NAME: Ethernet1/1,  DESCR: CISCO-AMPHENOL                          
PID: SFP-H25G-CU3M       ,  VID: NDCCGJ-C403,  SN: XYZ24001A1B          

NAME: Ethernet1/17,  DESCR: Siemon                                  
PID:                     ,  VID: S1S10F-V05.0M13,  SN: 12345678901A         

NAME: Ethernet1/49,  DESCR: CISCO-AMPHENOL                          
PID: QSFP-100G-CU1M      ,  VID: NDAAFF-C401,  SN: MNO24001Y9A-X        `

	// Parse the data
	inventory := parseInventory(sampleInput)

	// Should have 9 inventory items
	expectedCount := 9
	if len(inventory) != expectedCount {
		t.Errorf("Expected %d inventory items, got %d", expectedCount, len(inventory))
	}

	// Create map for easier testing
	inventoryMap := make(map[string]StandardizedEntry)
	for _, entry := range inventory {
		inventoryMap[entry.Message.Name] = entry
	}

	// Test Chassis
	if chassisEntry, exists := inventoryMap["Chassis"]; exists {
		chassis := chassisEntry.Message
		// Check standardized fields
		if chassisEntry.DataType != "cisco_nexus_inventory" {
			t.Errorf("Chassis data_type: expected 'cisco_nexus_inventory', got '%s'", chassisEntry.DataType)
		}
		if chassisEntry.Timestamp == "" {
			t.Errorf("Chassis timestamp should not be empty")
		}
		if chassisEntry.Date == "" {
			t.Errorf("Chassis date should not be empty")
		}
		// Check message fields
		if chassis.Description != "Nexus9000 C93180YC-FX Chassis" {
			t.Errorf("Chassis description: expected 'Nexus9000 C93180YC-FX Chassis', got '%s'", chassis.Description)
		}
		if chassis.ProductID != "N9K-C93180YC-FX" {
			t.Errorf("Chassis product ID: expected 'N9K-C93180YC-FX', got '%s'", chassis.ProductID)
		}
		if chassis.VersionID != "V04" {
			t.Errorf("Chassis version ID: expected 'V04', got '%s'", chassis.VersionID)
		}
		if chassis.SerialNumber != "FKE24000X1A" {
			t.Errorf("Chassis serial number: expected 'FKE24000X1A', got '%s'", chassis.SerialNumber)
		}
		if chassis.ComponentType != "chassis" {
			t.Errorf("Chassis component type: expected 'chassis', got '%s'", chassis.ComponentType)
		}
	} else {
		t.Error("Chassis not found in parsed data")
	}

	// Test Power Supply 1
	if ps1Entry, exists := inventoryMap["Power Supply 1"]; exists {
		ps1 := ps1Entry.Message
		if ps1.ComponentType != "power_supply" {
			t.Errorf("Power Supply 1 component type: expected 'power_supply', got '%s'", ps1.ComponentType)
		}
		if ps1.ProductID != "NXA-PAC-500W-PE" {
			t.Errorf("Power Supply 1 product ID: expected 'NXA-PAC-500W-PE', got '%s'", ps1.ProductID)
		}
	} else {
		t.Error("Power Supply 1 not found in parsed data")
	}

	// Test Fan 1 (with N/A serial number)
	if fan1Entry, exists := inventoryMap["Fan 1"]; exists {
		fan1 := fan1Entry.Message
		if fan1.ComponentType != "fan" {
			t.Errorf("Fan 1 component type: expected 'fan', got '%s'", fan1.ComponentType)
		}
		if fan1.SerialNumber != "N/A" {
			t.Errorf("Fan 1 serial number: expected 'N/A', got '%s'", fan1.SerialNumber)
		}
	} else {
		t.Error("Fan 1 not found in parsed data")
	}

	// Test Ethernet1/1 (transceiver)
	if eth1Entry, exists := inventoryMap["Ethernet1/1"]; exists {
		eth1 := eth1Entry.Message
		if eth1.ComponentType != "transceiver" {
			t.Errorf("Ethernet1/1 component type: expected 'transceiver', got '%s'", eth1.ComponentType)
		}
		if eth1.Description != "CISCO-AMPHENOL" {
			t.Errorf("Ethernet1/1 description: expected 'CISCO-AMPHENOL', got '%s'", eth1.Description)
		}
		if eth1.ProductID != "SFP-H25G-CU3M" {
			t.Errorf("Ethernet1/1 product ID: expected 'SFP-H25G-CU3M', got '%s'", eth1.ProductID)
		}
	} else {
		t.Error("Ethernet1/1 not found in parsed data")
	}

	// Test Ethernet1/17 (with empty PID)
	if eth17Entry, exists := inventoryMap["Ethernet1/17"]; exists {
		eth17 := eth17Entry.Message
		if eth17.ProductID != "" {
			t.Errorf("Ethernet1/17 product ID: expected empty string, got '%s'", eth17.ProductID)
		}
		if eth17.VersionID != "S1S10F-V05.0M13" {
			t.Errorf("Ethernet1/17 version ID: expected 'S1S10F-V05.0M13', got '%s'", eth17.VersionID)
		}
	} else {
		t.Error("Ethernet1/17 not found in parsed data")
	}

	// Test Ethernet1/49 (QSFP)
	if eth49Entry, exists := inventoryMap["Ethernet1/49"]; exists {
		eth49 := eth49Entry.Message
		if eth49.ProductID != "QSFP-100G-CU1M" {
			t.Errorf("Ethernet1/49 product ID: expected 'QSFP-100G-CU1M', got '%s'", eth49.ProductID)
		}
	} else {
		t.Error("Ethernet1/49 not found in parsed data")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(inventory)
	if err != nil {
		t.Errorf("Failed to marshal inventory to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledInventory []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledInventory)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if len(unmarshaledInventory) != len(inventory) {
		t.Errorf("JSON round-trip failed: expected %d inventory items, got %d", 
			len(inventory), len(unmarshaledInventory))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	// Create a temporary commands file
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "inventory",
				"command": "show inventory all"
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

	// Test finding inventory command
	command, err := findInventoryCommand(config)
	if err != nil {
		t.Errorf("Failed to find inventory command: %v", err)
	}

	expectedCommand := "show inventory all"
	if command != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, command)
	}
}