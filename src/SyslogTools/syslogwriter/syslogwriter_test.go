package syslogwriter

import (
	"encoding/json"
	"strings"
	"testing"
)

// Test data
var validJSONEntry = `{"data_type":"interface_counters","timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15","message":{"interface":"eth0","counters":{"rx_packets":12345,"tx_packets":67890}}}`

// Test validation without requiring syslog
func TestValidateEntryStandalone(t *testing.T) {
	// Create a mock writer for validation testing
	writer := &Writer{
		config: &Config{
			Tag:          "test",
			MaxEntrySize: 4096,
			Verbose:      false,
		},
	}

	tests := []struct {
		name      string
		jsonEntry string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid entry",
			jsonEntry: validJSONEntry,
			wantErr:   false,
		},
		{
			name:      "invalid JSON",
			jsonEntry: `{"invalid json}`,
			wantErr:   true,
			errMsg:    "invalid JSON",
		},
		{
			name:      "missing data_type",
			jsonEntry: `{"timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15","message":{"test":"data"}}`,
			wantErr:   true,
			errMsg:    "missing required field: data_type",
		},
		{
			name:      "missing timestamp",
			jsonEntry: `{"data_type":"interface_counters","date":"2024-01-15","message":{"test":"data"}}`,
			wantErr:   true,
			errMsg:    "missing required field: timestamp",
		},
		{
			name:      "missing date",
			jsonEntry: `{"data_type":"interface_counters","timestamp":"2024-01-15T10:30:00Z","message":{"test":"data"}}`,
			wantErr:   true,
			errMsg:    "missing required field: date",
		},
		{
			name:      "missing message",
			jsonEntry: `{"data_type":"interface_counters","timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15"}`,
			wantErr:   true,
			errMsg:    "missing required field: message",
		},
		{
			name:      "empty data_type",
			jsonEntry: `{"data_type":"","timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15","message":{"test":"data"}}`,
			wantErr:   true,
			errMsg:    "missing required field: data_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writer.validateEntry(tt.jsonEntry)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateEntry() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestCommonFields(t *testing.T) {
	// Test that CommonFields struct works correctly with JSON
	cf := CommonFields{
		DataType:  "interface_counters",
		Timestamp: "2024-01-15T10:30:00Z",
		Date:      "2024-01-15",
		Message:   map[string]interface{}{"interface": "eth0", "status": "up"},
	}

	jsonData, err := json.Marshal(cf)
	if err != nil {
		t.Fatalf("Failed to marshal CommonFields: %v", err)
	}

	var cf2 CommonFields
	err = json.Unmarshal(jsonData, &cf2)
	if err != nil {
		t.Fatalf("Failed to unmarshal CommonFields: %v", err)
	}

	if cf.DataType != cf2.DataType || cf.Timestamp != cf2.Timestamp || cf.Date != cf2.Date {
		t.Errorf("CommonFields round-trip failed for basic fields: %+v != %+v", cf, cf2)
	}
	
	// Check message field separately since maps can't be directly compared
	if cf2.Message == nil {
		t.Error("Message field should not be nil after round-trip")
	}
}

func TestDetectSystemLogger(t *testing.T) {
	// This is a simple smoke test - actual behavior depends on system
	result := DetectSystemLogger()
	
	// Just ensure it returns a boolean without panicking
	if result != true && result != false {
		t.Error("DetectSystemLogger() should return a boolean")
	}
}

// Integration test that requires syslog (will be skipped if syslog is not available)
func TestIntegrationWithSyslog(t *testing.T) {
	writer, err := NewWithDefaults("test-tag")
	if err != nil && strings.Contains(err.Error(), "Unix syslog delivery error") {
		t.Skipf("Skipping integration test - syslog not available: %v", err)
		return
	}
	if err != nil {
		t.Fatalf("NewWithDefaults() unexpected error = %v", err)
	}
	defer writer.Close()

	// Test basic configuration
	if writer.config.Tag != "test-tag" {
		t.Errorf("Config tag = %v, want %v", writer.config.Tag, "test-tag")
	}

	// Test writing a valid entry
	err = writer.WriteEntry(validJSONEntry)
	if err != nil {
		t.Errorf("WriteEntry() failed: %v", err)
	}
}

func TestMessageFieldValidation(t *testing.T) {
	writer := &Writer{
		config: &Config{
			Tag:          "test",
			MaxEntrySize: 4096,
			Verbose:      false,
		},
	}

	tests := []struct {
		name      string
		jsonEntry string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "message with object",
			jsonEntry: `{"data_type":"interface_counters","timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15","message":{"interface":"eth0","counters":{"rx":12345,"tx":67890}}}`,
			wantErr:   false,
		},
		{
			name:      "message with array",
			jsonEntry: `{"data_type":"interface_list","timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15","message":["eth0","eth1","eth2"]}`,
			wantErr:   false,
		},
		{
			name:      "message with string",
			jsonEntry: `{"data_type":"system_log","timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15","message":"System startup completed"}`,
			wantErr:   false,
		},
		{
			name:      "message with number",
			jsonEntry: `{"data_type":"metric","timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15","message":42}`,
			wantErr:   false,
		},
		{
			name:      "message with null",
			jsonEntry: `{"data_type":"test","timestamp":"2024-01-15T10:30:00Z","date":"2024-01-15","message":null}`,
			wantErr:   true,
			errMsg:    "missing required field: message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writer.validateEntry(tt.jsonEntry)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateEntry() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}
