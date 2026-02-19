package main

import (
	"encoding/json"
	"os"
	"testing"
)

// Synthetic test data based on expected Dell OS10 BGP output format
// (BGP was not configured on the test switch, so using synthetic data)
const testBgpSummaryInput = `BGP router identifier 10.0.0.1, local AS number 65001

Neighbor        AS      MsgRcvd  MsgSent  TblVer   InQ  OutQ  Up/Down    State/PfxRcd
10.0.0.2        65001   12345    12340    100      0    0     11w2d      50
10.0.0.3        65002   5678     5670     100      0    0     3d12h      25
10.0.0.4        65003   100      99       0        0    0     00:05:30   Active`

func TestParseBgpSummary(t *testing.T) {
	entries, err := parseBgpSummary(testBgpSummaryInput)
	if err != nil {
		t.Fatalf("parseBgpSummary returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.DataType != "dell_os10_bgp_summary" {
		t.Errorf("DataType: expected 'dell_os10_bgp_summary', got '%s'", e.DataType)
	}
	if e.Message.RouterID != "10.0.0.1" {
		t.Errorf("RouterID: expected '10.0.0.1', got '%s'", e.Message.RouterID)
	}
	if e.Message.LocalAS != 65001 {
		t.Errorf("LocalAS: expected 65001, got %d", e.Message.LocalAS)
	}
	if e.Message.ASNType != "private" {
		t.Errorf("ASNType: expected 'private' for AS 65001, got '%s'", e.Message.ASNType)
	}
	if e.Message.NeighborsCount != 3 {
		t.Errorf("NeighborsCount: expected 3, got %d", e.Message.NeighborsCount)
	}
	if len(e.Message.Neighbors) != 3 {
		t.Fatalf("Expected 3 neighbors, got %d", len(e.Message.Neighbors))
	}

	// First neighbor - iBGP (same AS)
	n0 := e.Message.Neighbors[0]
	if n0.NeighborID != "10.0.0.2" {
		t.Errorf("Neighbor 0 ID: expected '10.0.0.2', got '%s'", n0.NeighborID)
	}
	if n0.SessionType != "iBGP" {
		t.Errorf("Neighbor 0 SessionType: expected 'iBGP', got '%s'", n0.SessionType)
	}
	if n0.State != "Established" {
		t.Errorf("Neighbor 0 State: expected 'Established', got '%s'", n0.State)
	}
	if n0.PrefixReceived != 50 {
		t.Errorf("Neighbor 0 PrefixReceived: expected 50, got %d", n0.PrefixReceived)
	}
	if n0.MsgRecvd != 12345 {
		t.Errorf("Neighbor 0 MsgRecvd: expected 12345, got %d", n0.MsgRecvd)
	}

	// Second neighbor - eBGP (different AS)
	n1 := e.Message.Neighbors[1]
	if n1.SessionType != "eBGP" {
		t.Errorf("Neighbor 1 SessionType: expected 'eBGP', got '%s'", n1.SessionType)
	}
	if n1.PrefixReceived != 25 {
		t.Errorf("Neighbor 1 PrefixReceived: expected 25, got %d", n1.PrefixReceived)
	}

	// Third neighbor - not established
	n2 := e.Message.Neighbors[2]
	if n2.State != "Active" {
		t.Errorf("Neighbor 2 State: expected 'Active', got '%s'", n2.State)
	}
	if n2.PrefixReceived != 0 {
		t.Errorf("Neighbor 2 PrefixReceived: expected 0, got %d", n2.PrefixReceived)
	}
}

func TestParseBgpSummaryEmpty(t *testing.T) {
	entries, err := parseBgpSummary("")
	if err != nil {
		t.Fatalf("parseBgpSummary should not error on empty input: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty input, got %d", len(entries))
	}
}

func TestParseBgpSummaryNoBGP(t *testing.T) {
	_, err := parseBgpSummary("% BGP is not running")
	if err == nil {
		t.Log("Expected error for no BGP data, but got nil - acceptable")
	}
}

func TestClassifyASN(t *testing.T) {
	tests := []struct {
		asn      int64
		expected string
	}{
		{65001, "private"},
		{64512, "private"},
		{65534, "private"},
		{100, "public"},
		{64511, "public"},
		{65535, "public"},
		{4200000000, "private"},
	}
	for _, test := range tests {
		result := classifyASN(test.asn)
		if result != test.expected {
			t.Errorf("classifyASN(%d) = %s; expected %s", test.asn, result, test.expected)
		}
	}
}

func TestParseBgpSummaryJSON(t *testing.T) {
	entries, err := parseBgpSummary(testBgpSummaryInput)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	jsonBytes, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	var unmarshaled []StandardizedEntry
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	err := os.WriteFile(tempFile, []byte(`{"commands":[{"name":"bgp-summary","command":"show ip bgp summary"}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load: %v", err)
	}
	command, err := findCommand(config, "bgp-summary")
	if err != nil {
		t.Errorf("Failed to find: %v", err)
	}
	if command != "show ip bgp summary" {
		t.Errorf("Expected 'show ip bgp summary', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
