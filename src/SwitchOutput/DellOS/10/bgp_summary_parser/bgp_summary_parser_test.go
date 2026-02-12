package bgp_summary_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestClassifyASN(t *testing.T) {
	tests := []struct {
		asn      int64
		expected string
	}{
		{100, "public"},
		{64512, "private"},
		{65000, "private"},
		{65534, "private"},
		{65535, "public"},
		{4200000000, "private"},
		{4294967294, "private"},
		{4294967295, "public"},
		{1, "public"},
	}

	for _, test := range tests {
		result := classifyASN(test.asn)
		if result != test.expected {
			t.Errorf("classifyASN(%d) = %s; expected %s", test.asn, result, test.expected)
		}
	}
}

func TestDetermineSessionType(t *testing.T) {
	tests := []struct {
		neighborAS int64
		localAS    int64
		expected   string
	}{
		{65001, 65001, "iBGP"},
		{65001, 65002, "eBGP"},
		{100, 100, "iBGP"},
		{100, 200, "eBGP"},
	}

	for _, test := range tests {
		result := determineSessionType(test.neighborAS, test.localAS)
		if result != test.expected {
			t.Errorf("determineSessionType(%d, %d) = %s; expected %s", test.neighborAS, test.localAS, result, test.expected)
		}
	}
}

func TestParseUpDownTime(t *testing.T) {
	tests := []struct {
		input              string
		expectedRaw        string
		expectedWeeks      int
		expectedDays       int
		expectedHours      int
		expectedMinutes    int
		expectedSeconds    int
		expectedTotalSecs  int64
	}{
		{"11w2d", "11w2d", 11, 2, 0, 0, 0, 11*7*24*3600 + 2*24*3600},
		{"1d21h", "1d21h", 0, 1, 21, 0, 0, 1*24*3600 + 21*3600},
		{"00:05:30", "00:05:30", 0, 0, 0, 5, 30, 5*60 + 30},
		{"01:30:00", "01:30:00", 0, 0, 1, 30, 0, 1*3600 + 30*60},
		{"never", "", 0, 0, 0, 0, 0, 0},
		{"", "", 0, 0, 0, 0, 0, 0},
	}

	for _, test := range tests {
		raw, parsed := parseUpDownTime(test.input)
		if raw != test.expectedRaw {
			t.Errorf("parseUpDownTime(%s) raw = %s; expected %s", test.input, raw, test.expectedRaw)
		}
		if parsed == nil && test.expectedTotalSecs != 0 {
			t.Errorf("parseUpDownTime(%s) returned nil parsed; expected non-nil", test.input)
			continue
		}
		if parsed != nil {
			if parsed.Weeks != test.expectedWeeks {
				t.Errorf("parseUpDownTime(%s) weeks = %d; expected %d", test.input, parsed.Weeks, test.expectedWeeks)
			}
			if parsed.Days != test.expectedDays {
				t.Errorf("parseUpDownTime(%s) days = %d; expected %d", test.input, parsed.Days, test.expectedDays)
			}
			if parsed.Hours != test.expectedHours {
				t.Errorf("parseUpDownTime(%s) hours = %d; expected %d", test.input, parsed.Hours, test.expectedHours)
			}
			if parsed.Minutes != test.expectedMinutes {
				t.Errorf("parseUpDownTime(%s) minutes = %d; expected %d", test.input, parsed.Minutes, test.expectedMinutes)
			}
			if parsed.Seconds != test.expectedSeconds {
				t.Errorf("parseUpDownTime(%s) seconds = %d; expected %d", test.input, parsed.Seconds, test.expectedSeconds)
			}
			if parsed.TotalSeconds != test.expectedTotalSecs {
				t.Errorf("parseUpDownTime(%s) total_seconds = %d; expected %d", test.input, parsed.TotalSeconds, test.expectedTotalSecs)
			}
		}
	}
}

