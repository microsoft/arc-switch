package main

import (
	"encoding/json"
	"os"
	"testing"
)

const testProcessesCpuInput = `
CPU Statistics of Unit 1
========================
CPUID       5Sec(%)     1Min(%)     5Min(%)
-------------------------------------------
Overall     48.74       46.95       31.67

PID         Process           Runtime(s)      5sec(%)      1min(%)      5min(%)
2875150     .clish            9               97.15        14.95        3.19
1907        base_nas          5707867         17.5         17.5         17.5
2875153     sshd              8               1.45         0.22         0.05
1550        dn_xstp           5707869         1.1          1.1          1.1
2364        netconfd-pro      5707857         1.1          1.1          1.1
75          kipmi0            5707882         0.6          0.6          0.6
1547        dn_sm             5707869         0.6          0.6          0.6
1533        dn_l2_services_   5707870         0.5          0.5          0.5
1519        dn_app_timesync   5707870         0.4          0.4          0.4
1284        dn_pas_svc        5707876         0.3          0.3          0.3
1681        dn_lldp           5707869         0.3          0.3          0.3
1218        dn_pm             5707876         0.2          0.2          0.2`

func TestParseProcessesCpu(t *testing.T) {
	entries, err := parseProcessesCpu(testProcessesCpuInput)
	if err != nil {
		t.Fatalf("parseProcessesCpu returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.DataType != "dell_os10_processes_cpu" {
		t.Errorf("DataType: expected 'dell_os10_processes_cpu', got '%s'", e.DataType)
	}
	if e.Message.UnitID != 1 {
		t.Errorf("UnitID: expected 1, got %d", e.Message.UnitID)
	}
	if e.Message.OverallCPU5Sec != 48.74 {
		t.Errorf("OverallCPU5Sec: expected 48.74, got %f", e.Message.OverallCPU5Sec)
	}
	if e.Message.OverallCPU1Min != 46.95 {
		t.Errorf("OverallCPU1Min: expected 46.95, got %f", e.Message.OverallCPU1Min)
	}
	if e.Message.OverallCPU5Min != 31.67 {
		t.Errorf("OverallCPU5Min: expected 31.67, got %f", e.Message.OverallCPU5Min)
	}
	if len(e.Message.Processes) != 12 {
		t.Fatalf("Expected 12 processes, got %d", len(e.Message.Processes))
	}

	// Verify first process
	p0 := e.Message.Processes[0]
	if p0.PID != 2875150 {
		t.Errorf("Process 0 PID: expected 2875150, got %d", p0.PID)
	}
	if p0.Name != ".clish" {
		t.Errorf("Process 0 Name: expected '.clish', got '%s'", p0.Name)
	}
	if p0.RuntimeSec != 9 {
		t.Errorf("Process 0 Runtime: expected 9, got %d", p0.RuntimeSec)
	}
	if p0.CPU5Sec != 97.15 {
		t.Errorf("Process 0 CPU5Sec: expected 97.15, got %f", p0.CPU5Sec)
	}

	// Verify process with underscore in name
	p7 := e.Message.Processes[7]
	if p7.Name != "dn_l2_services_" {
		t.Errorf("Process 7 Name: expected 'dn_l2_services_', got '%s'", p7.Name)
	}

	// Verify last process
	p11 := e.Message.Processes[11]
	if p11.Name != "dn_pm" {
		t.Errorf("Process 11 Name: expected 'dn_pm', got '%s'", p11.Name)
	}
	if p11.CPU5Sec != 0.2 {
		t.Errorf("Process 11 CPU5Sec: expected 0.2, got %f", p11.CPU5Sec)
	}
}

func TestParseProcessesCpuEmpty(t *testing.T) {
	entries, err := parseProcessesCpu("")
	if err != nil {
		t.Fatalf("Error on empty: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry even for empty, got %d", len(entries))
	}
}

func TestParseProcessesCpuJSON(t *testing.T) {
	entries, err := parseProcessesCpu(testProcessesCpuInput)
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
	err := os.WriteFile(tempFile, []byte(`{"commands":[{"name":"processes-cpu","command":"show processes cpu"}]}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load: %v", err)
	}
	command, err := findCommand(config, "processes-cpu")
	if err != nil {
		t.Errorf("Failed to find: %v", err)
	}
	if command != "show processes cpu" {
		t.Errorf("Expected 'show processes cpu', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
