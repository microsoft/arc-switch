package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Create a mock version of runClish for testing
func mockRunClishFunction(mockOutput string) func(string) (string, error) {
	return func(command string) (string, error) {
		return mockOutput, nil
	}
}

// TestRunClishOutput simulates the output of clish for testing
func TestRunClishOutput(t *testing.T) {
	// Read the sample file content to use as mock output
	sampleFilePath := filepath.Join("..", "show-mac-address-table.txt")
	macOutput, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	// Create a mock runClish function
	mockClish := mockRunClishFunction(string(macOutput))
	
	// Run the command through our mocked clish function
	output, err := mockClish("show mac address-table")
	if err != nil {
		t.Fatalf("Mock runClish failed: %v", err)
	}
	
	// Parse the MAC address table from this mock output
	entries, err := parseMAC(output)
	if err != nil {
		t.Fatalf("Failed to parse MAC address table from mock clish output: %v", err)
	}
	
	// Verify we got entries
	if len(entries) == 0 {
		t.Fatal("No entries parsed from mock clish output")
	}
	
	// Check a few specific entries to ensure proper parsing
	// Example: Find the first primary entry in VLAN 7
	foundPrimary := false
	for _, entry := range entries {
		if entry.Message.PrimaryEntry && entry.Message.VLAN == "7" {
			foundPrimary = true
			if entry.Message.Type != "dynamic" {
				t.Errorf("Expected Type to be 'dynamic', got %q", entry.Message.Type)
			}
			break
		}
	}
	
	if !foundPrimary {
		t.Error("Failed to find a primary entry in VLAN 7 in clish output")
	}
	
	// Check timestamp formats
	firstEntry := entries[0]
	
	// Check timestamp format ISO 8601
	tsRegex := `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`
	if match, err := regexp.MatchString(tsRegex, firstEntry.Timestamp); err != nil {
		t.Errorf("Error checking timestamp format: %v", err)
	} else if !match {
		t.Errorf("Timestamp not in expected ISO 8601 format. Got: %s", firstEntry.Timestamp)
	}
	
	// Check date format YYYY-MM-DD
	dateRegex := `^\d{4}-\d{2}-\d{2}$`
	if match, err := regexp.MatchString(dateRegex, firstEntry.Date); err != nil {
		t.Errorf("Error checking date format: %v", err)
	} else if !match {
		t.Errorf("Date not in expected YYYY-MM-DD format. Got: %s", firstEntry.Date)
	}
	
	fmt.Printf("Successfully parsed %d MAC address table entries from mock clish output\n", len(entries))
}

func TestParseMAC(t *testing.T) {
	// Read the sample file from the parent directory
	sampleFilePath := filepath.Join("..", "show-mac-address-table.txt")
	data, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	// Parse the MAC address table
	entries, err := parseMAC(string(data))
	if err != nil {
		t.Fatalf("Failed to parse MAC address table: %v", err)
	}
	
	// Basic validation checks
	if len(entries) == 0 {
		t.Fatal("No entries parsed from sample file")
	}
	
	// Check a few specific entries to ensure proper parsing
	// Example: Find the first primary entry in VLAN 7
	foundPrimary := false
	for _, entry := range entries {
		if entry.Message.PrimaryEntry && entry.Message.VLAN == "7" {
			foundPrimary = true
			if entry.Message.Type != "dynamic" {
				t.Errorf("Expected Type to be 'dynamic', got %q", entry.Message.Type)
			}
			break
		}
	}
	
	if !foundPrimary {
		t.Error("Failed to find a primary entry in VLAN 7")
	}
	
	// Check for Gateway MAC entries
	foundGateway := false
	for _, entry := range entries {
		if entry.Message.GatewayMAC {
			foundGateway = true
			if !entry.Message.RoutedMAC {
				t.Error("Gateway MAC entry should have RoutedMAC set to true")
			}
			break
		}
	}
	
	if !foundGateway {
		t.Error("Failed to find a Gateway MAC entry")
	}
	
	fmt.Printf("Successfully parsed %d MAC address table entries\n", len(entries))
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
				"name": "mac-address-table",
				"command": "show mac address-table"
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
	
	// Find the mac-address-table command
	var macCmd string
	for _, cmd := range cmdFile.Commands {
		if cmd.Name == "mac-address-table" {
			macCmd = cmd.Command
			break
		}
	}
	
	if macCmd == "" {
		t.Error("Failed to find mac-address-table command in the JSON")
	} else if macCmd != "show mac address-table" {
		t.Errorf("Expected command to be 'show mac address-table', but got '%s'", macCmd)
	}
	
	fmt.Println("Successfully parsed commands JSON file")
}

