package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const testInventoryInput = `Product               : S5248F-ON
Description           : S5248F-ON 48x25GbE SFP28, 4x100GbE QSFP28, 2x200GbE QSFP-DD Interface Module
Software version      : 10.6.0.5
Product Base          :
Product Serial Number :
Product Part Number   :
Unit Type                     Part Number  Rev  Piece Part ID             Svc Tag  Exprs Svc Code
-------------------------------------------------------------------------------------------------
* 1  S5248F-ON                006Y6V       A03  TH-006Y6V-CET00-332-60OZ  5M44SR3  122 211 099 03`

func TestParseInventory(t *testing.T) {
	entries, err := parseInventory(testInventoryInput)
	if err != nil {
		t.Fatalf("parseInventory returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.DataType != "dell_os10_inventory" {
		t.Errorf("DataType: expected 'dell_os10_inventory', got '%s'", e.DataType)
	}
	if e.Message.Product != "S5248F-ON" {
		t.Errorf("Product: expected 'S5248F-ON', got '%s'", e.Message.Product)
	}
	if e.Message.SoftwareVersion != "10.6.0.5" {
		t.Errorf("SoftwareVersion: expected '10.6.0.5', got '%s'", e.Message.SoftwareVersion)
	}
	if !strings.Contains(e.Message.Description, "48x25GbE") {
		t.Errorf("Description should contain '48x25GbE', got '%s'", e.Message.Description)
	}
	if len(e.Message.Units) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(e.Message.Units))
	}
	if e.Message.Units[0].UnitID != "1" {
		t.Errorf("Unit ID: expected '1', got '%s'", e.Message.Units[0].UnitID)
	}
	if e.Message.Units[0].Type != "S5248F-ON" {
		t.Errorf("Unit Type: expected 'S5248F-ON', got '%s'", e.Message.Units[0].Type)
	}
	if e.Message.Units[0].ServiceTag != "5M44SR3" {
		t.Errorf("ServiceTag: expected '5M44SR3', got '%s'", e.Message.Units[0].ServiceTag)
	}
	if e.Message.Units[0].Revision != "A03" {
		t.Errorf("Revision: expected 'A03', got '%s'", e.Message.Units[0].Revision)
	}
}

func TestParseInventoryEmpty(t *testing.T) {
	entries, err := parseInventory("")
	if err != nil {
		t.Fatalf("Error on empty: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry even for empty, got %d", len(entries))
	}
}

func TestParseInventoryJSON(t *testing.T) {
	entries, err := parseInventory(testInventoryInput)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	jsonBytes, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	var unmarshaled []StandardizedEntry
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	err := os.WriteFile(tempFile, []byte(`{"commands":[{"name":"inventory","command":"show inventory"}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load: %v", err)
	}
	command, err := findCommand(config, "inventory")
	if err != nil {
		t.Errorf("Failed to find: %v", err)
	}
	if command != "show inventory" {
		t.Errorf("Expected 'show inventory', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
