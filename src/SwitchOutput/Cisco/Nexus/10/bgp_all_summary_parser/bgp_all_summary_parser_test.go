package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestParseBGPSummary tests parsing of BGP summary JSON
func TestParseBGPSummary(t *testing.T) {
	// Read the sample file from the parent directory
	sampleFilePath := filepath.Join("..", "show-bgp-all-summary.txt")
	data, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	// Parse the BGP summary
	entries, err := parseBGPSummary(string(data))
	if err != nil {
		t.Fatalf("Failed to parse BGP summary: %v", err)
	}
	
	// Basic validation checks
	if len(entries) == 0 {
		t.Fatal("No entries parsed from sample file")
	}
	
	// Validate first entry structure
	entry := entries[0]
	if entry.DataType != "cisco_nexus_bgp_summary" {
		t.Errorf("Expected DataType to be 'cisco_nexus_bgp_summary', got %q", entry.DataType)
	}
	
	// Validate VRF fields
	if entry.Message.VRFNameOut == "" {
		t.Error("VRFNameOut should not be empty")
	}
	
	if entry.Message.VRFRouterID == "" {
		t.Error("VRFRouterID should not be empty")
	}
	
	if entry.Message.VRFLocalAS == 0 {
		t.Error("VRFLocalAS should not be zero")
	}
	
	if entry.Message.ASNType == "" {
		t.Error("ASNType should not be empty")
	}
	
	// Validate address families
	if len(entry.Message.AddressFamilies) == 0 {
		t.Fatal("No address families parsed")
	}
	
	af := entry.Message.AddressFamilies[0]
	if af.AFName == "" {
		t.Error("AFName should not be empty")
	}
	
	// Validate neighbors
	if len(af.Neighbors) == 0 {
		t.Fatal("No neighbors parsed")
	}
	
	neighbor := af.Neighbors[0]
	if neighbor.NeighborID == "" {
		t.Error("NeighborID should not be empty")
	}
	
	if neighbor.SessionType == "" {
		t.Error("SessionType should be set")
	}
	
	fmt.Printf("Successfully parsed %d BGP summary entries with %d address families\n", 
		len(entries), len(entry.Message.AddressFamilies))
}

// TestISO8601DurationParsing tests the ISO 8601 duration parser
func TestISO8601DurationParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected *ParsedDuration
	}{
		{
			input: "P14W1D",
			expected: &ParsedDuration{
				Weeks:        14,
				Days:         1,
				TotalSeconds: (14*7+1) * 24 * 3600,
			},
		},
		{
			input: "P37W6D",
			expected: &ParsedDuration{
				Weeks:        37,
				Days:         6,
				TotalSeconds: (37*7+6) * 24 * 3600,
			},
		},
		{
			input: "P10W2D",
			expected: &ParsedDuration{
				Weeks:        10,
				Days:         2,
				TotalSeconds: (10*7+2) * 24 * 3600,
			},
		},
		{
			input:    "never",
			expected: nil,
		},
		{
			input:    "",
			expected: nil,
		},
	}
	
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := parseISO8601Duration(test.input)
			
			if test.expected == nil {
				if result != nil {
					t.Errorf("Expected nil for input %q, got %+v", test.input, result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("Expected non-nil result for input %q", test.input)
				return
			}
			
			if result.Weeks != test.expected.Weeks {
				t.Errorf("Weeks mismatch: expected %d, got %d", test.expected.Weeks, result.Weeks)
			}
			
			if result.Days != test.expected.Days {
				t.Errorf("Days mismatch: expected %d, got %d", test.expected.Days, result.Days)
			}
			
			if result.TotalSeconds != test.expected.TotalSeconds {
				t.Errorf("TotalSeconds mismatch: expected %d, got %d", 
					test.expected.TotalSeconds, result.TotalSeconds)
			}
		})
	}
}

// TestASNClassification tests the ASN classification function
func TestASNClassification(t *testing.T) {
	tests := []struct {
		asn      int
		expected string
	}{
		{64512, "private"},
		{65000, "private"},
		{65534, "private"},
		{65238, "private"},
		{1, "public"},
		{100, "public"},
		{64511, "public"},
		{65535, "public"},
		{4200000000, "private"},
		{4294967294, "private"},
		{4199999999, "public"},
	}
	
	for _, test := range tests {
		t.Run(fmt.Sprintf("ASN_%d", test.asn), func(t *testing.T) {
			result := classifyASN(test.asn)
			if result != test.expected {
				t.Errorf("Expected %q for ASN %d, got %q", test.expected, test.asn, result)
			}
		})
	}
}

