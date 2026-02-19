package interface_status_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseInterfaceStatusFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_interface_status.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&InterfaceStatusParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 5 {
		t.Fatalf("Expected 5, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg InterfaceStatusData
	json.Unmarshal(b, &msg)

	if msg.Port != "Eth 1/1/1:1" {
		t.Errorf("Port: got '%s'", msg.Port)
	}
	if msg.Status != "up" {
		t.Errorf("Status: got '%s'", msg.Status)
	}
	if !msg.IsUp {
		t.Error("IsUp: expected true")
	}
	if msg.Speed != "10G" {
		t.Errorf("Speed: got '%s'", msg.Speed)
	}

	b3, _ := json.Marshal(entries[2].Message)
	var msg3 InterfaceStatusData
	json.Unmarshal(b3, &msg3)
	if msg3.Status != "down" {
		t.Errorf("Third Status: got '%s'", msg3.Status)
	}

	b4, _ := json.Marshal(entries[3].Message)
	var msg4 InterfaceStatusData
	json.Unmarshal(b4, &msg4)
	if msg4.Port != "Po 128" {
		t.Errorf("Fourth Port: got '%s'", msg4.Port)
	}
	if msg4.Speed != "200G" {
		t.Errorf("Fourth Speed: got '%s'", msg4.Speed)
	}
}

func TestParseInterfaceStatusEmpty(t *testing.T) {
	result, err := (&InterfaceStatusParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 0 {
		t.Errorf("Expected 0, got %d", len(entries))
	}
}
