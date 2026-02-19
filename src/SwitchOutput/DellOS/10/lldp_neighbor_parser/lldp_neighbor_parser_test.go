package lldp_neighbor_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseLldpFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_lldp_neighbors_detail.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&LldpParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg LldpNeighborData
	json.Unmarshal(b, &msg)

	if msg.RemoteChassisID != "c8:4b:d6:90:c7:27" {
		t.Errorf("RemoteChassisID: got '%s'", msg.RemoteChassisID)
	}
	if msg.LocalPortID != "ethernet1/1/1:1" {
		t.Errorf("LocalPortID: got '%s'", msg.LocalPortID)
	}
	if msg.RemoteNeighborIndex != 4064 {
		t.Errorf("RemoteNeighborIndex: got %d", msg.RemoteNeighborIndex)
	}
	if msg.RemotePortDescription != "NIC 25Gb QSFP" {
		t.Errorf("RemotePortDescription: got '%s'", msg.RemotePortDescription)
	}
	if msg.RemoteTTL != 121 {
		t.Errorf("RemoteTTL: got %d", msg.RemoteTTL)
	}

	b2, _ := json.Marshal(entries[1].Message)
	var msg2 LldpNeighborData
	json.Unmarshal(b2, &msg2)
	if msg2.LocalPortID != "ethernet1/1/2:1" {
		t.Errorf("Second LocalPortID: got '%s'", msg2.LocalPortID)
	}
}

func TestParseLldpEmpty(t *testing.T) {
	result, err := (&LldpParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 0 {
		t.Errorf("Expected 0, got %d", len(entries))
	}
}

func TestLldpParserDescription(t *testing.T) {
	if (&LldpParser{}).GetDescription() == "" {
		t.Error("GetDescription should not be empty")
	}
}
