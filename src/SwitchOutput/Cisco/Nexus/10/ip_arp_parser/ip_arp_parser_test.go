package main

import (
	"encoding/json"
	"testing"
)

// Sample ARP table data for testing
const sampleARPData = `RR1-S46-R14-93180hl-22-1b# show ip arp 

Flags: * - Adjacencies learnt on non-active FHRP router
       + - Adjacencies synced via CFSoE
       # - Adjacencies Throttled for Glean
       CP - Added via L2RIB, Control plane Adjacencies
       PS - Added via L2RIB, Peer Sync
       RO - Re-Originated Peer Sync Entry
       D - Static Adjacencies attached to down interface

IP ARP Table for context default
Total number of entries: 102
Address         Age       MAC Address     Interface       Flags
100.69.161.1    00:17:58  0000.0c9f.f0c9  Vlan201                  
100.69.161.2    00:17:58  5ca6.2dbb.64a7  Vlan201                  
100.69.161.75   00:03:39  02ec.a040.0001  Vlan201         +        
100.71.83.66    00:08:54  f402.70e6.4220  Vlan125         +        
100.71.83.17    00:16:49  5ca6.2dbb.64a7  port-channel50           
100.71.83.13    00:18:12  689e.0baf.913b  Ethernet1/47             
100.69.160.4    00:00:19  0c42.a11c.d59e  Vlan7           +        `

func TestParseARP(t *testing.T) {
	entries, err := parseARP(sampleARPData)
	if err != nil {
		t.Fatalf("parseARP failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("Expected to parse some entries, got none")
	}

	// Test specific entries
	expectedEntries := map[string]struct {
		macAddress    string
		interfaceType string
		cfsoeSync     bool
	}{
		"100.69.161.1":  {"0000.0c9f.f0c9", "vlan", false},
		"100.69.161.75": {"02ec.a040.0001", "vlan", true},
		"100.71.83.17":  {"5ca6.2dbb.64a7", "port-channel", false},
		"100.71.83.13":  {"689e.0baf.913b", "ethernet", false},
	}

	entryMap := make(map[string]ARPTableEntry)
	for _, entry := range entries {
		entryMap[entry.IPAddress] = entry
	}

	for ip, expected := range expectedEntries {
		entry, exists := entryMap[ip]
		if !exists {
			t.Errorf("Expected entry for IP %s not found", ip)
			continue
		}

		if entry.MACAddress != expected.macAddress {
			t.Errorf("For IP %s, expected MAC %s, got %s", ip, expected.macAddress, entry.MACAddress)
		}

		if entry.InterfaceType != expected.interfaceType {
			t.Errorf("For IP %s, expected interface type %s, got %s", ip, expected.interfaceType, entry.InterfaceType)
		}

		if entry.CFSoESync != expected.cfsoeSync {
			t.Errorf("For IP %s, expected CFSoESync %v, got %v", ip, expected.cfsoeSync, entry.CFSoESync)
		}

		// Verify data type is set correctly
		if entry.DataType != "cisco_nexus_arp_entry" {
			t.Errorf("Expected DataType 'cisco_nexus_arp_entry', got %s", entry.DataType)
		}

		// Verify timestamp and date are set
		if entry.Timestamp == "" {
			t.Errorf("Timestamp should not be empty")
		}

		if entry.Date == "" {
			t.Errorf("Date should not be empty")
		}
	}
}

func TestParseARPLine(t *testing.T) {
	testCases := []struct {
		line         string
		expectedIP   string
		expectedMAC  string
		expectedIntf string
		expectedFlag bool // CFSoESync flag
	}{
		{
			line:         "100.69.161.75   00:03:39  02ec.a040.0001  Vlan201         +        ",
			expectedIP:   "100.69.161.75",
			expectedMAC:  "02ec.a040.0001",
			expectedIntf: "Vlan201",
			expectedFlag: true,
		},
		{
			line:         "100.69.161.1    00:17:58  0000.0c9f.f0c9  Vlan201                  ",
			expectedIP:   "100.69.161.1",
			expectedMAC:  "0000.0c9f.f0c9",
			expectedIntf: "Vlan201",
			expectedFlag: false,
		},
	}

	for _, tc := range testCases {
		entry := parseARPLine(tc.line, "test_timestamp", "test_date")
		if entry == nil {
			t.Errorf("Failed to parse line: %s", tc.line)
			continue
		}

		if entry.IPAddress != tc.expectedIP {
			t.Errorf("Expected IP %s, got %s", tc.expectedIP, entry.IPAddress)
		}

		if entry.MACAddress != tc.expectedMAC {
			t.Errorf("Expected MAC %s, got %s", tc.expectedMAC, entry.MACAddress)
		}

		if entry.Interface != tc.expectedIntf {
			t.Errorf("Expected interface %s, got %s", tc.expectedIntf, entry.Interface)
		}

		if entry.CFSoESync != tc.expectedFlag {
			t.Errorf("Expected CFSoESync %v, got %v", tc.expectedFlag, entry.CFSoESync)
		}
	}
}

func TestDetermineInterfaceType(t *testing.T) {
	testCases := []struct {
		interfaceName string
		expectedType  string
	}{
		{"Vlan201", "vlan"},
		{"Ethernet1/47", "ethernet"},
		{"port-channel50", "port-channel"},
		{"mgmt0", "management"},
		{"Tunnel1", "tunnel"},
		{"Loopback0", "loopback"},
		{"unknown-interface", "other"},
	}

	for _, tc := range testCases {
		result := determineInterfaceType(tc.interfaceName)
		if result != tc.expectedType {
			t.Errorf("For interface %s, expected type %s, got %s", tc.interfaceName, tc.expectedType, result)
		}
	}
}

func TestJSONOutput(t *testing.T) {
	entries, err := parseARP(sampleARPData)
	if err != nil {
		t.Fatalf("parseARP failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("Expected to parse some entries, got none")
	}

	// Test that we can marshal to JSON
	for _, entry := range entries {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			t.Errorf("Failed to marshal entry to JSON: %v", err)
		}

		// Test that we can unmarshal back
		var unmarshaled ARPTableEntry
		err = json.Unmarshal(jsonData, &unmarshaled)
		if err != nil {
			t.Errorf("Failed to unmarshal JSON: %v", err)
		}

		if unmarshaled.IPAddress != entry.IPAddress {
			t.Errorf("JSON round-trip failed for IP address")
		}
	}
}

func TestParseFlags(t *testing.T) {
	entry := &ARPTableEntry{}
	
	// Test various flag combinations
	parseFlags(entry, "+")
	if !entry.CFSoESync {
		t.Error("Expected CFSoESync to be true")
	}

	entry = &ARPTableEntry{}
	parseFlags(entry, "*")
	if !entry.NonActiveFHRP {
		t.Error("Expected NonActiveFHRP to be true")
	}

	entry = &ARPTableEntry{}
	parseFlags(entry, "CP")
	if !entry.ControlPlaneL2RIB {
		t.Error("Expected ControlPlaneL2RIB to be true")
	}

	// Test multiple flags
	entry = &ARPTableEntry{}
	parseFlags(entry, "+ *")
	if !entry.CFSoESync || !entry.NonActiveFHRP {
		t.Error("Expected both CFSoESync and NonActiveFHRP to be true")
	}
}
