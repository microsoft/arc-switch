package system_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseSystemFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_system.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&SystemParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg SystemData
	json.Unmarshal(b, &msg)

	if msg.NodeID != 1 {
		t.Errorf("NodeID: got %d", msg.NodeID)
	}
	if msg.MAC != "c4:5a:b1:36:bb:85" {
		t.Errorf("MAC: got '%s'", msg.MAC)
	}
	if msg.UpTime != "9 weeks 3 days 01:30:55" {
		t.Errorf("UpTime: got '%s'", msg.UpTime)
	}
	if len(msg.Units) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(msg.Units))
	}
	if msg.Units[0].CurrentType != "S5248F" {
		t.Errorf("CurrentType: got '%s'", msg.Units[0].CurrentType)
	}
	if msg.Units[0].SoftwareVersion != "10.6.0.5" {
		t.Errorf("SoftwareVersion: got '%s'", msg.Units[0].SoftwareVersion)
	}
	if len(msg.PowerSupplies) != 2 {
		t.Fatalf("Expected 2 PSUs, got %d", len(msg.PowerSupplies))
	}
	if msg.PowerSupplies[0].Power != 50 {
		t.Errorf("PSU1 Power: got %d", msg.PowerSupplies[0].Power)
	}
	if msg.PowerSupplies[1].Power != 70 {
		t.Errorf("PSU2 Power: got %d", msg.PowerSupplies[1].Power)
	}
	if len(msg.FanTrays) != 4 {
		t.Fatalf("Expected 4 fan trays, got %d", len(msg.FanTrays))
	}
	if len(msg.FanTrays[0].Fans) != 2 {
		t.Fatalf("FanTray1: expected 2 fans, got %d", len(msg.FanTrays[0].Fans))
	}
	if msg.FanTrays[0].Fans[0].Speed != 8520 {
		t.Errorf("FanTray1 Fan1 Speed: got %d", msg.FanTrays[0].Fans[0].Speed)
	}
}

func TestParseSystemEmpty(t *testing.T) {
	result, err := (&SystemParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
}
