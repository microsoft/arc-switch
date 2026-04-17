package transform

import (
	"encoding/json"
	"os"
	"testing"

	"gnmi-collector/internal/gnmi"
)

// loadTestData loads a gNMI dump JSON file and returns parsed Notifications.
func loadTestData(t *testing.T, filename string) []gnmi.Notification {
	t.Helper()
	data, err := os.ReadFile("../../testdata/" + filename)
	if err != nil {
		t.Fatalf("failed to read testdata/%s: %v", filename, err)
	}
	if len(data) == 0 {
		return nil
	}
	var notifications []gnmi.Notification
	if err := json.Unmarshal(data, &notifications); err != nil {
		t.Fatalf("failed to parse testdata/%s: %v", filename, err)
	}
	return notifications
}

func TestInterfaceCountersTransformer(t *testing.T) {
	notifications := loadTestData(t, "interface-counters.json")
	tr := &InterfaceCountersTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results, got 0")
	}

	// Check first result has required fields
	entry := results[0]
	if entry.DataType != "interface_counters" {
		t.Errorf("data_type = %q, want interface_counters", entry.DataType)
	}
	if entry.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}

	msg, ok := entry.Message.(map[string]interface{})
	if !ok {
		t.Fatalf("message is not a map, got %T", entry.Message)
	}

	requiredFields := []string{
		"interface_name", "interface_type",
		"in_octets", "in_ucast_pkts", "in_mcast_pkts", "in_bcast_pkts",
		"out_octets", "out_ucast_pkts", "out_mcast_pkts", "out_bcast_pkts",
	}
	for _, f := range requiredFields {
		if _, ok := msg[f]; !ok {
			t.Errorf("missing required field %q", f)
		}
	}

	t.Logf("Produced %d interface counter entries", len(results))
}

func TestInterfaceStatusTransformer(t *testing.T) {
	notifications := loadTestData(t, "interface-status.json")
	tr := &InterfaceStatusTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result (wrapper), got %d", len(results))
	}

	entry := results[0]
	if entry.DataType != "interface_status" {
		t.Errorf("data_type = %q", entry.DataType)
	}

	msg, ok := entry.Message.(map[string]interface{})
	if !ok {
		t.Fatalf("message is not a map")
	}

	ifaces, ok := msg["interfaces"]
	if !ok {
		t.Fatal("missing interfaces field")
	}

	ifList, ok := ifaces.([]InterfaceStatusEntry)
	if !ok {
		t.Fatalf("interfaces is not []InterfaceStatusEntry, got %T", ifaces)
	}

	if len(ifList) == 0 {
		t.Fatal("expected interface entries")
	}

	t.Logf("Produced %d interface status entries", len(ifList))

	// Spot check: all entries should have a port and status
	for i, e := range ifList {
		if e.Port == "" {
			t.Errorf("entry[%d] missing port", i)
		}
		if e.Status == "" {
			t.Errorf("entry[%d] missing status", i)
		}
	}
}

func TestBgpSummaryTransformer(t *testing.T) {
	notifications := loadTestData(t, "bgp-neighbors.json")
	tr := &BgpSummaryTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected BGP results")
	}

	// We know there are 4 neighbors from the dump
	if len(results) < 3 {
		t.Errorf("expected at least 3 BGP entries, got %d", len(results))
	}

	// Check entries have core fields (some neighbors may have partial data)
	hasFullEntry := false
	for i, entry := range results {
		msg, ok := entry.Message.(map[string]interface{})
		if !ok {
			t.Fatalf("result[%d] message not a map", i)
		}
		if msg["neighbor_address"] == "" {
			t.Errorf("result[%d] missing neighbor_address", i)
		}
		if msg["session_state"] != "" {
			hasFullEntry = true
		}
	}

	if !hasFullEntry {
		t.Error("expected at least one BGP entry with session_state")
	}

	t.Logf("Produced %d BGP neighbor entries", len(results))
}

func TestLldpNeighborTransformer(t *testing.T) {
	notifications := loadTestData(t, "lldp-neighbors.json")
	tr := &LldpNeighborTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected LLDP results")
	}

	for i, entry := range results {
		msg, ok := entry.Message.(map[string]interface{})
		if !ok {
			t.Fatalf("result[%d] message not a map", i)
		}
		if msg["chassis_id"] == "" {
			t.Errorf("result[%d] missing chassis_id", i)
		}
		if msg["system_name"] == "" {
			t.Errorf("result[%d] missing system_name", i)
		}
	}

	t.Logf("Produced %d LLDP neighbor entries", len(results))
}

