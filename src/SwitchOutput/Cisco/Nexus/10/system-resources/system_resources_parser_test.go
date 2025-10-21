package system_resources_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseSystemResources(t *testing.T) {
	// Read the sample input file
	inputData, err := os.ReadFile("../show-system-resources.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	// Parse the system resources
	entries, err := parseSystemResources(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse system resources: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Validate data type
	if entry.DataType != "cisco_nexus_system_resources" {
		t.Errorf("Expected data_type 'cisco_nexus_system_resources', got %s", entry.DataType)
	}

	// Validate timestamp format
	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}

	// Validate date format
	if entry.Date == "" {
		t.Error("Date should not be empty")
	}

	// Validate load averages
	if entry.Message.LoadAvg1Min != "0.60" {
		t.Errorf("Expected LoadAvg1Min '0.60', got '%s'", entry.Message.LoadAvg1Min)
	}
	if entry.Message.LoadAvg5Min != "0.63" {
		t.Errorf("Expected LoadAvg5Min '0.63', got '%s'", entry.Message.LoadAvg5Min)
	}
	if entry.Message.LoadAvg15Min != "0.73" {
		t.Errorf("Expected LoadAvg15Min '0.73', got '%s'", entry.Message.LoadAvg15Min)
	}

	// Validate processes
	if entry.Message.ProcessesTotal != 954 {
		t.Errorf("Expected ProcessesTotal 954, got %d", entry.Message.ProcessesTotal)
	}
	if entry.Message.ProcessesRunning != 5 {
		t.Errorf("Expected ProcessesRunning 5, got %d", entry.Message.ProcessesRunning)
	}

	// Validate overall CPU states
	if entry.Message.CPUStateUser != "11.43" {
		t.Errorf("Expected CPUStateUser '11.43', got '%s'", entry.Message.CPUStateUser)
	}
	if entry.Message.CPUStateKernel != "4.52" {
		t.Errorf("Expected CPUStateKernel '4.52', got '%s'", entry.Message.CPUStateKernel)
	}
	if entry.Message.CPUStateIdle != "84.04" {
		t.Errorf("Expected CPUStateIdle '84.04', got '%s'", entry.Message.CPUStateIdle)
	}

	// Validate per-CPU states
	if len(entry.Message.CPUUsage) != 8 {
		t.Errorf("Expected 8 CPU cores, got %d", len(entry.Message.CPUUsage))
	}

	// Validate CPU0 specifically
	if len(entry.Message.CPUUsage) > 0 {
		cpu0 := entry.Message.CPUUsage[0]
		if cpu0.CPUID != "0" {
			t.Errorf("Expected CPU0 CPUID '0', got '%s'", cpu0.CPUID)
		}
		if cpu0.User != "28.71" {
			t.Errorf("Expected CPU0 User '28.71', got '%s'", cpu0.User)
		}
		if cpu0.Kernel != "6.93" {
			t.Errorf("Expected CPU0 Kernel '6.93', got '%s'", cpu0.Kernel)
		}
		if cpu0.Idle != "64.35" {
			t.Errorf("Expected CPU0 Idle '64.35', got '%s'", cpu0.Idle)
		}
	}

	// Validate memory usage
	if entry.Message.MemoryUsageTotal != 24538812 {
		t.Errorf("Expected MemoryUsageTotal 24538812, got %d", entry.Message.MemoryUsageTotal)
	}
	if entry.Message.MemoryUsageUsed != 10765396 {
		t.Errorf("Expected MemoryUsageUsed 10765396, got %d", entry.Message.MemoryUsageUsed)
	}
	if entry.Message.MemoryUsageFree != 13773416 {
		t.Errorf("Expected MemoryUsageFree 13773416, got %d", entry.Message.MemoryUsageFree)
	}

	// Validate kernel buffers and cache
	if entry.Message.KernelBuffers != 59712 {
		t.Errorf("Expected KernelBuffers 59712, got %d", entry.Message.KernelBuffers)
	}
	if entry.Message.KernelCached != 5993960 {
		t.Errorf("Expected KernelCached 5993960, got %d", entry.Message.KernelCached)
	}

	// Validate memory status
	if entry.Message.CurrentMemoryStatus != "OK" {
		t.Errorf("Expected CurrentMemoryStatus 'OK', got '%s'", entry.Message.CurrentMemoryStatus)
	}

	// Validate kernel vmalloc
	if entry.Message.KernelVmallocTotal != 0 {
		t.Errorf("Expected KernelVmallocTotal 0, got %d", entry.Message.KernelVmallocTotal)
	}
	if entry.Message.KernelVmallocFree != 0 {
		t.Errorf("Expected KernelVmallocFree 0, got %d", entry.Message.KernelVmallocFree)
	}
}

func TestJSONSerialization(t *testing.T) {
	// Read the sample input file
	inputData, err := os.ReadFile("../show-system-resources.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	// Parse the system resources
	entries, err := parseSystemResources(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse system resources: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(entries[0])
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Deserialize back
	var entry StandardizedEntry
	err = json.Unmarshal(jsonData, &entry)
	if err != nil {
		t.Fatalf("Failed to deserialize from JSON: %v", err)
	}

	// Validate the round-trip
	if entry.DataType != "cisco_nexus_system_resources" {
		t.Errorf("Round-trip failed: data_type mismatch")
	}
}

func TestUnifiedParserInterface(t *testing.T) {
	parser := &UnifiedParser{}

	// Verify description
	desc := parser.GetDescription()
	if desc == "" {
		t.Error("GetDescription should return a non-empty string")
	}

	// Test parsing
	inputData, err := os.ReadFile("../show-system-resources.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	result, err := parser.Parse(inputData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify result type
	entries, ok := result.([]StandardizedEntry)
	if !ok {
		t.Fatal("Parse should return []StandardizedEntry")
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

func TestCPUCoresParsing(t *testing.T) {
	// Read the sample input file
	inputData, err := os.ReadFile("../show-system-resources.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	// Parse the system resources
	entries, err := parseSystemResources(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse system resources: %v", err)
	}

	entry := entries[0]
	cpuUsage := entry.Message.CPUUsage

	// Validate all 8 CPU cores
	expectedCPUs := []struct {
		id     string
		user   string
		kernel string
		idle   string
	}{
		{"0", "28.71", "6.93", "64.35"},
		{"1", "0.00", "0.00", "100.00"},
		{"2", "0.00", "1.00", "99.00"},
		{"3", "1.96", "1.96", "96.07"},
		{"4", "28.28", "4.04", "67.67"},
		{"5", "20.20", "12.12", "67.67"},
		{"6", "9.09", "7.07", "83.83"},
		{"7", "4.00", "3.00", "93.00"},
	}

	if len(cpuUsage) != len(expectedCPUs) {
		t.Fatalf("Expected %d CPU cores, got %d", len(expectedCPUs), len(cpuUsage))
	}

	for i, expected := range expectedCPUs {
		cpu := cpuUsage[i]
		if cpu.CPUID != expected.id {
			t.Errorf("CPU%d: Expected CPUID '%s', got '%s'", i, expected.id, cpu.CPUID)
		}
		if cpu.User != expected.user {
			t.Errorf("CPU%d: Expected User '%s', got '%s'", i, expected.user, cpu.User)
		}
		if cpu.Kernel != expected.kernel {
			t.Errorf("CPU%d: Expected Kernel '%s', got '%s'", i, expected.kernel, cpu.Kernel)
		}
		if cpu.Idle != expected.idle {
			t.Errorf("CPU%d: Expected Idle '%s', got '%s'", i, expected.idle, cpu.Idle)
		}
	}
}
