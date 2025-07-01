package syslogwriter

import (
	"encoding/json"
	"log/syslog"
	"strings"
	"testing"
	"time"
)

// Test data
var validJSONEntry = `{"data_type":"test_data","timestamp":"2024-01-01T12:00:00Z","date":"2024-01-01","message":"test message"}`

// Test validation without requiring syslog
func TestValidateEntryStandalone(t *testing.T) {
	// Create a mock writer for validation testing
	writer := &Writer{
		config: &Config{
			Priority:     syslog.LOG_LOCAL0 | syslog.LOG_INFO,
			Tag:          "test",
			MaxEntrySize: 4096,
		},
		stats: &Statistics{
			StartTime: time.Now(),
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
			jsonEntry: `{"timestamp":"2024-01-01T12:00:00Z","date":"2024-01-01"}`,
			wantErr:   true,
			errMsg:    "missing required field: data_type",
		},
		{
			name:      "missing timestamp",
			jsonEntry: `{"data_type":"test","date":"2024-01-01"}`,
			wantErr:   true,
			errMsg:    "missing required field: timestamp",
		},
		{
			name:      "missing date",
			jsonEntry: `{"data_type":"test","timestamp":"2024-01-01T12:00:00Z"}`,
			wantErr:   true,
			errMsg:    "missing required field: date",
		},
		{
			name:      "empty data_type",
			jsonEntry: `{"data_type":"","timestamp":"2024-01-01T12:00:00Z","date":"2024-01-01"}`,
			wantErr:   true,
			errMsg:    "missing required field: data_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writer.ValidateEntry(tt.jsonEntry)
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

func TestParseSyslogPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		expected syslog.Priority
		wantErr  bool
	}{
		{"emergency", "emerg", syslog.LOG_EMERG, false},
		{"alert", "alert", syslog.LOG_ALERT, false},
		{"critical", "crit", syslog.LOG_CRIT, false},
		{"error", "err", syslog.LOG_ERR, false},
		{"warning", "warning", syslog.LOG_WARNING, false},
		{"notice", "notice", syslog.LOG_NOTICE, false},
		{"info", "info", syslog.LOG_INFO, false},
		{"debug", "debug", syslog.LOG_DEBUG, false},
		{"invalid", "invalid", 0, true},
		{"case insensitive", "INFO", syslog.LOG_INFO, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSyslogPriority(tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSyslogPriority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("ParseSyslogPriority() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseSyslogFacility(t *testing.T) {
	tests := []struct {
		name     string
		facility string
		expected syslog.Priority
		wantErr  bool
	}{
		{"local0", "local0", syslog.LOG_LOCAL0, false},
		{"local1", "local1", syslog.LOG_LOCAL1, false},
		{"user", "user", syslog.LOG_USER, false},
		{"mail", "mail", syslog.LOG_MAIL, false},
		{"daemon", "daemon", syslog.LOG_DAEMON, false},
		{"invalid", "invalid", 0, true},
		{"case insensitive", "LOCAL0", syslog.LOG_LOCAL0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSyslogFacility(tt.facility)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSyslogFacility() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("ParseSyslogFacility() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCommonFields(t *testing.T) {
	// Test that CommonFields struct works correctly with JSON
	cf := CommonFields{
		DataType:  "test_type",
		Timestamp: "2024-01-01T12:00:00Z",
		Date:      "2024-01-01",
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

	if cf != cf2 {
		t.Errorf("CommonFields round-trip failed: %+v != %+v", cf, cf2)
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

	// Check statistics
	stats := writer.GetStatistics()
	if stats.TotalEntries != 1 {
		t.Errorf("TotalEntries = %v, want 1", stats.TotalEntries)
	}
	if stats.SuccessEntries != 1 {
		t.Errorf("SuccessEntries = %v, want 1", stats.SuccessEntries)
	}
}
