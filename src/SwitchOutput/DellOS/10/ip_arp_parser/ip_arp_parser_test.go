package ip_arp_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDetermineInterfaceType(t *testing.T) {
	tests := []struct {
		interfaceName string
		expected      string
	}{
		{"vlan100", "vlan"},
		{"Vlan100", "vlan"},
		{"ethernet1/1/1", "ethernet"},
		{"Ethernet1/1/1", "ethernet"},
		{"port-channel10", "port-channel"},
		{"Port-channel10", "port-channel"},
		{"mgmt1/1/1", "management"},
		{"Management1/1/1", "management"},
		{"loopback0", "loopback"},
		{"Loopback0", "loopback"},
		{"virtual-network1", "virtual-network"},
		{"unknown", "other"},
	}

	for _, test := range tests {
		result := determineInterfaceType(test.interfaceName)
		if result != test.expected {
			t.Errorf("determineInterfaceType(%s) = %s; expected %s", test.interfaceName, result, test.expected)
		}
	}
}

func TestParseARP(t *testing.T) {
	// Sample Dell OS10 show ip arp output
	sampleInput := `Address          Hardware address      Interface          Egress Interface
-------          ----------------      ---------          ----------------
192.168.2.2      90:b1:1c:f4:a6:e6    ethernet1/1/49:1   ethernet1/1/49:1
10.1.1.1         00:1a:2b:3c:4d:5e    vlan100            vlan100
172.16.0.1       aa:bb:cc:dd:ee:ff    port-channel10     port-channel10`

	entries, err := parseARP(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse ARP table: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// Verify first entry
	if len(entries) > 0 {
		entry := entries[0]
		if entry.DataType != "dell_os10_arp_entry" {
			t.Errorf("data_type: expected 'dell_os10_arp_entry', got '%s'", entry.DataType)
		}
		if entry.Message.IPAddress != "192.168.2.2" {
			t.Errorf("IPAddress: expected '192.168.2.2', got '%s'", entry.Message.IPAddress)
		}
		if entry.Message.HardwareAddress != "90:b1:1c:f4:a6:e6" {
			t.Errorf("HardwareAddress: expected '90:b1:1c:f4:a6:e6', got '%s'", entry.Message.HardwareAddress)
		}
		if entry.Message.Interface != "ethernet1/1/49:1" {
			t.Errorf("Interface: expected 'ethernet1/1/49:1', got '%s'", entry.Message.Interface)
		}
		if entry.Message.InterfaceType != "ethernet" {
			t.Errorf("InterfaceType: expected 'ethernet', got '%s'", entry.Message.InterfaceType)
		}
		if entry.Message.EgressInterface != "ethernet1/1/49:1" {
			t.Errorf("EgressInterface: expected 'ethernet1/1/49:1', got '%s'", entry.Message.EgressInterface)
		}
	}

	// Verify VLAN interface type
	if len(entries) > 1 {
		if entries[1].Message.InterfaceType != "vlan" {
			t.Errorf("InterfaceType for vlan: expected 'vlan', got '%s'", entries[1].Message.InterfaceType)
		}
	}

	// Verify port-channel interface type
	if len(entries) > 2 {
		if entries[2].Message.InterfaceType != "port-channel" {
			t.Errorf("InterfaceType for port-channel: expected 'port-channel', got '%s'", entries[2].Message.InterfaceType)
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

func TestParseARPNoHeader(t *testing.T) {
	// Input without header
	sampleInput := `Some random text
without a proper header`

	_, err := parseARP(sampleInput)
	if err == nil {
		t.Error("Expected error for input without header")
	}
}

func TestParseARPEmptyTable(t *testing.T) {
	// Input with header but no entries
	sampleInput := `Address          Hardware address      Interface          Egress Interface
-------          ----------------      ---------          ----------------`

	entries, err := parseARP(sampleInput)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty table, got %d", len(entries))
	}
}

func TestParseARPWithSummary(t *testing.T) {
	// Input with summary lines at the end
	sampleInput := `Address          Hardware address      Interface          Egress Interface
-------          ----------------      ---------          ----------------
192.168.2.2      90:b1:1c:f4:a6:e6    ethernet1/1/49:1   ethernet1/1/49:1

Total Entries: 1
Static Entries: 0`

	entries, err := parseARP(sampleInput)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (excluding summary lines), got %d", len(entries))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "arp-table",
				"command": "show ip arp"
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

	command, err := findArpCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show ip arp" {
		t.Errorf("Expected command 'show ip arp', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
