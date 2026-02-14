package version_parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	// Read the sample input file
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	// Parse the version info
	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Validate data type
	if entry.DataType != "dell_os10_version" {
		t.Errorf("Expected data_type 'dell_os10_version', got %s", entry.DataType)
	}

	// Validate timestamp format
	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}

	// Validate date format
	if entry.Date == "" {
		t.Error("Date should not be empty")
	}
}

func TestOSName(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.OSName != "Dell SmartFabric OS10 Enterprise" {
		t.Errorf("Expected OSName 'Dell SmartFabric OS10 Enterprise', got '%s'", data.OSName)
	}
}

func TestOSVersion(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.OSVersion != "10.6.0.5" {
		t.Errorf("Expected OSVersion '10.6.0.5', got '%s'", data.OSVersion)
	}
}

func TestBuildVersion(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.BuildVersion != "10.6.0.5.139" {
		t.Errorf("Expected BuildVersion '10.6.0.5.139', got '%s'", data.BuildVersion)
	}
}

func TestBuildTime(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.BuildTime != "2025-07-02T19:13:52+0000" {
		t.Errorf("Expected BuildTime '2025-07-02T19:13:52+0000', got '%s'", data.BuildTime)
	}
}

func TestSystemType(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.SystemType != "S5248F-ON" {
		t.Errorf("Expected SystemType 'S5248F-ON', got '%s'", data.SystemType)
	}
}

func TestArchitecture(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.Architecture != "x86_64" {
		t.Errorf("Expected Architecture 'x86_64', got '%s'", data.Architecture)
	}
}

func TestUpTime(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.UpTime != "6 weeks 1 day 17:55:03" {
		t.Errorf("Expected UpTime '6 weeks 1 day 17:55:03', got '%s'", data.UpTime)
	}
}

func TestDeviceName(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.DeviceName != "rr1-s46-r06-5248hl-6-1a" {
		t.Errorf("Expected DeviceName 'rr1-s46-r06-5248hl-6-1a', got '%s'", data.DeviceName)
	}
}

func TestKernelUptime(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	uptime := entries[0].Message.KernelUptime

	if uptime.Weeks != 6 {
		t.Errorf("Expected Uptime Weeks 6, got %d", uptime.Weeks)
	}

	if uptime.Days != 1 {
		t.Errorf("Expected Uptime Days 1, got %d", uptime.Days)
	}

	if uptime.Hours != 17 {
		t.Errorf("Expected Uptime Hours 17, got %d", uptime.Hours)
	}

	if uptime.Minutes != 55 {
		t.Errorf("Expected Uptime Minutes 55, got %d", uptime.Minutes)
	}

	if uptime.Seconds != 3 {
		t.Errorf("Expected Uptime Seconds 3, got %d", uptime.Seconds)
	}
}

