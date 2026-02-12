package transceiver_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"25.5", 25.5},
		{"25.5C", 25.5},
		{"-10.5dBm", -10.5},
		{"3.3V", 3.3},
		{"5.0mA", 5.0},
		{"  25.5  ", 25.5},
		{"invalid", 0},
	}

	for _, test := range tests {
		result := parseFloat(test.input)
		if result != test.expected {
			t.Errorf("parseFloat(%s) = %f; expected %f", test.input, result, test.expected)
		}
	}
}

func TestParseTransceivers(t *testing.T) {
	sampleInput := `ethernet1/1/1:
    Identifier: QSFP28
    Vendor Name: ACME OPTICS
    Vendor PN: QSFP-100G-SR4
    Vendor SN: ABC123456789
    Vendor Rev: A1
    Connector: MPO 1x12
    Length Cable Assembly(m): 100

ethernet1/1/2:
    Identifier: SFP+
    Vendor Name: GENERIC INC
    Vendor PN: SFP-10G-LR
    Vendor SN: XYZ987654321
    Vendor Rev: B2
    Connector: LC`

	transceivers := parseTransceivers(sampleInput)

	if len(transceivers) != 2 {
		t.Errorf("Expected 2 transceivers, got %d", len(transceivers))
	}

	// Verify first transceiver
	if len(transceivers) > 0 {
		entry := transceivers[0]
		if entry.DataType != "dell_os10_transceiver" {
			t.Errorf("data_type: expected 'dell_os10_transceiver', got '%s'", entry.DataType)
		}
		if entry.Message.InterfaceName != "ethernet1/1/1" {
			t.Errorf("InterfaceName: expected 'ethernet1/1/1', got '%s'", entry.Message.InterfaceName)
		}
		if entry.Message.Type != "QSFP28" {
			t.Errorf("Type: expected 'QSFP28', got '%s'", entry.Message.Type)
		}
		if entry.Message.Vendor != "ACME OPTICS" {
			t.Errorf("Vendor: expected 'ACME OPTICS', got '%s'", entry.Message.Vendor)
		}
		if entry.Message.PartNumber != "QSFP-100G-SR4" {
			t.Errorf("PartNumber: expected 'QSFP-100G-SR4', got '%s'", entry.Message.PartNumber)
		}
		if entry.Message.SerialNumber != "ABC123456789" {
			t.Errorf("SerialNumber: expected 'ABC123456789', got '%s'", entry.Message.SerialNumber)
		}
		if !entry.Message.TransceiverPresent {
			t.Error("TransceiverPresent should be true")
		}
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(transceivers)
	if err != nil {
		t.Errorf("Failed to marshal transceivers to JSON: %v", err)
	}

	var unmarshaledTransceivers []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledTransceivers)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
}

func TestParseTransceiversWithDOM(t *testing.T) {
	sampleInput := `ethernet1/1/1:
    Identifier: SFP+
    Vendor Name: ACME
    Vendor PN: SFP-10G
    Vendor SN: SN123
    Temperature: 35.5
    Voltage: 3.3
    RxPower: -5.5
    TxPower: -3.2
    TxBias: 6.0`

	transceivers := parseTransceivers(sampleInput)

	if len(transceivers) != 1 {
		t.Errorf("Expected 1 transceiver, got %d", len(transceivers))
	}

	if len(transceivers) > 0 {
		msg := transceivers[0].Message
		if !msg.DOMSupported {
			t.Error("DOMSupported should be true")
		}
		if msg.DOMData == nil {
			t.Error("DOMData should not be nil")
		} else {
			if msg.DOMData.Temperature != 35.5 {
				t.Errorf("Temperature: expected 35.5, got %f", msg.DOMData.Temperature)
			}
			if msg.DOMData.Voltage != 3.3 {
				t.Errorf("Voltage: expected 3.3, got %f", msg.DOMData.Voltage)
			}
		}
	}
}

func TestParseTransceiversEmptyInput(t *testing.T) {
	transceivers := parseTransceivers("")
	if len(transceivers) != 0 {
		t.Errorf("Expected 0 transceivers for empty input, got %d", len(transceivers))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "transceiver",
				"command": "show interface transceiver"
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

	command, err := findTransceiverCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show interface transceiver" {
		t.Errorf("Expected command 'show interface transceiver', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
