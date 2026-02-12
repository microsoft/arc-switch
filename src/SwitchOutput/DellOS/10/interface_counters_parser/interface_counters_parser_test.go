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
		{"Ethernet 1/1/1", "ethernet"},
		{"ethernet1/1/1", "ethernet"},
		{"Port-channel 10", "port-channel"},
		{"port-channel10", "port-channel"},
		{"Vlan 100", "vlan"},
		{"vlan100", "vlan"},
		{"Management 1/1/1", "management"},
		{"Loopback 0", "loopback"},
		{"Unknown", "unknown"},
	}

	for _, test := range tests {
		result := parseInterfaceType(test.interfaceName)
		if result != test.expected {
			t.Errorf("parseInterfaceType(%s) = %s; expected %s", test.interfaceName, result, test.expected)
		}
	}
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"123456", 123456},
		{"0", 0},
		{"--", -1},
		{"", -1},
		{"  123456  ", 123456},
		{"1,234,567", 1234567},
		{"invalid", -1},
	}

	for _, test := range tests {
		result := parseInt64(test.input)
		if result != test.expected {
			t.Errorf("parseInt64(%s) = %d; expected %d", test.input, result, test.expected)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"12345", 12345},
		{"0", 0},
		{"--", -1},
		{"", -1},
		{"  12345  ", 12345},
		{"1,234", 1234},
		{"invalid", -1},
	}

	for _, test := range tests {
		result := parseInt(test.input)
		if result != test.expected {
			t.Errorf("parseInt(%s) = %d; expected %d", test.input, result, test.expected)
		}
	}
}

func TestParseInterfaceCounters(t *testing.T) {
	// Sample Dell OS10 show interface output
	sampleInput := `Ethernet 1/1/1 is up, line protocol is up
Hardware is DellEth, address is 90:b1:1c:f4:a6:00
    Interface index is 1
    Internet address is not assigned
    MTU 1500 bytes, IP MTU 1500 bytes
    LineSpeed 100G
    Input statistics:
        258756070469 packets, 224206026489044 octets
        3322520 Multicasts, 5843785 Broadcasts, 258746758082 Unicasts
        0 runts, 0 giants, 104 throttles
        0 CRC, 0 overrun, 0 discarded
    Output statistics:
        464017689029 packets, 460659024245083 octets
        114945286 Multicasts, 27839366 Broadcasts, 463860550477 Unicasts
        0 throttles, 296897 discarded, 0 Collisions
    Rate Info:
        Input 11 Mbits/sec, 1806 packets/sec, 0% of line rate
        Output 14 Mbits/sec, 2102 packets/sec, 0% of line rate

Ethernet 1/1/2 is down, line protocol is down
Hardware is DellEth, address is 90:b1:1c:f4:a6:01
    Interface index is 2
    Input statistics:
        0 packets, 0 octets
        0 Multicasts, 0 Broadcasts, 0 Unicasts
        0 runts, 0 giants, 0 throttles
        0 CRC, 0 overrun, 0 discarded
    Output statistics:
        0 packets, 0 octets
        0 Multicasts, 0 Broadcasts, 0 Unicasts
        0 throttles, 0 discarded, 0 Collisions`

	interfaces := parseInterfaceCounters(sampleInput)

	if len(interfaces) != 2 {
		t.Errorf("Expected 2 interfaces, got %d", len(interfaces))
	}

	// Find Ethernet 1/1/1 and verify
	for _, entry := range interfaces {
		if entry.Message.InterfaceName == "Ethernet 1/1/1" {
			if entry.DataType != "dell_os10_interface_counters" {
				t.Errorf("data_type: expected 'dell_os10_interface_counters', got '%s'", entry.DataType)
			}
			if entry.Message.Status != "up" {
				t.Errorf("Status: expected 'up', got '%s'", entry.Message.Status)
			}
			if entry.Message.LineProtocol != "up" {
				t.Errorf("LineProtocol: expected 'up', got '%s'", entry.Message.LineProtocol)
			}
			if entry.Message.InterfaceType != "ethernet" {
				t.Errorf("InterfaceType: expected 'ethernet', got '%s'", entry.Message.InterfaceType)
			}
			if entry.Message.InPackets != 258756070469 {
				t.Errorf("InPackets: expected 258756070469, got %d", entry.Message.InPackets)
			}
			if entry.Message.InOctets != 224206026489044 {
				t.Errorf("InOctets: expected 224206026489044, got %d", entry.Message.InOctets)
			}
			if entry.Message.InMcastPkts != 3322520 {
				t.Errorf("InMcastPkts: expected 3322520, got %d", entry.Message.InMcastPkts)
			}
			if entry.Message.InBcastPkts != 5843785 {
				t.Errorf("InBcastPkts: expected 5843785, got %d", entry.Message.InBcastPkts)
			}
			if entry.Message.OutPackets != 464017689029 {
				t.Errorf("OutPackets: expected 464017689029, got %d", entry.Message.OutPackets)
			}
			if entry.Message.OutDiscarded != 296897 {
				t.Errorf("OutDiscarded: expected 296897, got %d", entry.Message.OutDiscarded)
			}
			if entry.Message.InputRateMbps != 11 {
				t.Errorf("InputRateMbps: expected 11, got %d", entry.Message.InputRateMbps)
			}
			if !entry.Message.HasIngressData {
				t.Error("HasIngressData should be true")
			}
			if !entry.Message.HasEgressData {
				t.Error("HasEgressData should be true")
			}
			break
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

func TestParseInterfaceCountersEmptyInput(t *testing.T) {
	interfaces := parseInterfaceCounters("")
	if len(interfaces) != 0 {
		t.Errorf("Expected 0 interfaces for empty input, got %d", len(interfaces))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "interface-counter",
				"command": "show interface"
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

	command, err := findInterfaceCountersCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show interface" {
		t.Errorf("Expected command 'show interface', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
