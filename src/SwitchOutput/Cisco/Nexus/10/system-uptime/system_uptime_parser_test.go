package system_uptime_parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestParseSystemUptimeText tests parsing of text format output
func TestParseSystemUptimeText(t *testing.T) {
	// Read the sample file from the parent directory
	sampleFilePath := filepath.Join("..", "show-system-uptime.txt")
	data, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	// Parse the system uptime
	entry, err := parseSystemUptime(string(data))
	if err != nil {
		t.Fatalf("Failed to parse system uptime: %v", err)
	}
	
	// Verify basic structure
	if entry.DataType != "cisco_nexus_system_uptime" {
		t.Errorf("Expected DataType to be 'cisco_nexus_system_uptime', got %q", entry.DataType)
	}
	
	// Check timestamp format ISO 8601
	tsRegex := `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`
	if match, err := regexp.MatchString(tsRegex, entry.Timestamp); err != nil {
		t.Errorf("Error checking timestamp format: %v", err)
	} else if !match {
		t.Errorf("Timestamp not in expected ISO 8601 format. Got: %s", entry.Timestamp)
	}
	
	// Check date format YYYY-MM-DD
	dateRegex := `^\d{4}-\d{2}-\d{2}$`
	if match, err := regexp.MatchString(dateRegex, entry.Date); err != nil {
		t.Errorf("Error checking date format: %v", err)
	} else if !match {
		t.Errorf("Date not in expected YYYY-MM-DD format. Got: %s", entry.Date)
	}
	
	// Verify system uptime data
	if entry.Message.SystemStartTime == "" {
		t.Error("SystemStartTime should not be empty")
	}
	
	if entry.Message.SystemUptimeDays == "" {
		t.Error("SystemUptimeDays should not be empty")
	}
	
	if entry.Message.SystemUptimeHours == "" {
		t.Error("SystemUptimeHours should not be empty")
	}
	
	if entry.Message.KernelUptimeDays == "" {
		t.Error("KernelUptimeDays should not be empty")
	}
	
	// Check specific values from the sample file
	if entry.Message.SystemStartTime != "Fri Oct 17 13:50:28 2025" {
		t.Errorf("Expected SystemStartTime 'Fri Oct 17 13:50:28 2025', got %q", entry.Message.SystemStartTime)
	}
	
	if entry.Message.SystemUptimeDays != "2" {
		t.Errorf("Expected SystemUptimeDays '2', got %q", entry.Message.SystemUptimeDays)
	}
	
	if entry.Message.SystemUptimeHours != "22" {
		t.Errorf("Expected SystemUptimeHours '22', got %q", entry.Message.SystemUptimeHours)
	}
	
	if entry.Message.KernelUptimeDays != "2" {
		t.Errorf("Expected KernelUptimeDays '2', got %q", entry.Message.KernelUptimeDays)
	}
	
	fmt.Println("Successfully parsed system uptime in text format")
}

// TestParseSystemUptimeJSON tests parsing of JSON format output
func TestParseSystemUptimeJSON(t *testing.T) {
	jsonInput := `{
		"sys_st_time": "Fri Oct 17 13:50:28 2025",
		"sys_up_days": "2",
		"sys_up_hrs": "22",
		"sys_up_mins": "1",
		"sys_up_secs": "51",
		"kn_up_days": "2",
		"kn_up_hrs": "22",
		"kn_up_mins": "3",
		"kn_up_secs": "58"
	}`
	
	entry, err := parseSystemUptime(jsonInput)
	if err != nil {
		t.Fatalf("Failed to parse JSON system uptime: %v", err)
	}
	
	// Verify basic structure
	if entry.DataType != "cisco_nexus_system_uptime" {
		t.Errorf("Expected DataType to be 'cisco_nexus_system_uptime', got %q", entry.DataType)
	}
	
	// Verify parsed values
	if entry.Message.SystemStartTime != "Fri Oct 17 13:50:28 2025" {
		t.Errorf("Expected SystemStartTime 'Fri Oct 17 13:50:28 2025', got %q", entry.Message.SystemStartTime)
	}
	
	if entry.Message.SystemUptimeDays != "2" {
		t.Errorf("Expected SystemUptimeDays '2', got %q", entry.Message.SystemUptimeDays)
	}
	
	if entry.Message.SystemUptimeHours != "22" {
		t.Errorf("Expected SystemUptimeHours '22', got %q", entry.Message.SystemUptimeHours)
	}
	
	if entry.Message.SystemUptimeMinutes != "1" {
		t.Errorf("Expected SystemUptimeMinutes '1', got %q", entry.Message.SystemUptimeMinutes)
	}
	
	if entry.Message.SystemUptimeSeconds != "51" {
		t.Errorf("Expected SystemUptimeSeconds '51', got %q", entry.Message.SystemUptimeSeconds)
	}
	
	if entry.Message.KernelUptimeMinutes != "3" {
		t.Errorf("Expected KernelUptimeMinutes '3', got %q", entry.Message.KernelUptimeMinutes)
	}
	
	if entry.Message.KernelUptimeSeconds != "58" {
		t.Errorf("Expected KernelUptimeSeconds '58', got %q", entry.Message.KernelUptimeSeconds)
	}
	
	fmt.Println("Successfully parsed system uptime in JSON format")
}

