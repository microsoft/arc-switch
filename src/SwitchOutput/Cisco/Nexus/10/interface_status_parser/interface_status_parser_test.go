package interface_status_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseInterfaceStatus(t *testing.T) {
	// Read the sample input file
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	// Parse the interface status
	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Validate data type
	if entry.DataType != "cisco_nexus_interface_status" {
		t.Errorf("Expected data_type 'cisco_nexus_interface_status', got %s", entry.DataType)
	}

	// Validate timestamp format
	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}

	// Validate date format
	if entry.Date == "" {
		t.Error("Date should not be empty")
	}

	// Validate total number of interfaces
	// mgmt0 (1) + Ethernet (39) + Port-channel (3) + Loopback (1) + Vlan (23) = 67
	expectedInterfaceCount := 67
	if len(entry.Message.Interfaces) != expectedInterfaceCount {
		t.Errorf("Expected %d interfaces, got %d", expectedInterfaceCount, len(entry.Message.Interfaces))
	}
}

func TestMgmtInterface(t *testing.T) {
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	// Find mgmt0 interface
	var mgmt0 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "mgmt0" {
			mgmt0 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if mgmt0 == nil {
		t.Fatal("mgmt0 interface not found")
	}

	if mgmt0.Name != "BMCMgmt_switch_vir" {
		t.Errorf("Expected mgmt0 Name 'BMCMgmt_switch_vir', got '%s'", mgmt0.Name)
	}
	if mgmt0.Status != "disabled" {
		t.Errorf("Expected mgmt0 Status 'disabled', got '%s'", mgmt0.Status)
	}
	if mgmt0.Vlan != "routed" {
		t.Errorf("Expected mgmt0 Vlan 'routed', got '%s'", mgmt0.Vlan)
	}
	if mgmt0.Duplex != "auto" {
		t.Errorf("Expected mgmt0 Duplex 'auto', got '%s'", mgmt0.Duplex)
	}
	if mgmt0.Speed != "auto" {
		t.Errorf("Expected mgmt0 Speed 'auto', got '%s'", mgmt0.Speed)
	}
	if mgmt0.Type != "--" {
		t.Errorf("Expected mgmt0 Type '--', got '%s'", mgmt0.Type)
	}
}

func TestEthernetInterfaces(t *testing.T) {
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	// Test Eth1/1
	var eth1_1 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Eth1/1" {
			eth1_1 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if eth1_1 == nil {
		t.Fatal("Eth1/1 interface not found")
	}

	if eth1_1.Name != "Switched-Compute" {
		t.Errorf("Expected Eth1/1 Name 'Switched-Compute', got '%s'", eth1_1.Name)
	}
	if eth1_1.Status != "connected" {
		t.Errorf("Expected Eth1/1 Status 'connected', got '%s'", eth1_1.Status)
	}
	if eth1_1.Vlan != "trunk" {
		t.Errorf("Expected Eth1/1 Vlan 'trunk', got '%s'", eth1_1.Vlan)
	}
	if eth1_1.Duplex != "full" {
		t.Errorf("Expected Eth1/1 Duplex 'full', got '%s'", eth1_1.Duplex)
	}
	if eth1_1.Speed != "100G" {
		t.Errorf("Expected Eth1/1 Speed '100G', got '%s'", eth1_1.Speed)
	}
	if eth1_1.Type != "QSFP-100G-PCC" {
		t.Errorf("Expected Eth1/1 Type 'QSFP-100G-PCC', got '%s'", eth1_1.Type)
	}

	// Test Eth1/7 (notconnected interface)
	var eth1_7 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Eth1/7" {
			eth1_7 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if eth1_7 == nil {
		t.Fatal("Eth1/7 interface not found")
	}

	if eth1_7.Status != "notconnec" {
		t.Errorf("Expected Eth1/7 Status 'notconnec', got '%s'", eth1_7.Status)
	}
	if eth1_7.Duplex != "auto" {
		t.Errorf("Expected Eth1/7 Duplex 'auto', got '%s'", eth1_7.Duplex)
	}
	if eth1_7.Speed != "auto" {
		t.Errorf("Expected Eth1/7 Speed 'auto', got '%s'", eth1_7.Speed)
	}

	// Test Eth1/33 (routed interface)
	var eth1_33 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Eth1/33" {
			eth1_33 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if eth1_33 == nil {
		t.Fatal("Eth1/33 interface not found")
	}

	if eth1_33.Name != "P2P_IBGP" {
		t.Errorf("Expected Eth1/33 Name 'P2P_IBGP', got '%s'", eth1_33.Name)
	}
	if eth1_33.Vlan != "routed" {
		t.Errorf("Expected Eth1/33 Vlan 'routed', got '%s'", eth1_33.Vlan)
	}
	if eth1_33.Type != "QSFP-100G-CR4" {
		t.Errorf("Expected Eth1/33 Type 'QSFP-100G-CR4', got '%s'", eth1_33.Type)
	}

	// Test Eth1/36/1 (breakout interface)
	var eth1_36_1 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Eth1/36/1" {
			eth1_36_1 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if eth1_36_1 == nil {
		t.Fatal("Eth1/36/1 interface not found")
	}

	if eth1_36_1.Name != "P2P_Border1" {
		t.Errorf("Expected Eth1/36/1 Name 'P2P_Border1', got '%s'", eth1_36_1.Name)
	}
	if eth1_36_1.Speed != "10G" {
		t.Errorf("Expected Eth1/36/1 Speed '10G', got '%s'", eth1_36_1.Speed)
	}
	if eth1_36_1.Type != "QSFP-40G-CSR4" {
		t.Errorf("Expected Eth1/36/1 Type 'QSFP-40G-CSR4', got '%s'", eth1_36_1.Type)
	}
}

func TestPortChannelInterfaces(t *testing.T) {
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	// Test Po50
	var po50 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Po50" {
			po50 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if po50 == nil {
		t.Fatal("Po50 interface not found")
	}

	if po50.Name != "VPC:P2P_IBGP" {
		t.Errorf("Expected Po50 Name 'VPC:P2P_IBGP', got '%s'", po50.Name)
	}
	if po50.Status != "connected" {
		t.Errorf("Expected Po50 Status 'connected', got '%s'", po50.Status)
	}
	if po50.Vlan != "routed" {
		t.Errorf("Expected Po50 Vlan 'routed', got '%s'", po50.Vlan)
	}
	if po50.Speed != "100G" {
		t.Errorf("Expected Po50 Speed '100G', got '%s'", po50.Speed)
	}
	if po50.Type != "--" {
		t.Errorf("Expected Po50 Type '--', got '%s'", po50.Type)
	}

	// Test Po102
	var po102 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Po102" {
			po102 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if po102 == nil {
		t.Fatal("Po102 interface not found")
	}

	if po102.Name != "VPC:TOR_BMC" {
		t.Errorf("Expected Po102 Name 'VPC:TOR_BMC', got '%s'", po102.Name)
	}
	if po102.Vlan != "trunk" {
		t.Errorf("Expected Po102 Vlan 'trunk', got '%s'", po102.Vlan)
	}
	if po102.Speed != "10G" {
		t.Errorf("Expected Po102 Speed '10G', got '%s'", po102.Speed)
	}
}

func TestLoopbackInterface(t *testing.T) {
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	// Test Lo0
	var lo0 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Lo0" {
			lo0 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if lo0 == nil {
		t.Fatal("Lo0 interface not found")
	}

	if lo0.Name != "Loopback0_Tor2" {
		t.Errorf("Expected Lo0 Name 'Loopback0_Tor2', got '%s'", lo0.Name)
	}
	if lo0.Status != "connected" {
		t.Errorf("Expected Lo0 Status 'connected', got '%s'", lo0.Status)
	}
	if lo0.Vlan != "routed" {
		t.Errorf("Expected Lo0 Vlan 'routed', got '%s'", lo0.Vlan)
	}
	if lo0.Type != "--" {
		t.Errorf("Expected Lo0 Type '--', got '%s'", lo0.Type)
	}
}

func TestVlanInterfaces(t *testing.T) {
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	// Test Vlan1 (down interface)
	var vlan1 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Vlan1" {
			vlan1 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if vlan1 == nil {
		t.Fatal("Vlan1 interface not found")
	}

	if vlan1.Name != "--" {
		t.Errorf("Expected Vlan1 Name '--', got '%s'", vlan1.Name)
	}
	if vlan1.Status != "down" {
		t.Errorf("Expected Vlan1 Status 'down', got '%s'", vlan1.Status)
	}
	if vlan1.Vlan != "routed" {
		t.Errorf("Expected Vlan1 Vlan 'routed', got '%s'", vlan1.Vlan)
	}

	// Test Vlan6
	var vlan6 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Vlan6" {
			vlan6 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if vlan6 == nil {
		t.Fatal("Vlan6 interface not found")
	}

	if vlan6.Name != "HNVPA_6" {
		t.Errorf("Expected Vlan6 Name 'HNVPA_6', got '%s'", vlan6.Name)
	}
	if vlan6.Status != "connected" {
		t.Errorf("Expected Vlan6 Status 'connected', got '%s'", vlan6.Status)
	}

	// Test Vlan516 (last vlan)
	var vlan516 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Vlan516" {
			vlan516 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if vlan516 == nil {
		t.Fatal("Vlan516 interface not found")
	}

	if vlan516.Name != "L3forward_516" {
		t.Errorf("Expected Vlan516 Name 'L3forward_516', got '%s'", vlan516.Name)
	}
}

func TestJSONSerialization(t *testing.T) {
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(entries[0])
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Deserialize back
	var entry StandardizedEntry
	err = json.Unmarshal(jsonData, &entry)
	if err != nil {
		t.Fatalf("Failed to deserialize from JSON: %v", err)
	}

	// Validate the round-trip
	if entry.DataType != "cisco_nexus_interface_status" {
		t.Errorf("Round-trip failed: data_type mismatch")
	}

	if len(entry.Message.Interfaces) != 67 {
		t.Errorf("Round-trip failed: expected 67 interfaces, got %d", len(entry.Message.Interfaces))
	}
}

func TestUnifiedParserInterface(t *testing.T) {
	parser := &UnifiedParser{}

	// Verify description
	desc := parser.GetDescription()
	if desc == "" {
		t.Error("GetDescription should return a non-empty string")
	}

	// Test parsing
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	result, err := parser.Parse(inputData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify result type
	entries, ok := result.([]StandardizedEntry)
	if !ok {
		t.Fatal("Parse should return []StandardizedEntry")
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

func TestDisabledInterface(t *testing.T) {
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	// Test Eth1/36/3 (disabled interface)
	var eth1_36_3 *InterfaceEntry
	for i, iface := range entries[0].Message.Interfaces {
		if iface.Port == "Eth1/36/3" {
			eth1_36_3 = &entries[0].Message.Interfaces[i]
			break
		}
	}

	if eth1_36_3 == nil {
		t.Fatal("Eth1/36/3 interface not found")
	}

	if eth1_36_3.Name != "--" {
		t.Errorf("Expected Eth1/36/3 Name '--', got '%s'", eth1_36_3.Name)
	}
	if eth1_36_3.Status != "disabled" {
		t.Errorf("Expected Eth1/36/3 Status 'disabled', got '%s'", eth1_36_3.Status)
	}
	if eth1_36_3.Vlan != "routed" {
		t.Errorf("Expected Eth1/36/3 Vlan 'routed', got '%s'", eth1_36_3.Vlan)
	}
}

func TestInterfaceTypesByCategory(t *testing.T) {
	inputData, err := os.ReadFile("show-interface-status.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseInterfaceStatus(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse interface status: %v", err)
	}

	// Count interfaces by category
	var mgmtCount, ethCount, poCount, loCount, vlanCount int
	for _, iface := range entries[0].Message.Interfaces {
		switch {
		case len(iface.Port) >= 4 && iface.Port[:4] == "mgmt":
			mgmtCount++
		case len(iface.Port) >= 3 && iface.Port[:3] == "Eth":
			ethCount++
		case len(iface.Port) >= 2 && iface.Port[:2] == "Po":
			poCount++
		case len(iface.Port) >= 2 && iface.Port[:2] == "Lo":
			loCount++
		case len(iface.Port) >= 4 && iface.Port[:4] == "Vlan":
			vlanCount++
		}
	}

	if mgmtCount != 1 {
		t.Errorf("Expected 1 mgmt interface, got %d", mgmtCount)
	}
	if ethCount != 39 {
		t.Errorf("Expected 39 Ethernet interfaces, got %d", ethCount)
	}
	if poCount != 3 {
		t.Errorf("Expected 3 Port-channel interfaces, got %d", poCount)
	}
	if loCount != 1 {
		t.Errorf("Expected 1 Loopback interface, got %d", loCount)
	}
	if vlanCount != 23 {
		t.Errorf("Expected 23 VLAN interfaces, got %d", vlanCount)
	}
}
