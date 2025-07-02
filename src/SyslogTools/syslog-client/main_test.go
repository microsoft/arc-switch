package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// TestValidateEntry tests the validation of JSON entries
func TestValidateEntry(t *testing.T) {
	tests := []struct {
		name        string
		entry       string
		expectError bool
	}{
		{
			name:        "valid entry with all required fields",
			entry:       `{"data_type":"test","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025","extra":"data"}`,
			expectError: false,
		},
		{
			name:        "missing data_type",
			entry:       `{"timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025"}`,
			expectError: true,
		},
		{
			name:        "missing timestamp",
			entry:       `{"data_type":"test","date":"06/23/2025"}`,
			expectError: true,
		},
		{
			name:        "missing date",
			entry:       `{"data_type":"test","timestamp":"06/23/2025 05:05:01 PM"}`,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			entry:       `{"data_type":"test","timestamp":"06/23/2025 05:05:01 PM"`,
			expectError: true,
		},
		{
			name:        "empty data_type",
			entry:       `{"data_type":"","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025"}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSingleEntry(tt.entry)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestCommonFieldsParsing tests parsing of common fields
func TestCommonFieldsParsing(t *testing.T) {
	entry := `{"data_type":"cisco_nexus_mac_table","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025","vlan":"7","mac_address":"02ec.a004.0000"}`
	
	var fields CommonFields
	err := json.Unmarshal([]byte(entry), &fields)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	if fields.DataType != "cisco_nexus_mac_table" {
		t.Errorf("Expected data_type 'cisco_nexus_mac_table', got '%s'", fields.DataType)
	}
	
	if fields.Timestamp != "06/23/2025 05:05:01 PM" {
		t.Errorf("Expected timestamp '06/23/2025 05:05:01 PM', got '%s'", fields.Timestamp)
	}
	
	if fields.Date != "06/23/2025" {
		t.Errorf("Expected date '06/23/2025', got '%s'", fields.Date)
	}
}

// TestHandleEntrySize tests entry size handling and truncation
func TestHandleEntrySize(t *testing.T) {
	config := &SyslogConfig{
		MaxEntrySize: 100,
	}
	
	writer := &SyslogWriter{
		config: config,
		stats: &Statistics{
			StartTime: time.Now(),
		},
	}
	
	// Test entry within size limit
	shortEntry := `{"data_type":"test","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025"}`
	result := writer.handleEntrySize(shortEntry, false)
	if result != shortEntry {
		t.Errorf("Short entry should not be truncated")
	}
	
	// Test entry exceeding size limit
	longEntry := strings.Repeat("a", 150)
	result = writer.handleEntrySize(longEntry, false)
	if len(result) != 100 {
		t.Errorf("Expected truncated length 100, got %d", len(result))
	}
	
	if !strings.HasSuffix(result, "...") {
		t.Errorf("Truncated entry should end with '...'")
	}
	
	if writer.stats.TruncatedEntries != 1 {
		t.Errorf("Expected 1 truncated entry, got %d", writer.stats.TruncatedEntries)
	}
}

// TestStatistics tests statistics tracking
func TestStatistics(t *testing.T) {
	writer := &SyslogWriter{
		config: &SyslogConfig{MaxEntrySize: 4096},
		stats: &Statistics{
			StartTime: time.Now(),
		},
	}
	
	// Test initial statistics
	stats := writer.GetStatistics()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 total entries, got %d", stats.TotalEntries)
	}
	
	// Simulate processing
	writer.stats.TotalEntries = 10
	writer.stats.SuccessEntries = 8
	writer.stats.FailedEntries = 2
	
	stats = writer.GetStatistics()
	if stats.TotalEntries != 10 {
		t.Errorf("Expected 10 total entries, got %d", stats.TotalEntries)
	}
	if stats.SuccessEntries != 8 {
		t.Errorf("Expected 8 success entries, got %d", stats.SuccessEntries)
	}
	if stats.FailedEntries != 2 {
		t.Errorf("Expected 2 failed entries, got %d", stats.FailedEntries)
	}
}

// TestValidateFile tests file validation functionality
func TestValidateFile(t *testing.T) {
	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	// Write test data
	testData := `{"data_type":"test1","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025","value":"1"}
{"data_type":"test2","timestamp":"06/23/2025 05:05:02 PM","date":"06/23/2025","value":"2"}

{"data_type":"test3","timestamp":"06/23/2025 05:05:03 PM","date":"06/23/2025","value":"3"}
`
	
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tmpFile.Close()
	
	// Test validation
	err = validateFile(tmpFile.Name(), false)
	if err != nil {
		t.Errorf("Expected successful validation, got error: %v", err)
	}
}

// TestInvalidFile tests validation of file with invalid entries
func TestInvalidFile(t *testing.T) {
	// Create a temporary test file with invalid data
	tmpFile, err := os.CreateTemp("", "invalid_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	// Write test data with invalid entries
	testData := `{"data_type":"test1","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025"}
{"timestamp":"06/23/2025 05:05:02 PM","date":"06/23/2025"}
{"data_type":"test3","timestamp":"06/23/2025 05:05:03 PM","date":"06/23/2025"}
`
	
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tmpFile.Close()
	
	// Test validation - should fail
	err = validateFile(tmpFile.Name(), false)
	if err == nil {
		t.Errorf("Expected validation to fail, but it succeeded")
	}
}

// TestCiscoMACTableEntry tests validation of Cisco MAC table entries
func TestCiscoMACTableEntry(t *testing.T) {
	// Test entry from the sample file
	entry := `{"data_type":"cisco_nexus_mac_table","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025","primary_entry":true,"gateway_mac":false,"routed_mac":false,"overlay_mac":false,"vlan":"7","mac_address":"02ec.a004.0000","type":"dynamic","age":"NA","secure":"F","ntfy":"F","port":"Eth1/1"}`
	
	err := validateSingleEntry(entry)
	if err != nil {
		t.Errorf("Cisco MAC table entry should be valid, got error: %v", err)
	}
	
	// Parse and verify fields
	var fields map[string]interface{}
	err = json.Unmarshal([]byte(entry), &fields)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	
	// Check required fields
	if fields["data_type"] != "cisco_nexus_mac_table" {
		t.Errorf("Expected data_type 'cisco_nexus_mac_table'")
	}
	
	// Check additional fields are preserved
	if fields["vlan"] != "7" {
		t.Errorf("Expected vlan '7', got %v", fields["vlan"])
	}
	
	if fields["mac_address"] != "02ec.a004.0000" {
		t.Errorf("Expected mac_address '02ec.a004.0000', got %v", fields["mac_address"])
	}
}

// BenchmarkValidateEntry benchmarks the entry validation function
func BenchmarkValidateEntry(b *testing.B) {
	entry := `{"data_type":"test","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025","extra":"data"}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateSingleEntry(entry)
	}
}

// BenchmarkHandleEntrySize benchmarks the entry size handling function
func BenchmarkHandleEntrySize(b *testing.B) {
	config := &SyslogConfig{
		MaxEntrySize: 4096,
	}
	
	writer := &SyslogWriter{
		config: config,
		stats: &Statistics{
			StartTime: time.Now(),
		},
	}
	
	entry := strings.Repeat("a", 3000) // Under the limit
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = writer.handleEntrySize(entry, false)
	}
}
