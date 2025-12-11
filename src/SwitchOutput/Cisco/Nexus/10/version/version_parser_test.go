package version_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseVersion(t *testing.T) {
	// Read the sample input file
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	// Parse the version info
	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Validate data type
	if entry.DataType != "cisco_nexus_version" {
		t.Errorf("Expected data_type 'cisco_nexus_version', got %s", entry.DataType)
	}

	// Validate timestamp format
	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}

	// Validate date format
	if entry.Date == "" {
		t.Error("Date should not be empty")
	}
}

func TestBIOSVersion(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.BIOSVersion != "05.53" {
		t.Errorf("Expected BIOSVersion '05.53', got '%s'", data.BIOSVersion)
	}

	if data.BIOSCompileTime != "01/22/2025" {
		t.Errorf("Expected BIOSCompileTime '01/22/2025', got '%s'", data.BIOSCompileTime)
	}
}

func TestNXOSVersion(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.NXOSVersion != "10.6(1)" {
		t.Errorf("Expected NXOSVersion '10.6(1)', got '%s'", data.NXOSVersion)
	}

	if data.ReleaseType != "Feature Release" {
		t.Errorf("Expected ReleaseType 'Feature Release', got '%s'", data.ReleaseType)
	}

	if data.HostNXOSVersion != "10.6(1)" {
		t.Errorf("Expected HostNXOSVersion '10.6(1)', got '%s'", data.HostNXOSVersion)
	}

	if data.NXOSImageFile != "bootflash:///nxos64-cs.10.6.1.F.bin" {
		t.Errorf("Expected NXOSImageFile 'bootflash:///nxos64-cs.10.6.1.F.bin', got '%s'", data.NXOSImageFile)
	}

	if data.NXOSCompileTime != "7/31/2025 12:00:00" {
		t.Errorf("Expected NXOSCompileTime '7/31/2025 12:00:00', got '%s'", data.NXOSCompileTime)
	}

	if data.NXOSTimestamp != "08/12/2025 14:18:15" {
		t.Errorf("Expected NXOSTimestamp '08/12/2025 14:18:15', got '%s'", data.NXOSTimestamp)
	}

	if data.BootMode != "LXC" {
		t.Errorf("Expected BootMode 'LXC', got '%s'", data.BootMode)
	}
}

func TestHardwareInfo(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	data := entries[0].Message

	if data.ChassisID != "cisco Nexus9000 C9336C-FX2 Chassis" {
		t.Errorf("Expected ChassisID 'cisco Nexus9000 C9336C-FX2 Chassis', got '%s'", data.ChassisID)
	}

	if data.CPUName != "Intel(R) Xeon(R) CPU D-1526 @ 1.80GHz" {
		t.Errorf("Expected CPUName 'Intel(R) Xeon(R) CPU D-1526 @ 1.80GHz', got '%s'", data.CPUName)
	}

	if data.MemoryKB != 24538812 {
		t.Errorf("Expected MemoryKB 24538812, got %d", data.MemoryKB)
	}

	if data.ProcessorBoardID != "FLM27210AP2" {
		t.Errorf("Expected ProcessorBoardID 'FLM27210AP2', got '%s'", data.ProcessorBoardID)
	}

	if data.DeviceName != "rr1-n42-r07-9336hl-13-1a" {
		t.Errorf("Expected DeviceName 'rr1-n42-r07-9336hl-13-1a', got '%s'", data.DeviceName)
	}

	if data.BootflashKB != 115805708 {
		t.Errorf("Expected BootflashKB 115805708, got %d", data.BootflashKB)
	}
}

func TestKernelUptime(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	uptime := entries[0].Message.KernelUptime

	if uptime.Days != 2 {
		t.Errorf("Expected Uptime Days 2, got %d", uptime.Days)
	}

	if uptime.Hours != 23 {
		t.Errorf("Expected Uptime Hours 23, got %d", uptime.Hours)
	}

	if uptime.Minutes != 30 {
		t.Errorf("Expected Uptime Minutes 30, got %d", uptime.Minutes)
	}

	if uptime.Seconds != 26 {
		t.Errorf("Expected Uptime Seconds 26, got %d", uptime.Seconds)
	}
}