// TestSessionTypeDetection tests session type determination
func TestSessionTypeDetection(t *testing.T) {
	tests := []struct {
		name         string
		neighborAS   int
		localAS      int
		expectedType string
	}{
		{
			name:         "iBGP session",
			neighborAS:   65238,
			localAS:      65238,
			expectedType: "iBGP",
		},
		{
			name:         "eBGP session",
			neighborAS:   64846,
			localAS:      65238,
			expectedType: "eBGP",
		},
		{
			name:         "Another eBGP session",
			neighborAS:   65239,
			localAS:      65238,
			expectedType: "eBGP",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := determineSessionType(test.neighborAS, test.localAS)
			if result != test.expectedType {
				t.Errorf("Expected SessionType %q, got %q", test.expectedType, result)
			}
		})
	}
}

// TestCommandJsonParsing tests that we can correctly parse a commands JSON file
func TestCommandJsonParsing(t *testing.T) {
	// Create a temporary JSON file with test commands
	tempDir := t.TempDir()
	commandsFilePath := filepath.Join(tempDir, "commands.json")
	
	// Define sample JSON content
	jsonContent := `{
		"commands": [
			{
				"name": "bgp-all-summary",
				"command": "show bgp all summary | json"
			},
			{
				"name": "other-command",
				"command": "show something else"
			}
		]
	}`
	
	// Write the content to a file
	if err := os.WriteFile(commandsFilePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test commands JSON file: %v", err)
	}
	
	// Read and parse the JSON file
	data, err := os.ReadFile(commandsFilePath)
	if err != nil {
		t.Fatalf("Failed to read commands JSON file: %v", err)
	}
	
	var cmdFile struct {
		Commands []struct {
			Name    string `json:"name"`
			Command string `json:"command"`
		} `json:"commands"`
	}
	
	if err := json.Unmarshal(data, &cmdFile); err != nil {
		t.Fatalf("Failed to parse commands JSON: %v", err)
	}
	
	// Verify we found our commands
	if len(cmdFile.Commands) != 2 {
		t.Errorf("Expected 2 commands in the JSON, but got %d", len(cmdFile.Commands))
	}
	
	// Find the bgp-all-summary command
	var bgpCmd string
	for _, cmd := range cmdFile.Commands {
		if cmd.Name == "bgp-all-summary" {
			bgpCmd = cmd.Command
			break
		}
	}
	
	if bgpCmd == "" {
		t.Error("Failed to find bgp-all-summary command in the JSON")
	} else if bgpCmd != "show bgp all summary | json" {
		t.Errorf("Expected command to be 'show bgp all summary | json', but got '%s'", bgpCmd)
	}
	
	fmt.Println("Successfully parsed commands JSON file")
}

// TestInvalidInput tests error handling for invalid input
func TestInvalidInput(t *testing.T) {
	// Test with invalid JSON
	_, err := parseBGPSummary("invalid json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	
	// Test with empty JSON - should fail to find VRF data
	_, err = parseBGPSummary("{}")
	if err == nil {
		t.Error("Expected error for empty JSON structure")
	}
	
	// Test with structure missing address families - parser is lenient and will create entry
	// but with no address families, which is technically valid (though unlikely in practice)
	entries, err := parseBGPSummary(`{"TABLE_vrf": {"ROW_vrf": {"vrf-name-out": "test", "vrf-router-id": "1.1.1.1", "vrf-local-as": 65000}}}`)
	if err != nil {
		t.Errorf("Parser should handle missing address families gracefully: %v", err)
	}
	if len(entries) > 0 && len(entries[0].Message.AddressFamilies) != 0 {
		t.Error("Expected zero address families for structure without TABLE_af")
	}
}

// TestDataTypeField specifically tests the data_type field for KQL queries
func TestDataTypeField(t *testing.T) {
	sampleFilePath := filepath.Join("..", "show-bgp-all-summary.txt")
	data, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	entries, err := parseBGPSummary(string(data))
	if err != nil {
		t.Fatalf("Failed to parse BGP summary: %v", err)
	}
	
	// Verify that each entry has the expected data_type for KQL queries
	for i, entry := range entries {
		if entry.DataType != "cisco_nexus_bgp_summary" {
			t.Errorf("Entry %d: expected DataType to be 'cisco_nexus_bgp_summary', got %q", 
				i, entry.DataType)
		}
	}
	
	fmt.Println("All entries have the correct data_type field")
}