func TestParseBGPSummary(t *testing.T) {
	// Sample Dell OS10 show ip bgp summary output
	sampleInput := `BGP router identifier 10.1.1.1, local AS number 65001
BGP local RIB : Routes to be Added 0, Replaced 0, Withdrawn 0

2 neighbor(s) using 4096 bytes of memory

Neighbor        AS      MsgRcvd MsgSent TblVer  InQ OutQ Up/Down   State/PfxRcd
10.1.1.2        65001   12345   12346   100     0   0    11w2d     150
10.1.2.1        65002   9876    9877    100     0   0    1d21h     200`

	entries, err := parseBGPSummary(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse BGP summary: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	if len(entries) > 0 {
		entry := entries[0]
		if entry.DataType != "dell_os10_bgp_summary" {
			t.Errorf("data_type: expected 'dell_os10_bgp_summary', got '%s'", entry.DataType)
		}
		if entry.Message.RouterID != "10.1.1.1" {
			t.Errorf("RouterID: expected '10.1.1.1', got '%s'", entry.Message.RouterID)
		}
		if entry.Message.LocalAS != 65001 {
			t.Errorf("LocalAS: expected 65001, got %d", entry.Message.LocalAS)
		}
		if entry.Message.ASNType != "private" {
			t.Errorf("ASNType: expected 'private', got '%s'", entry.Message.ASNType)
		}
		if entry.Message.NeighborsCount != 2 {
			t.Errorf("NeighborsCount: expected 2, got %d", entry.Message.NeighborsCount)
		}
		if entry.Message.MemoryUsed != 4096 {
			t.Errorf("MemoryUsed: expected 4096, got %d", entry.Message.MemoryUsed)
		}
		if entry.Message.VRF != "default" {
			t.Errorf("VRF: expected 'default', got '%s'", entry.Message.VRF)
		}

		// Verify neighbors
		if len(entry.Message.Neighbors) != 2 {
			t.Errorf("Expected 2 neighbors, got %d", len(entry.Message.Neighbors))
		}

		// Verify first neighbor (iBGP)
		if len(entry.Message.Neighbors) > 0 {
			neighbor := entry.Message.Neighbors[0]
			if neighbor.NeighborID != "10.1.1.2" {
				t.Errorf("NeighborID: expected '10.1.1.2', got '%s'", neighbor.NeighborID)
			}
			if neighbor.NeighborAS != 65001 {
				t.Errorf("NeighborAS: expected 65001, got %d", neighbor.NeighborAS)
			}
			if neighbor.SessionType != "iBGP" {
				t.Errorf("SessionType: expected 'iBGP', got '%s'", neighbor.SessionType)
			}
			if neighbor.State != "Established" {
				t.Errorf("State: expected 'Established', got '%s'", neighbor.State)
			}
			if neighbor.PrefixReceived != 150 {
				t.Errorf("PrefixReceived: expected 150, got %d", neighbor.PrefixReceived)
			}
			if neighbor.MsgRecvd != 12345 {
				t.Errorf("MsgRecvd: expected 12345, got %d", neighbor.MsgRecvd)
			}
			if neighbor.MsgSent != 12346 {
				t.Errorf("MsgSent: expected 12346, got %d", neighbor.MsgSent)
			}
		}

		// Verify second neighbor (eBGP)
		if len(entry.Message.Neighbors) > 1 {
			neighbor := entry.Message.Neighbors[1]
			if neighbor.SessionType != "eBGP" {
				t.Errorf("SessionType for second neighbor: expected 'eBGP', got '%s'", neighbor.SessionType)
			}
		}
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(entries)
	if err != nil {
		t.Errorf("Failed to marshal entries to JSON: %v", err)
	}

	var unmarshaledEntries []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledEntries)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
}

func TestParseBGPSummaryWithVRF(t *testing.T) {
	sampleInput := `VRF: production
BGP router identifier 10.2.1.1, local AS number 65100
BGP local RIB : Routes to be Added 5, Replaced 2, Withdrawn 1

1 neighbor(s) using 2048 bytes of memory

Neighbor        AS      MsgRcvd MsgSent TblVer  InQ OutQ Up/Down   State/PfxRcd
10.2.1.2        65100   5000    5001    50      0   0    2d3h      100`

	entries, err := parseBGPSummary(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse BGP summary: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	if len(entries) > 0 {
		if entries[0].Message.VRF != "production" {
			t.Errorf("VRF: expected 'production', got '%s'", entries[0].Message.VRF)
		}
		if entries[0].Message.RoutesToAdd != 5 {
			t.Errorf("RoutesToAdd: expected 5, got %d", entries[0].Message.RoutesToAdd)
		}
		if entries[0].Message.RoutesToReplace != 2 {
			t.Errorf("RoutesToReplace: expected 2, got %d", entries[0].Message.RoutesToReplace)
		}
		if entries[0].Message.RoutesWithdrawn != 1 {
			t.Errorf("RoutesWithdrawn: expected 1, got %d", entries[0].Message.RoutesWithdrawn)
		}
	}
}

func TestParseBGPSummaryNeighborStates(t *testing.T) {
	sampleInput := `BGP router identifier 10.1.1.1, local AS number 65001

1 neighbor(s) using 2048 bytes of memory

Neighbor        AS      MsgRcvd MsgSent TblVer  InQ OutQ Up/Down   State/PfxRcd
10.1.1.2        65001   0       0       0       0   0    never     Idle`

	entries, err := parseBGPSummary(sampleInput)
	if err != nil {
		t.Errorf("Failed to parse BGP summary: %v", err)
	}

	if len(entries) > 0 && len(entries[0].Message.Neighbors) > 0 {
		neighbor := entries[0].Message.Neighbors[0]
		if neighbor.State != "Idle" {
			t.Errorf("State: expected 'Idle', got '%s'", neighbor.State)
		}
		if neighbor.PrefixReceived != 0 {
			t.Errorf("PrefixReceived: expected 0 for Idle state, got %d", neighbor.PrefixReceived)
		}
	}
}

func TestParseBGPSummaryEmptyInput(t *testing.T) {
	_, err := parseBGPSummary("")
	if err == nil {
		t.Error("Expected error for empty input")
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "bgp-summary",
				"command": "show ip bgp summary"
			}
		]
	}`

	err := os.WriteFile(tempFile, []byte(commandsData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test commands file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load commands from file: %v", err)
	}

	command, err := findBGPSummaryCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show ip bgp summary" {
		t.Errorf("Expected command 'show ip bgp summary', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
