package processes_cpu_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseProcessesCpuFromFile(t *testing.T) {
	data, err := os.ReadFile("testdata/show_processes_cpu.txt")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	result, err := (&ProcessesCpuParser{}).Parse(data)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
	b, _ := json.Marshal(entries[0].Message)
	var msg ProcessesCpuData
	json.Unmarshal(b, &msg)

	if msg.UnitID != 1 {
		t.Errorf("UnitID: got %d", msg.UnitID)
	}
	if msg.OverallCPU5Sec != 48.74 {
		t.Errorf("OverallCPU5Sec: got %f", msg.OverallCPU5Sec)
	}
	if msg.OverallCPU1Min != 46.95 {
		t.Errorf("OverallCPU1Min: got %f", msg.OverallCPU1Min)
	}
	if msg.OverallCPU5Min != 31.67 {
		t.Errorf("OverallCPU5Min: got %f", msg.OverallCPU5Min)
	}
	if len(msg.Processes) != 12 {
		t.Fatalf("Expected 12 processes, got %d", len(msg.Processes))
	}
	if msg.Processes[0].PID != 2875150 {
		t.Errorf("Process 0 PID: got %d", msg.Processes[0].PID)
	}
	if msg.Processes[0].Name != ".clish" {
		t.Errorf("Process 0 Name: got '%s'", msg.Processes[0].Name)
	}
	if msg.Processes[7].Name != "dn_l2_services_" {
		t.Errorf("Process 7 Name: got '%s'", msg.Processes[7].Name)
	}
}

func TestParseProcessesCpuEmpty(t *testing.T) {
	result, err := (&ProcessesCpuParser{}).Parse([]byte(""))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	entries := result.([]StandardizedEntry)
	if len(entries) != 1 {
		t.Fatalf("Expected 1, got %d", len(entries))
	}
}