func TestLastReset(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	reset := entries[0].Message.LastReset

	if reset.Usecs != 825938 {
		t.Errorf("Expected LastReset Usecs 825938, got %d", reset.Usecs)
	}

	if reset.Time != "Fri Oct 17 13:47:16 2025" {
		t.Errorf("Expected LastReset Time 'Fri Oct 17 13:47:16 2025', got '%s'", reset.Time)
	}

	if reset.Reason != "Reset Requested by CLI command reload" {
		t.Errorf("Expected LastReset Reason 'Reset Requested by CLI command reload', got '%s'", reset.Reason)
	}

	if reset.SystemVersion != "10.6(1)" {
		t.Errorf("Expected LastReset SystemVersion '10.6(1)', got '%s'", reset.SystemVersion)
	}

	// Service should be empty in this case
	if reset.Service != "" {
		t.Errorf("Expected LastReset Service to be empty, got '%s'", reset.Service)
	}
}

func TestPlugins(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	plugins := entries[0].Message.Plugins

	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(plugins))
	}

	expectedPlugins := []string{"Core Plugin", "Ethernet Plugin"}
	for i, expected := range expectedPlugins {
		if i < len(plugins) && plugins[i] != expected {
			t.Errorf("Expected plugin[%d] '%s', got '%s'", i, expected, plugins[i])
		}
	}
}

func TestActivePackages(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	packages := entries[0].Message.ActivePackages

	t.Logf("Found %d active packages", len(packages))
}

func TestJSONSerialization(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
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
	if entry.DataType != "cisco_nexus_version" {
		t.Errorf("Round-trip failed: data_type mismatch")
	}

	if entry.Message.NXOSVersion != "10.6(1)" {
		t.Errorf("Round-trip failed: NXOSVersion mismatch")
	}

	if entry.Message.ChassisID != "cisco Nexus9000 C9336C-FX2 Chassis" {
		t.Errorf("Round-trip failed: ChassisID mismatch")
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
	inputData, err := os.ReadFile("show-version.txt")
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

func TestJSONOutputStructure(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	// Serialize to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(entries[0], "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Verify the JSON structure contains expected keys
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Check top-level keys
	expectedKeys := []string{"data_type", "timestamp", "date", "message"}
	for _, key := range expectedKeys {
		if _, ok := jsonMap[key]; !ok {
			t.Errorf("Expected key '%s' not found in JSON output", key)
		}
	}

	// Check message structure
	message, ok := jsonMap["message"].(map[string]interface{})
	if !ok {
		t.Fatal("Message should be a map")
	}

	expectedMessageKeys := []string{
		"bios_version", "nxos_version", "release_type", "host_nxos_version",
		"bios_compile_time", "nxos_image_file", "nxos_compile_time", "nxos_timestamp",
		"boot_mode", "chassis_id", "cpu_name", "memory_kb", "processor_board_id",
		"device_name", "bootflash_kb", "kernel_uptime", "last_reset", "plugins",
		"active_packages",
	}
	for _, key := range expectedMessageKeys {
		if _, ok := message[key]; !ok {
			t.Errorf("Expected message key '%s' not found in JSON output", key)
		}
	}
}

func TestKernelUptimeStructure(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(entries[0], "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Parse back and check kernel_uptime structure
	var jsonMap map[string]interface{}
	json.Unmarshal(jsonData, &jsonMap)

	message := jsonMap["message"].(map[string]interface{})
	uptime := message["kernel_uptime"].(map[string]interface{})

	expectedUptimeKeys := []string{"days", "hours", "minutes", "seconds"}
	for _, key := range expectedUptimeKeys {
		if _, ok := uptime[key]; !ok {
			t.Errorf("Expected kernel_uptime key '%s' not found", key)
		}
	}
}

func TestLastResetStructure(t *testing.T) {
	inputData, err := os.ReadFile("show-version.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	entries, err := parseVersion(string(inputData))
	if err != nil {
		t.Fatalf("Failed to parse version: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(entries[0], "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	// Parse back and check last_reset structure
	var jsonMap map[string]interface{}
	json.Unmarshal(jsonData, &jsonMap)

	message := jsonMap["message"].(map[string]interface{})
	lastReset := message["last_reset"].(map[string]interface{})

	expectedResetKeys := []string{"usecs", "time", "reason", "system_version", "service"}
	for _, key := range expectedResetKeys {
		if _, ok := lastReset[key]; !ok {
			t.Errorf("Expected last_reset key '%s' not found", key)
		}
	}
}
