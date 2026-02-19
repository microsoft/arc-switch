package main

import (
	"encoding/json"
	"os"
	"testing"
)

const testSystemInput = `Node Id              : 1
MAC                  : c4:5a:b1:36:bb:85
Number of MACs       : 256
Up Time              : 9 weeks 3 days 01:30:55
DiagOS               : 3.40.4.1-9
PCIe Version         : 2.6

-- Unit 1 --
Status                     : up
System Identifier          : 1
Down Reason                : unknown
Digital Optical Monitoring : disable
System Location LED        : off
Required Type              : S5248F
Current Type               : S5248F
Hardware Revision          : A03
Software Version           : 10.6.0.5
Physical Ports             : 48x25GbE, 4x100GbE, 2x200GbE
BIOS                          :   3.40.0.9-15
BMC                           :   1.07
ONIE                          :   3.40.1.1-9
SSD                           :   L20B12
OCORE-FPGA@pci_0000_04_00.0   :   3.4
System CPLD                   : 0.8
Secondary CPLD 1              : 1.0
Secondary CPLD 2              : 1.0
Secondary CPLD 3              : 0.0
Secondary CPLD 4              : 0.0

-- Power Supplies --
PSU-ID  Status      Type    Power(w) AvgPower(w) AvgPowerStartTime AirFlow   Fan  Speed(rpm)  Status
-------------------------------------------------------------------------------------------------------
1       up          AC      50       50          12/14/2025-19:27  REVERSE   1    8280        up

2       up          AC      70       70          12/14/2025-19:27  REVERSE   1    8280        up


-- Fan Status --
FanTray  Status      AirFlow   Fan  Speed(rpm)  Status
----------------------------------------------------------------
1        up          REVERSE   1    8520        up
                               2    7680        up

2        up          REVERSE   1    8400        up
                               2    7800        up

3        up          REVERSE   1    8520        up
                               2    7680        up

4        up          REVERSE   1    8520        up
                               2    7800        up`

func TestParseSystem(t *testing.T) {
	entries, err := parseSystem(testSystemInput)
	if err != nil {
		t.Fatalf("parseSystem returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.DataType != "dell_os10_system" {
		t.Errorf("DataType: expected 'dell_os10_system', got '%s'", e.DataType)
	}

	// Top-level fields
	if e.Message.NodeID != 1 {
		t.Errorf("NodeID: expected 1, got %d", e.Message.NodeID)
	}
	if e.Message.MAC != "c4:5a:b1:36:bb:85" {
		t.Errorf("MAC: expected 'c4:5a:b1:36:bb:85', got '%s'", e.Message.MAC)
	}
	if e.Message.NumberOfMACs != 256 {
		t.Errorf("NumberOfMACs: expected 256, got %d", e.Message.NumberOfMACs)
	}
	if e.Message.UpTime != "9 weeks 3 days 01:30:55" {
		t.Errorf("UpTime: expected '9 weeks 3 days 01:30:55', got '%s'", e.Message.UpTime)
	}

	// Unit
	if len(e.Message.Units) != 1 {
		t.Fatalf("Expected 1 unit, got %d", len(e.Message.Units))
	}
	if e.Message.Units[0].Status != "up" {
		t.Errorf("Unit Status: expected 'up', got '%s'", e.Message.Units[0].Status)
	}
	if e.Message.Units[0].CurrentType != "S5248F" {
		t.Errorf("Unit CurrentType: expected 'S5248F', got '%s'", e.Message.Units[0].CurrentType)
	}
	if e.Message.Units[0].SoftwareVersion != "10.6.0.5" {
		t.Errorf("Unit SoftwareVersion: expected '10.6.0.5', got '%s'", e.Message.Units[0].SoftwareVersion)
	}
	if e.Message.Units[0].HardwareRevision != "A03" {
		t.Errorf("Unit HardwareRevision: expected 'A03', got '%s'", e.Message.Units[0].HardwareRevision)
	}

	// Power supplies
	if len(e.Message.PowerSupplies) != 2 {
		t.Fatalf("Expected 2 PSUs, got %d", len(e.Message.PowerSupplies))
	}
	if e.Message.PowerSupplies[0].PSUID != 1 {
		t.Errorf("PSU1 ID: expected 1, got %d", e.Message.PowerSupplies[0].PSUID)
	}
	if e.Message.PowerSupplies[0].Status != "up" {
		t.Errorf("PSU1 Status: expected 'up', got '%s'", e.Message.PowerSupplies[0].Status)
	}
	if e.Message.PowerSupplies[0].Power != 50 {
		t.Errorf("PSU1 Power: expected 50, got %d", e.Message.PowerSupplies[0].Power)
	}
	if e.Message.PowerSupplies[0].AirFlow != "REVERSE" {
		t.Errorf("PSU1 AirFlow: expected 'REVERSE', got '%s'", e.Message.PowerSupplies[0].AirFlow)
	}
	if e.Message.PowerSupplies[1].Power != 70 {
		t.Errorf("PSU2 Power: expected 70, got %d", e.Message.PowerSupplies[1].Power)
	}

	// Fan trays
	if len(e.Message.FanTrays) != 4 {
		t.Fatalf("Expected 4 fan trays, got %d", len(e.Message.FanTrays))
	}
	if e.Message.FanTrays[0].TrayID != 1 {
		t.Errorf("FanTray1 ID: expected 1, got %d", e.Message.FanTrays[0].TrayID)
	}
	if len(e.Message.FanTrays[0].Fans) != 2 {
		t.Fatalf("FanTray1: expected 2 fans, got %d", len(e.Message.FanTrays[0].Fans))
	}
	if e.Message.FanTrays[0].Fans[0].Speed != 8520 {
		t.Errorf("FanTray1 Fan1 Speed: expected 8520, got %d", e.Message.FanTrays[0].Fans[0].Speed)
	}
	if e.Message.FanTrays[0].Fans[1].Speed != 7680 {
		t.Errorf("FanTray1 Fan2 Speed: expected 7680, got %d", e.Message.FanTrays[0].Fans[1].Speed)
	}
	if e.Message.FanTrays[0].Fans[0].Status != "up" {
		t.Errorf("FanTray1 Fan1 Status: expected 'up', got '%s'", e.Message.FanTrays[0].Fans[0].Status)
	}
}

func TestParseSystemEmpty(t *testing.T) {
	entries, err := parseSystem("")
	if err != nil {
		t.Fatalf("Error on empty: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry even for empty, got %d", len(entries))
	}
}

func TestParseSystemJSON(t *testing.T) {
	entries, err := parseSystem(testSystemInput)
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
	err := os.WriteFile(tempFile, []byte(`{"commands":[{"name":"system","command":"show system"}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load: %v", err)
	}
	command, err := findCommand(config, "system")
	if err != nil {
		t.Errorf("Failed to find: %v", err)
	}
	if command != "show system" {
		t.Errorf("Expected 'show system', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