func TestMacAddressTransformer(t *testing.T) {
	notifications := loadTestData(t, "mac-table.json")
	tr := &MacAddressTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected MAC table results")
	}

	for i, entry := range results {
		msg, ok := entry.Message.(map[string]interface{})
		if !ok {
			t.Fatalf("result[%d] message not a map", i)
		}
		if msg["mac_address"] == "" {
			t.Errorf("result[%d] missing mac_address", i)
		}
	}

	t.Logf("Produced %d MAC table entries", len(results))
}

func TestEnvironmentTempTransformer(t *testing.T) {
	notifications := loadTestData(t, "temperature.json")
	tr := &EnvironmentTempTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected temperature results")
	}

	entry := results[0]
	msg, ok := entry.Message.(map[string]interface{})
	if !ok {
		t.Fatal("message not a map")
	}
	if msg["current_temp"] == "" {
		t.Error("missing current_temp")
	}
	if msg["status"] == "" {
		t.Error("missing status")
	}

	t.Logf("Produced %d temperature entries", len(results))
}

func TestEnvironmentPowerTransformer(t *testing.T) {
notifications := loadTestData(t, "power-supply.json")
tr := &EnvironmentPowerTransformer{}

results, err := tr.Transform(notifications)
if err != nil {
t.Fatalf("transform error: %v", err)
}

if len(results) < 2 {
t.Fatalf("expected at least 2 PSU results (one per PSU), got %d", len(results))
}

// Each result is now a flat PSU entry
for _, entry := range results {
msg, ok := entry.Message.(map[string]interface{})
if !ok {
t.Fatal("message not a map")
}

// Check base64 decode worked - capacity should be a numeric string, not base64
if cap, ok := msg["total_capacity"].(string); ok {
if cap == "RIl9cQ==" {
t.Error("capacity was not decoded from base64")
}
}
}

t.Logf("Produced %d PSU entries", len(results))
}
func TestInventoryTransformer(t *testing.T) {
	notifications := loadTestData(t, "platform-inventory.json")
	tr := &InventoryTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected inventory results")
	}

	for i, entry := range results {
		msg, ok := entry.Message.(map[string]interface{})
		if !ok {
			t.Fatalf("result[%d] message not a map", i)
		}
		if msg["name"] == "" && msg["serial_number"] == "" {
			t.Errorf("result[%d] missing both name and serial_number", i)
		}
	}

	t.Logf("Produced %d inventory entries", len(results))
}

func TestTransceiverTransformer(t *testing.T) {
	notifications := loadTestData(t, "transceiver.json")
	tr := &TransceiverTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected transceiver results")
	}

	for i, entry := range results {
		msg, ok := entry.Message.(map[string]interface{})
		if !ok {
			t.Fatalf("result[%d] message not a map", i)
		}
		if _, ok := msg["transceiver_present"]; !ok {
			t.Errorf("result[%d] missing transceiver_present", i)
		}
	}

	t.Logf("Produced %d transceiver entries", len(results))
}

func TestSystemResourcesTransformer(t *testing.T) {
	notifications := loadTestData(t, "system-cpus.json")
	tr := &SystemResourcesTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	msg, ok := results[0].Message.(map[string]interface{})
	if !ok {
		t.Fatal("message not a map")
	}

	if _, ok := msg["cpu_usage"]; !ok {
		t.Error("missing cpu_usage")
	}

	t.Logf("System resources: %v", msg)
}

func TestSystemUptimeTransformer(t *testing.T) {
	notifications := loadTestData(t, "system-state.json")
	tr := &SystemUptimeTransformer{}

	results, err := tr.Transform(notifications)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	msg, ok := results[0].Message.(map[string]interface{})
	if !ok {
		t.Fatal("message not a map")
	}

	if msg["hostname"] != "rr1-n42-r07-9336hl-13-1a" {
		t.Errorf("hostname = %q, want rr1-n42-r07-9336hl-13-1a", msg["hostname"])
	}
	if _, ok := msg["system_uptime_days"]; !ok {
		t.Error("missing system_uptime_days")
	}

	t.Logf("System uptime: %v", msg)
}

func TestArpTransformerEmpty(t *testing.T) {
	// ARP table dump was empty (0 bytes) — transformer should handle gracefully
	tr := &ArpTransformer{}
	results, err := tr.Transform(nil)
	if err != nil {
		t.Fatalf("transform error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty ARP, got %d", len(results))
	}
}

func TestParseCapabilityString(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"  ", nil},
		{"bridge", []string{"bridge"}},
		{"bridge,router", []string{"bridge", "router"}},
		{"bridge router", []string{"bridge", "router"}},
		{"bridge, router", []string{"bridge", "router"}},
		{"bridge, router, station-only", []string{"bridge", "router", "station-only"}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCapabilityString(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseCapabilityString(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseCapabilityString(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}