// TestDataTypeField specifically tests the data_type field for KQL queries
func TestDataTypeField(t *testing.T) {
	// Read the sample file
	sampleFilePath := filepath.Join("..", "show-system-uptime.txt")
	data, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	// Parse the system uptime
	entry, err := parseSystemUptime(string(data))
	if err != nil {
		t.Fatalf("Failed to parse system uptime: %v", err)
	}
	
	// Verify that entry has the expected data_type for KQL queries
	if entry.DataType != "cisco_nexus_system_uptime" {
		t.Errorf("Expected DataType to be 'cisco_nexus_system_uptime', got %q", entry.DataType)
	}
	
	fmt.Println("Entry has the correct data_type field")
}

// TestInvalidInputs tests how the parser handles incorrect input cases
func TestInvalidInputs(t *testing.T) {
	// Test parsing with invalid format
	invalidInput := "This is not a valid system uptime output"
	entry, err := parseSystemUptime(invalidInput)
	if err == nil {
		t.Error("Expected an error when parsing invalid uptime data, but got none")
	}
	if entry != nil {
		t.Error("Expected nil entry for invalid input, but got non-nil")
	}
	
	// Test with empty input
	entry, err = parseSystemUptime("")
	if err == nil {
		t.Error("Expected an error when parsing empty input, but got none")
	}
	if entry != nil {
		t.Error("Expected nil entry for empty input, but got non-nil")
	}
	
	fmt.Println("Invalid input tests passed")
}

// TestJSONSerialization tests that the output can be properly serialized to JSON
func TestJSONSerialization(t *testing.T) {
	// Read the sample file
	sampleFilePath := filepath.Join("..", "show-system-uptime.txt")
	data, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	// Parse the system uptime
	entry, err := parseSystemUptime(string(data))
	if err != nil {
		t.Fatalf("Failed to parse system uptime: %v", err)
	}
	
	// Serialize to JSON
	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}
	
	// Verify we can deserialize back
	var deserializedEntry StandardizedEntry
	if err := json.Unmarshal(jsonData, &deserializedEntry); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	
	// Compare values
	if deserializedEntry.DataType != entry.DataType {
		t.Errorf("DataType mismatch after deserialization")
	}
	
	if deserializedEntry.Message.SystemStartTime != entry.Message.SystemStartTime {
		t.Errorf("SystemStartTime mismatch after deserialization")
	}
	
	fmt.Println("JSON serialization/deserialization successful")
}

// TestUnifiedParserInterface tests the UnifiedParser interface
func TestUnifiedParserInterface(t *testing.T) {
	parser := &UnifiedParser{}
	
	// Test GetDescription
	desc := parser.GetDescription()
	if desc == "" {
		t.Error("GetDescription should return a non-empty string")
	}
	
	// Test Parse method
	sampleFilePath := filepath.Join("..", "show-system-uptime.txt")
	data, err := os.ReadFile(sampleFilePath)
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}
	
	result, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("UnifiedParser.Parse failed: %v", err)
	}
	
	// Verify result is of correct type
	entry, ok := result.(*StandardizedEntry)
	if !ok {
		t.Fatalf("Parse result is not *StandardizedEntry")
	}
	
	if entry.DataType != "cisco_nexus_system_uptime" {
		t.Errorf("Expected DataType 'cisco_nexus_system_uptime', got %q", entry.DataType)
	}
	
	fmt.Println("UnifiedParser interface tests passed")
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
				"name": "system-uptime",
				"command": "show system uptime"
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
	
	// Find the system-uptime command
	var uptimeCmd string
	for _, cmd := range cmdFile.Commands {
		if cmd.Name == "system-uptime" {
			uptimeCmd = cmd.Command
			break
		}
	}
	
	if uptimeCmd == "" {
		t.Error("Failed to find system-uptime command in the JSON")
	} else if uptimeCmd != "show system uptime" {
		t.Errorf("Expected command to be 'show system uptime', but got '%s'", uptimeCmd)
	}
	
	fmt.Println("Successfully parsed commands JSON file")
}
