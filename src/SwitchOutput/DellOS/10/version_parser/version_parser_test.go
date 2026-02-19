package version_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseVersionFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_version.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&VersionParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg VersionData
	json.Unmarshal(b, &msg)

	if msg.OSName != "Dell SmartFabric OS10 Enterprise" {
		t.Errorf("OSName: got '%s'", msg.OSName)
	}
	if msg.OSVersion != "10.6.0.5" {
		t.Errorf("OSVersion: got '%s'", msg.OSVersion)
	}
	if msg.BuildVersion != "10.6.0.5.139" {
		t.Errorf("BuildVersion: got '%s'", msg.BuildVersion)
	}
	if msg.SystemType != "S5248F-ON" {
		t.Errorf("SystemType: got '%s'", msg.SystemType)
	}
	if msg.Architecture != "x86_64" {
		t.Errorf("Architecture: got '%s'", msg.Architecture)
	}
	if entries[0].DataType != "dell_os10_version" {
		t.Errorf("DataType: got '%s'", entries[0].DataType)
	}
}

func TestParseVersionEmpty(t *testing.T) {
	result, err := (&VersionParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
}

func TestVersionParserDescription(t *testing.T) {
	p := &VersionParser{}
	if p.GetDescription() == "" {
		t.Error("GetDescription should not be empty")
	}
}
