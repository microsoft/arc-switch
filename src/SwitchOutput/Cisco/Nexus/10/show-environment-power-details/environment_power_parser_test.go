package environment_power_parser

import (
	"os"
	"testing"
)

func TestParsePowerEnvironment(t *testing.T) {
	// Read the sample file
	content, err := os.ReadFile("../show-environment-power-detail.txt")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	entries, err := parsePowerEnvironment(string(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("Expected at least one entry")
	}

	entry := entries[0]

	// Check data type
	if entry.DataType != "cisco_nexus_environment_power" {
		t.Errorf("Expected data_type 'cisco_nexus_environment_power', got '%s'", entry.DataType)
	}

	// Check timestamp and date are not empty
	if entry.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	if entry.Date == "" {
		t.Error("Date should not be empty")
	}

	// Check voltage is parsed
	if entry.Message.Voltage == "" {
		t.Error("Voltage should not be empty")
	}

	// Check power supplies are parsed
	if len(entry.Message.PowerSupplies) == 0 {
		t.Error("Expected at least one power supply")
	} else {
		ps := entry.Message.PowerSupplies[0]
		if ps.PSNumber == "" {
			t.Error("PS number should not be empty")
		}
		if ps.Model == "" {
			t.Error("PS model should not be empty")
		}
		if ps.Status == "" {
			t.Error("PS status should not be empty")
		}
	}

	// Check power summary is parsed
	if entry.Message.PowerSummary.PSRedundancyModeConfigured == "" {
		t.Error("PS redundancy mode (configured) should not be empty")
	}
	if entry.Message.PowerSummary.TotalPowerCapacity == "" {
		t.Error("Total power capacity should not be empty")
	}

	// Check power details are parsed
	if entry.Message.PowerDetails.AllInletCordsConnected == "" {
		t.Error("All inlet cords connected status should not be empty")
	}

	// Check PS details are parsed
	if len(entry.Message.PSDetails) == 0 {
		t.Error("Expected at least one PS detail")
	} else {
		psDetail := entry.Message.PSDetails[0]
		if psDetail.Name == "" {
			t.Error("PS detail name should not be empty")
		}
		if psDetail.TotalCapacity == "" {
			t.Error("PS detail total capacity should not be empty")
		}
		if psDetail.Voltage == "" {
			t.Error("PS detail voltage should not be empty")
		}
	}
}

func TestParsePowerEnvironmentValues(t *testing.T) {
	// Read the sample file
	content, err := os.ReadFile("../show-environment-power-detail.txt")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	entries, err := parsePowerEnvironment(string(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entry := entries[0]

	// Check specific values from the test file
	if entry.Message.Voltage != "12 Volts" {
		t.Errorf("Expected voltage '12 Volts', got '%s'", entry.Message.Voltage)
	}

	// Check power supply details
	if len(entry.Message.PowerSupplies) >= 2 {
		ps1 := entry.Message.PowerSupplies[0]
		if ps1.PSNumber != "1" {
			t.Errorf("Expected PS number '1', got '%s'", ps1.PSNumber)
		}
		if ps1.Model != "NXA-PAC-500W-PE" {
			t.Errorf("Expected model 'NXA-PAC-500W-PE', got '%s'", ps1.Model)
		}
		if ps1.Status != "Ok" {
			t.Errorf("Expected status 'Ok', got '%s'", ps1.Status)
		}
	}

	// Check power summary values
	if entry.Message.PowerSummary.PSRedundancyModeConfigured != "PS-Redundant" {
		t.Errorf("Expected 'PS-Redundant', got '%s'", entry.Message.PowerSummary.PSRedundancyModeConfigured)
	}

	// Check power details
	if entry.Message.PowerDetails.AllInletCordsConnected != "Yes" {
		t.Errorf("Expected 'Yes', got '%s'", entry.Message.PowerDetails.AllInletCordsConnected)
	}
}
