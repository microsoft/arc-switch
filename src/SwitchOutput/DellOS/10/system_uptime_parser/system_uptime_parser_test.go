package system_uptime_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseUptimeFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_uptime.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&UptimeParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg UptimeData
	json.Unmarshal(b, &msg)

	if msg.Weeks != 9 {
		t.Errorf("Weeks: got %d", msg.Weeks)
	}
	if msg.Days != 3 {
		t.Errorf("Days: got %d", msg.Days)
	}
	if msg.Hours != 1 {
		t.Errorf("Hours: got %d", msg.Hours)
	}
	if msg.Minutes != 24 {
		t.Errorf("Minutes: got %d", msg.Minutes)
	}
	if msg.Seconds != 11 {
		t.Errorf("Seconds: got %d", msg.Seconds)
	}
	expected := int64(9*7*24*3600 + 3*24*3600 + 1*3600 + 24*60 + 11)
	if msg.TotalSeconds != expected {
		t.Errorf("TotalSeconds: expected %d, got %d", expected, msg.TotalSeconds)
	}
}

func TestParseUptimeSingular(t *testing.T) {
	result, _ := (&UptimeParser{}).Parse([]byte("1 week 1 day 00:00:01"))
	entries := result.([]StandardizedEntry)
	b, _ := json.Marshal(entries[0].Message)
	var msg UptimeData
	json.Unmarshal(b, &msg)
	if msg.Weeks != 1 || msg.Days != 1 {
		t.Errorf("Expected 1w 1d, got %dw %dd", msg.Weeks, msg.Days)
	}
}

func TestParseUptimeEmpty(t *testing.T) {
	result, err := (&UptimeParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
}