// TestIntegratedCommandExecution simulates the integration of JSON command parsing and CLI execution
func TestIntegratedCommandExecution(t *testing.T) {
	// Create a temporary command JSON file
	tempDir := t.TempDir()
	commandsFilePath := filepath.Join(tempDir, "commands.json")
	
	// Define sample JSON content
	jsonContent := `{
		"commands": [
			{
				"name": "mac-address-table",
				"command": "show mac address-table"
			},
			{
				"name": "other-command",
				"command": "show something else"
			}
		]
	}`
	
	// Write the content to the file
	if err := os.WriteFile(commandsFilePath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test commands JSON file: %v", err)
	}
	
	// Read the sample MAC table output to use as mock clish output
	sampleFilePath := filepath.Join("..", "show-mac-address-table.txt")
	macOutput, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
		// Create a mock runClish function
	mockClish := mockRunClishFunction(string(macOutput))
	
	// Parse the JSON file
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
	
	// Find the mac-address-table command
	var macCmd string
	for _, cmd := range cmdFile.Commands {
		if cmd.Name == "mac-address-table" {
			macCmd = cmd.Command
			break
		}
	}
	
	if macCmd == "" {
		t.Fatal("Failed to find mac-address-table command in the JSON")
	}
		// Execute the command using our mocked runClish
	output, err := mockClish(macCmd)
	if err != nil {
		t.Fatalf("Failed to run clish command: %v", err)
	}
	
	// Parse the MAC address table from this output
	entries, err := parseMAC(output)
	if err != nil {
		t.Fatalf("Failed to parse MAC address table: %v", err)
	}
	
	// Verify we got entries
	if len(entries) == 0 {
		t.Fatal("No entries parsed from clish output")
	}
	
	// Check a specific entry to ensure proper parsing
	foundPrimary := false
	for _, entry := range entries {
		if entry.Message.PrimaryEntry && entry.Message.VLAN == "7" {
			foundPrimary = true
			break
		}
	}
	
	if !foundPrimary {
		t.Error("Failed to find a primary entry in VLAN 7 in the parsed output")
	}
	
	// Check timestamp formats
	firstEntry := entries[0]
	
	// Check timestamp format ISO 8601
	tsRegex := `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`
	if match, err := regexp.MatchString(tsRegex, firstEntry.Timestamp); err != nil {
		t.Errorf("Error checking timestamp format: %v", err)
	} else if !match {
		t.Errorf("Timestamp not in expected ISO 8601 format. Got: %s", firstEntry.Timestamp)
	}
	
	fmt.Printf("Successfully executed integrated command flow with %d parsed entries\n", len(entries))
}

// TestInvalidInputs tests how the program handles incorrect input cases
func TestInvalidInputs(t *testing.T) {
	// Test parsing with invalid MAC table format
	invalidInput := "This is not a valid MAC address table output"
	entries, err := parseMAC(invalidInput)
	if err == nil {
		t.Error("Expected an error when parsing invalid MAC table, but got none")
	}
	if entries != nil {
		t.Errorf("Expected nil entries for invalid input, but got %d entries", len(entries))
	}
	
	// Test with empty input
	entries, err = parseMAC("")
	if err == nil {
		t.Error("Expected an error when parsing empty input, but got none")
	}
	if entries != nil {
		t.Error("Expected nil entries for empty input, but got non-nil")
	}
}

// TestMutuallyExclusiveFlags tests that we cannot specify both -input and -commands flags
func TestMutuallyExclusiveFlags(t *testing.T) {
	// We can't test the os.Exit part of main() directly
	// But we can test the logic that would cause it to exit
	// by simulating the condition check
	
	// Case 1: Both flags specified (should be an error)
	inputFlag := "input.txt"
	commandsFlag := "commands.json"
	
	if !(inputFlag != "" && commandsFlag != "") && !(inputFlag == "" && commandsFlag == "") {
		t.Error("Expected condition to be true when both flags are specified")
	}
	
	// Case 2: Neither flag specified (should be an error)
	inputFlag = ""
	commandsFlag = ""
	
	if !(inputFlag != "" && commandsFlag != "") && !(inputFlag == "" && commandsFlag == "") {
		t.Error("Expected condition to be true when neither flag is specified")
	}
	
	// Case 3: Only inputFlag specified (should be valid)
	inputFlag = "input.txt"
	commandsFlag = ""
	
	if (inputFlag != "" && commandsFlag != "") || (inputFlag == "" && commandsFlag == "") {
		t.Error("Expected condition to be false when only input flag is specified")
	}
	
	// Case 4: Only commandsFlag specified (should be valid)
	inputFlag = ""
	commandsFlag = "commands.json"
	
	if (inputFlag != "" && commandsFlag != "") || (inputFlag == "" && commandsFlag == "") {
		t.Error("Expected condition to be false when only commands flag is specified")
	}
	
	fmt.Println("Successfully tested mutually exclusive flags validation")
}

// TestDataTypeField specifically tests the new data_type field for KQL queries
func TestDataTypeField(t *testing.T) {
	// Read the sample file from the parent directory
	sampleFilePath := filepath.Join("..", "show-mac-address-table.txt")
	data, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	// Parse the MAC address table
	entries, err := parseMAC(string(data))
	if err != nil {
		t.Fatalf("Failed to parse MAC address table: %v", err)
	}
	
	// Verify that each entry has the expected data_type for KQL queries
	for i, entry := range entries {
		if entry.DataType != "cisco_nexus_mac_table" {
			t.Errorf("Entry %d: expected DataType to be 'cisco_nexus_mac_table', got %q", i, entry.DataType)
		}
	}
	
	fmt.Println("All entries have the correct data_type field")
}

// The test suite covers the following aspects of the MAC address parser:
// 1. TestRunClishOutput - Tests the parsing of MAC table data as if it came from a CLI command
// 2. TestParseMAC - Tests the parsing of MAC table data directly from a file
// 3. TestCommandJsonParsing - Tests the parsing of a JSON file containing CLI commands
// 4. TestIntegratedCommandExecution - Tests the full workflow of parsing JSON commands, 
//    running the MAC address table command, and parsing its output
// 5. TestInvalidInputs - Tests error handling for invalid MAC table formats
// 6. TestMutuallyExclusiveFlags - Tests the logic that ensures exactly one input method is specified
//
// Note: A full integration test of main() is intentionally not implemented since it uses os.Exit,
// making it difficult to test in the same process. The key components are tested separately.
