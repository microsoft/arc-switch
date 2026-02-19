package bgp_summary_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseBgpSummaryFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_ip_bgp_summary.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&BgpSummaryParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg BgpSummaryData
	json.Unmarshal(b, &msg)

	if msg.RouterID != "10.0.0.1" {
		t.Errorf("RouterID: got '%s'", msg.RouterID)
	}
	if msg.LocalAS != 65001 {
		t.Errorf("LocalAS: got %d", msg.LocalAS)
	}
	if msg.ASNType != "private" {
		t.Errorf("ASNType: got '%s'", msg.ASNType)
	}
	if msg.NeighborsCount != 3 {
		t.Errorf("NeighborsCount: got %d", msg.NeighborsCount)
	}
	if len(msg.Neighbors) != 3 {
		t.Fatalf("Expected 3 neighbors, got %d", len(msg.Neighbors))
	}
	if msg.Neighbors[0].SessionType != "iBGP" {
		t.Errorf("Neighbor 0 SessionType: got '%s'", msg.Neighbors[0].SessionType)
	}
	if msg.Neighbors[0].State != "Established" {
		t.Errorf("Neighbor 0 State: got '%s'", msg.Neighbors[0].State)
	}
	if msg.Neighbors[0].PrefixReceived != 50 {
		t.Errorf("Neighbor 0 PrefixReceived: got %d", msg.Neighbors[0].PrefixReceived)
	}
	if msg.Neighbors[1].SessionType != "eBGP" {
		t.Errorf("Neighbor 1 SessionType: got '%s'", msg.Neighbors[1].SessionType)
	}
	if msg.Neighbors[2].State != "Active" {
		t.Errorf("Neighbor 2 State: got '%s'", msg.Neighbors[2].State)
	}
}

func TestParseBgpSummaryEmpty(t *testing.T) {
	result, err := (&BgpSummaryParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error on empty: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 0 {
		t.Errorf("Expected 0, got %d", len(entries))
	}
}

func TestClassifyASN(t *testing.T) {
	tests := []struct {
		asn      int64
		expected string
	}{
		{65001, "private"}, {64512, "private"}, {65534, "private"},
		{100, "public"}, {64511, "public"}, {4200000000, "private"},
	}
	for _, tc := range tests {
		if result := ClassifyASN(tc.asn); result != tc.expected {
			t.Errorf("ClassifyASN(%d) = %s; want %s", tc.asn, result, tc.expected)
		}
	}
}