func TestJSONSerialization(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
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
	if entry.DataType != "dell_os10_version" {
		t.Errorf("Round-trip failed: data_type mismatch")
	}

	if entry.Message.OSName != "Dell SmartFabric OS10 Enterprise" {
		t.Errorf("Round-trip failed: OSName mismatch")
	}

	if entry.Message.OSVersion != "10.6.0.5" {
		t.Errorf("Round-trip failed: OSVersion mismatch")
	}

	if entry.Message.SystemType != "S5248F-ON" {
		t.Errorf("Round-trip failed: SystemType mismatch")
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
	inputData, err := os.ReadFile("show-version.txt")
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

func TestJSONOutputStructure(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	// Serialize to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(entries[0], "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Verify the JSON structure contains expected keys
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Check top-level keys
	expectedKeys := []string{"data_type", "timestamp", "date", "message"}
	for _, key := range expectedKeys {
		if _, ok := jsonMap[key]; !ok {
			t.Errorf("Expected key '%s' not found in JSON output", key)
		}
	}

	// Check message structure
	message, ok := jsonMap["message"].(map[string]interface{})
	if !ok {
		t.Fatal("Message should be a map")
	}

	expectedMessageKeys := []string{
		"nxos_version", "bios_version", "nxos_compile_time", "bios_compile_time",
		"chassis_id", "cpu_name", "boot_mode", "device_name", "kernel_uptime",
	}
	for _, key := range expectedMessageKeys {
		if _, ok := message[key]; !ok {
			t.Errorf("Expected message key '%s' not found in JSON output", key)
		}
	}
}

func TestKernelUptimeStructure(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(entries[0], "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Parse back and check kernel_uptime structure
	var jsonMap map[string]interface{}
	json.Unmarshal(jsonData, &jsonMap)

	message := jsonMap["message"].(map[string]interface{})
	uptime := message["kernel_uptime"].(map[string]interface{})

	expectedUptimeKeys := []string{"weeks", "days", "hours", "minutes", "seconds"}
	for _, key := range expectedUptimeKeys {
		if _, ok := uptime[key]; !ok {
			t.Errorf("Expected kernel_uptime key '%s' not found", key)
		}
	}
}

func TestUptimeVariations(t *testing.T) {
	tests := []struct {
		name           string
		uptimeStr      string
		expectedWeeks  int
		expectedDays   int
		expectedHours  int
		expectedMinutes int
		expectedSeconds int
	}{
		{
			name:           "Weeks, days and time",
			uptimeStr:      "6 weeks 1 day 17:55:03",
			expectedWeeks:  6,
			expectedDays:   1,
			expectedHours:  17,
			expectedMinutes: 55,
			expectedSeconds: 3,
		},
		{
			name:           "Days and time only",
			uptimeStr:      "48 days 21:15:01",
			expectedWeeks:  0,
			expectedDays:   48,
			expectedHours:  21,
			expectedMinutes: 15,
			expectedSeconds: 1,
		},
		{
			name:           "Time only",
			uptimeStr:      "05:30:45",
			expectedWeeks:  0,
			expectedDays:   0,
			expectedHours:  5,
			expectedMinutes: 30,
			expectedSeconds: 45,
		},
		{
			name:           "Single week",
			uptimeStr:      "1 week 0 days 00:00:00",
			expectedWeeks:  1,
			expectedDays:   0,
			expectedHours:  0,
			expectedMinutes: 0,
			expectedSeconds: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf("rr1-test# show version\nUp Time: %s\n", tt.uptimeStr)
			entries, err := parseVersion(input)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			uptime := entries[0].Message.KernelUptime
			if uptime.Weeks != tt.expectedWeeks {
				t.Errorf("Expected Weeks %d, got %d", tt.expectedWeeks, uptime.Weeks)
			}
			if uptime.Days != tt.expectedDays {
				t.Errorf("Expected Days %d, got %d", tt.expectedDays, uptime.Days)
			}
			if uptime.Hours != tt.expectedHours {
				t.Errorf("Expected Hours %d, got %d", tt.expectedHours, uptime.Hours)
			}
			if uptime.Minutes != tt.expectedMinutes {
				t.Errorf("Expected Minutes %d, got %d", tt.expectedMinutes, uptime.Minutes)
			}
			if uptime.Seconds != tt.expectedSeconds {
				t.Errorf("Expected Seconds %d, got %d", tt.expectedSeconds, uptime.Seconds)
			}
		})
	}
}

func TestJSONKeyAlignment(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(entries[0])
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Verify that JSON uses Cisco-aligned key names
	jsonStr := string(jsonData)

	// Check for Cisco-aligned keys in the JSON output
	expectedKeys := []string{
		"\"nxos_version\"",      // Maps to OSName
		"\"bios_version\"",       // Maps to OSVersion
		"\"nxos_compile_time\"",  // Maps to BuildVersion
		"\"bios_compile_time\"",  // Maps to BuildTime
		"\"chassis_id\"",         // Maps to SystemType
		"\"cpu_name\"",           // Maps to Architecture
		"\"boot_mode\"",          // Maps to UpTime string
		"\"device_name\"",        // Maps to DeviceName
		"\"kernel_uptime\"",      // Maps to parsed uptime
	}

	for _, key := range expectedKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("Expected JSON to contain key %s", key)
		}
	}
}

func TestEmptyInput(t *testing.T) {
	entries, err := parseVersion("")
	if err != nil {
		t.Fatalf("Empty input should not cause error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry for empty input, got %d", len(entries))
	}
}

func TestPartialInput(t *testing.T) {
	input := `Dell SmartFabric OS10 Enterprise
OS Version: 10.6.0.5`
	
	entries, err := parseVersion(input)
	if err != nil {
		t.Fatalf("Partial input should not cause error: %v", err)
	}
	
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
	
	data := entries[0].Message
	if data.OSName != "Dell SmartFabric OS10 Enterprise" {
		t.Errorf("Expected OSName to be parsed correctly, got '%s'", data.OSName)
	}
	if data.OSVersion != "10.6.0.5" {
		t.Errorf("Expected OSVersion to be parsed correctly, got '%s'", data.OSVersion)
	}
}

func TestInvalidUptimeFormat(t *testing.T) {
	input := `rr1-test# show version
Up Time: invalid format`
	
	entries, err := parseVersion(input)
	if err != nil {
		t.Fatalf("Invalid uptime format should not cause fatal error: %v", err)
	}
	
	data := entries[0].Message
	if data.UpTime != "invalid format" {
		t.Errorf("Expected UpTime string to be stored as-is, got '%s'", data.UpTime)
	}
	// Kernel uptime should be zero values when parsing fails
	if data.KernelUptime.Weeks != 0 || data.KernelUptime.Days != 0 {
		t.Errorf("Expected zero uptime values for invalid format")
	}
}
