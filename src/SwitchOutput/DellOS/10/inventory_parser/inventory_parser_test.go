package inventory_parser

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseInventoryFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_inventory.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&InventoryParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg InventoryData
	json.Unmarshal(b, &msg)

	if msg.Product != "S5248F-ON" {
		t.Errorf("Product: got '%s'", msg.Product)
	}
	if msg.SoftwareVersion != "10.6.0.5" {
		t.Errorf("SoftwareVersion: got '%s'", msg.SoftwareVersion)
	}
	if !strings.Contains(msg.Description, "48x25GbE") {
		t.Errorf("Description missing '48x25GbE': '%s'", msg.Description)
	}
	if len(msg.Units) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(msg.Units))
	}
	if msg.Units[0].ServiceTag != "5M44SR3" {
		t.Errorf("ServiceTag: got '%s'", msg.Units[0].ServiceTag)
	}
	if msg.Units[0].Revision != "A03" {
		t.Errorf("Revision: got '%s'", msg.Units[0].Revision)
	}
}

func TestParseInventoryEmpty(t *testing.T) {
	result, err := (&InventoryParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
}
