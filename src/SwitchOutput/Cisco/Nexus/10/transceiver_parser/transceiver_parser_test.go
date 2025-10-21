package transceiver_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseFloatValue(t *testing.T) {
	tests := []struct {
		input        string
		expectedVal  float64
		expectedUnit string
	}{
		{"34.22 C", 34.22, "C"},
		{"3.26 V", 3.26, "V"},
		{"6.76 mA", 6.76, "mA"},
		{"-1.45 dBm", -1.45, "dBm"},
		{"0", 0, ""},
		{"invalid", 0, ""},
	}

	for _, test := range tests {
		val, unit := parseFloatValue(test.input)
		if val != test.expectedVal || unit != test.expectedUnit {
			t.Errorf("parseFloatValue(%s) = (%f, %s); expected (%f, %s)",
				test.input, val, unit, test.expectedVal, test.expectedUnit)
		}
	}
}

func TestParseBitrate(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"25500 MBit/sec", 25500},
		{"10300 MBit/sec", 10300},
		{"0", 0},
		{"invalid", 0},
	}

	for _, test := range tests {
		result := parseBitrate(test.input)
		if result != test.expected {
			t.Errorf("parseBitrate(%s) = %d; expected %d", test.input, result, test.expected)
		}
	}
}

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		value       float64
		alarmHigh   float64
		alarmLow    float64
		warningHigh float64
		warningLow  float64
		expected    string
	}{
		{34.22, 80.00, -10.00, 75.00, -5.00, "normal"},
		{76.00, 80.00, -10.00, 75.00, -5.00, "high-warning"},
		{81.00, 80.00, -10.00, 75.00, -5.00, "high-alarm"},
		{-6.00, 80.00, -10.00, 75.00, -5.00, "low-warning"},
		{-11.00, 80.00, -10.00, 75.00, -5.00, "low-alarm"},
	}

	for _, test := range tests {
		result := determineStatus(test.value, test.alarmHigh, test.alarmLow, test.warningHigh, test.warningLow)
		if result != test.expected {
			t.Errorf("determineStatus(%f, %f, %f, %f, %f) = %s; expected %s",
				test.value, test.alarmHigh, test.alarmLow, test.warningHigh, test.warningLow,
				result, test.expected)
		}
	}
}

func TestParseTransceivers(t *testing.T) {
	// Sample input data
	sampleInput := `CONTOSO-TOR1# show interface transceiver details | no

Ethernet1/1
    transceiver is present
    type is SFP-H25GB-CU3M
    name is CISCO-AMPHENOL
    part number is NDCCGJ-C403
    revision is A
    serial number is XYZ24001A1A
    nominal bitrate is 25500 MBit/sec
    Link length supported for copper is 3 m
    cable type is CA-S
    cisco id is 3
    cisco extended id number is 4
    cisco part number is 37-1792-01
    cisco product id is SFP-H25G-CU3M
    cisco version id is V01

DOM is not supported

Ethernet1/17
    transceiver is present
    type is 10Gbase-SR
    name is Siemon
    part number is S1S10F-V05.0M13
    revision is A
    serial number is SIM24001B1A
    nominal bitrate is 10300 MBit/sec
    cisco id is 3
    cisco extended id number is 4

           SFP Detail Diagnostics Information (internal calibration)
  ----------------------------------------------------------------------------
                Current              Alarms                  Warnings
                Measurement     High        Low         High          Low
  ----------------------------------------------------------------------------
  Temperature   34.22 C        80.00 C    -10.00 C     75.00 C       -5.00 C
  Voltage        3.26 V         3.60 V      3.00 V      3.50 V        3.10 V
  Current        6.76 mA       15.00 mA     0.00 mA    12.00 mA       0.00 mA
  Tx Power      -1.45 dBm       0.99 dBm   -8.32 dBm    0.00 dBm     -7.79 dBm
  Rx Power      -1.89 dBm       0.99 dBm  -10.91 dBm    0.00 dBm     -9.91 dBm
  Transmit Fault Count = 0
  ----------------------------------------------------------------------------
  Note: ++  high-alarm; +  high-warning; --  low-alarm; -  low-warning

Ethernet1/33
    transceiver is not present

Ethernet1/49
    transceiver is present
    type is QSFP-100G-CR4
    name is CISCO-AMPHENOL
    part number is NDAAFF-C401
    revision is A 
    serial number is XYZ23003Q1A-A
    nominal bitrate is 25500 MBit/sec
    Link length supported for copper is 1 m
    cisco id is 17
    cisco extended id number is 16
    cisco part number is 37-1666-01
    cisco product id is QSFP-100G-CU1M
    cisco version id is V01

DOM is not supported`

	// Parse the data
	transceivers := parseTransceivers(sampleInput)

	// Should have 4 transceivers
	expectedCount := 4
	if len(transceivers) != expectedCount {
		t.Errorf("Expected %d transceivers, got %d", expectedCount, len(transceivers))
	}

	// Create map for easier testing
	transceiverMap := make(map[string]StandardizedEntry)
	for _, entry := range transceivers {
		transceiverMap[entry.Message.InterfaceName] = entry
	}

	// Test Ethernet1/1 (copper transceiver without DOM)
	if eth1Entry, exists := transceiverMap["Ethernet1/1"]; exists {
		eth1 := eth1Entry.Message
		// Check standardized fields
		if eth1Entry.DataType != "cisco_nexus_transceiver" {
			t.Errorf("Ethernet1/1 data_type: expected 'cisco_nexus_transceiver', got '%s'", eth1Entry.DataType)
		}
		if eth1Entry.Timestamp == "" {
			t.Errorf("Ethernet1/1 timestamp should not be empty")
		}
		if eth1Entry.Date == "" {
			t.Errorf("Ethernet1/1 date should not be empty")
		}
		// Check message fields
		if !eth1.TransceiverPresent {
			t.Errorf("Ethernet1/1 transceiver should be present")
		}
		if eth1.Type != "SFP-H25GB-CU3M" {
			t.Errorf("Ethernet1/1 type: expected 'SFP-H25GB-CU3M', got '%s'", eth1.Type)
		}
		if eth1.Manufacturer != "CISCO-AMPHENOL" {
			t.Errorf("Ethernet1/1 manufacturer: expected 'CISCO-AMPHENOL', got '%s'", eth1.Manufacturer)
		}
		if eth1.SerialNumber != "XYZ24001A1A" {
			t.Errorf("Ethernet1/1 serial number: expected 'XYZ24001A1A', got '%s'", eth1.SerialNumber)
		}
		if eth1.NominalBitrate != 25500 {
			t.Errorf("Ethernet1/1 nominal bitrate: expected 25500, got %d", eth1.NominalBitrate)
		}
		if eth1.DOMSupported {
			t.Errorf("Ethernet1/1 DOM should not be supported")
		}
	} else {
		t.Error("Ethernet1/1 transceiver not found in parsed data")
	}

	// Test Ethernet1/17 (optical transceiver with DOM)
	if eth17Entry, exists := transceiverMap["Ethernet1/17"]; exists {
		eth17 := eth17Entry.Message
		if !eth17.TransceiverPresent {
			t.Errorf("Ethernet1/17 transceiver should be present")
		}
		if eth17.Type != "10Gbase-SR" {
			t.Errorf("Ethernet1/17 type: expected '10Gbase-SR', got '%s'", eth17.Type)
		}
		if !eth17.DOMSupported {
			t.Errorf("Ethernet1/17 DOM should be supported")
		}
		if eth17.DOMData == nil {
			t.Error("Ethernet1/17 should have DOM data")
		} else {
			// Check DOM Temperature
			if eth17.DOMData.Temperature == nil {
				t.Error("Ethernet1/17 DOM should have temperature data")
			} else {
				if eth17.DOMData.Temperature.CurrentValue != 34.22 {
					t.Errorf("Ethernet1/17 temperature: expected 34.22, got %f", eth17.DOMData.Temperature.CurrentValue)
				}
				if eth17.DOMData.Temperature.Unit != "C" {
					t.Errorf("Ethernet1/17 temperature unit: expected 'C', got '%s'", eth17.DOMData.Temperature.Unit)
				}
				if eth17.DOMData.Temperature.Status != "normal" {
					t.Errorf("Ethernet1/17 temperature status: expected 'normal', got '%s'", eth17.DOMData.Temperature.Status)
				}
			}
			// Check DOM Voltage
			if eth17.DOMData.Voltage == nil {
				t.Error("Ethernet1/17 DOM should have voltage data")
			} else {
				if eth17.DOMData.Voltage.CurrentValue != 3.26 {
					t.Errorf("Ethernet1/17 voltage: expected 3.26, got %f", eth17.DOMData.Voltage.CurrentValue)
				}
			}
			// Check Transmit Fault Count
			if eth17.DOMData.TransmitFaultCount != 0 {
				t.Errorf("Ethernet1/17 transmit fault count: expected 0, got %d", eth17.DOMData.TransmitFaultCount)
			}
		}
	} else {
		t.Error("Ethernet1/17 transceiver not found in parsed data")
	}

	// Test Ethernet1/33 (not present)
	if eth33Entry, exists := transceiverMap["Ethernet1/33"]; exists {
		eth33 := eth33Entry.Message
		if eth33.TransceiverPresent {
			t.Errorf("Ethernet1/33 transceiver should not be present")
		}
		if eth33.Type != "" {
			t.Errorf("Ethernet1/33 type should be empty for not present transceiver")
		}
	} else {
		t.Error("Ethernet1/33 transceiver not found in parsed data")
	}

	// Test Ethernet1/49 (QSFP transceiver)
	if eth49Entry, exists := transceiverMap["Ethernet1/49"]; exists {
		eth49 := eth49Entry.Message
		if eth49.Type != "QSFP-100G-CR4" {
			t.Errorf("Ethernet1/49 type: expected 'QSFP-100G-CR4', got '%s'", eth49.Type)
		}
		if eth49.CiscoProductID != "QSFP-100G-CU1M" {
			t.Errorf("Ethernet1/49 cisco product id: expected 'QSFP-100G-CU1M', got '%s'", eth49.CiscoProductID)
		}
	} else {
		t.Error("Ethernet1/49 transceiver not found in parsed data")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(transceivers)
	if err != nil {
		t.Errorf("Failed to marshal transceivers to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledTransceivers []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledTransceivers)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if len(unmarshaledTransceivers) != len(transceivers) {
		t.Errorf("JSON round-trip failed: expected %d transceivers, got %d", 
			len(transceivers), len(unmarshaledTransceivers))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	// Create a temporary commands file
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "transceiver",
				"command": "show interface transceiver details"
			},
			{
				"name": "test-command",
				"command": "show test"
			}
		]
	}`

	err := os.WriteFile(tempFile, []byte(commandsData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test commands file: %v", err)
	}
	defer os.Remove(tempFile)

	// Test loading commands
	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load commands from file: %v", err)
	}

	if len(config.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(config.Commands))
	}

	// Test finding transceiver command
	command, err := findTransceiverCommand(config)
	if err != nil {
		t.Errorf("Failed to find transceiver command: %v", err)
	}

	expectedCommand := "show interface transceiver details"
	if command != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, command)
	}
}

func TestParseJSON(t *testing.T) {
	// Sample JSON input with QSFP transceiver with DOM data
	sampleJSON := `{
    "TABLE_interface": {
        "ROW_interface": [
            {
                "interface": "Ethernet1/1",
                "sfp": "present",
                "type": "QSFP-100G-PCC",
                "name": "LENV-Amphenol",
                "partnum": "0000001GV910",
                "rev": "01",
                "serialnum": "Y050FY45FLLY",
                "nom_bitrate": "25500",
                "len_cu": "3",
                "ciscoid": "17",
                "ciscoid_1": "0"
            },
            {
                "interface": "Ethernet1/36/1",
                "sfp": "present",
                "type": "QSFP-40G-CSR4",
                "name": "FS",
                "partnum": "QSFP-CSR4-40G",
                "rev": "C",
                "serialnum": "C2407418236",
                "nom_bitrate": "10300",
                "len_cu": "200",
                "len_50_OM3": "300",
                "ciscoid": "13",
                "ciscoid_1": "16",
                "TABLE_lane": {
                    "ROW_lane": {
                        "lane_number": "1",
                        "temperature": "34.48",
                        "temp_flag": null,
                        "temp_alrm_hi": "75.00",
                        "temp_alrm_lo": "-5.00",
                        "temp_warn_hi": "70.00",
                        "temp_warn_lo": "0.00",
                        "voltage": "3.28",
                        "volt_flag": null,
                        "volt_alrm_hi": "3.63",
                        "volt_alrm_lo": "2.97",
                        "volt_warn_hi": "3.46",
                        "volt_warn_lo": "3.13",
                        "current": "6.49",
                        "current_flag": null,
                        "current_alrm_hi": "15.00",
                        "current_alrm_lo": "0.50",
                        "current_warn_hi": "12.00",
                        "current_warn_lo": "2.00",
                        "tx_pwr": "-1.78",
                        "tx_pwr_flag": null,
                        "tx_pwr_alrm_hi": "2.99",
                        "tx_pwr_alrm_lo": "-9.50",
                        "tx_pwr_warn_hi": "1.99",
                        "tx_pwr_warn_lo": "-7.52",
                        "rx_pwr": "-1.24",
                        "rx_pwr_flag": null,
                        "rx_pwr_alrm_hi": "3.40",
                        "rx_pwr_alrm_lo": "-14.10",
                        "rx_pwr_warn_hi": "2.39",
                        "rx_pwr_warn_lo": "-11.10",
                        "xmit_faults": "0"
                    }
                }
            }
        ]
    }
}`

	// Parse the JSON
	transceivers, err := parseJSON(sampleJSON)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should have 2 transceivers
	expectedCount := 2
	if len(transceivers) != expectedCount {
		t.Fatalf("Expected %d transceivers, got %d", expectedCount, len(transceivers))
	}

	// Create map for easier testing
	transceiverMap := make(map[string]StandardizedEntry)
	for _, entry := range transceivers {
		transceiverMap[entry.Message.InterfaceName] = entry
	}

	// Test Ethernet1/1 (QSFP without DOM)
	if eth1Entry, exists := transceiverMap["Ethernet1/1"]; exists {
		eth1 := eth1Entry.Message
		// Check standardized fields
		if eth1Entry.DataType != "cisco_nexus_transceiver" {
			t.Errorf("Ethernet1/1 data_type: expected 'cisco_nexus_transceiver', got '%s'", eth1Entry.DataType)
		}
		if !eth1.TransceiverPresent {
			t.Errorf("Ethernet1/1 transceiver should be present")
		}
		if eth1.Type != "QSFP-100G-PCC" {
			t.Errorf("Ethernet1/1 type: expected 'QSFP-100G-PCC', got '%s'", eth1.Type)
		}
		if eth1.Manufacturer != "LENV-Amphenol" {
			t.Errorf("Ethernet1/1 manufacturer: expected 'LENV-Amphenol', got '%s'", eth1.Manufacturer)
		}
		if eth1.SerialNumber != "Y050FY45FLLY" {
			t.Errorf("Ethernet1/1 serial number: expected 'Y050FY45FLLY', got '%s'", eth1.SerialNumber)
		}
		if eth1.NominalBitrate != 25500 {
			t.Errorf("Ethernet1/1 nominal bitrate: expected 25500, got %d", eth1.NominalBitrate)
		}
	} else {
		t.Error("Ethernet1/1 transceiver not found in parsed data")
	}

	// Test Ethernet1/36/1 (QSFP with DOM)
	if eth36Entry, exists := transceiverMap["Ethernet1/36/1"]; exists {
		eth36 := eth36Entry.Message
		if !eth36.TransceiverPresent {
			t.Errorf("Ethernet1/36/1 transceiver should be present")
		}
		if eth36.Type != "QSFP-40G-CSR4" {
			t.Errorf("Ethernet1/36/1 type: expected 'QSFP-40G-CSR4', got '%s'", eth36.Type)
		}
		if !eth36.DOMSupported {
			t.Errorf("Ethernet1/36/1 DOM should be supported")
		}
		if eth36.DOMData == nil {
			t.Fatal("Ethernet1/36/1 should have DOM data")
		}

		// Check DOM Temperature
		if eth36.DOMData.Temperature == nil {
			t.Error("Ethernet1/36/1 DOM should have temperature data")
		} else {
			if eth36.DOMData.Temperature.CurrentValue != 34.48 {
				t.Errorf("Ethernet1/36/1 temperature: expected 34.48, got %f", eth36.DOMData.Temperature.CurrentValue)
			}
			if eth36.DOMData.Temperature.Unit != "C" {
				t.Errorf("Ethernet1/36/1 temperature unit: expected 'C', got '%s'", eth36.DOMData.Temperature.Unit)
			}
			if eth36.DOMData.Temperature.AlarmHigh != 75.00 {
				t.Errorf("Ethernet1/36/1 temperature alarm high: expected 75.00, got %f", eth36.DOMData.Temperature.AlarmHigh)
			}
			if eth36.DOMData.Temperature.Status != "normal" {
				t.Errorf("Ethernet1/36/1 temperature status: expected 'normal', got '%s'", eth36.DOMData.Temperature.Status)
			}
		}

		// Check DOM Voltage
		if eth36.DOMData.Voltage == nil {
			t.Error("Ethernet1/36/1 DOM should have voltage data")
		} else {
			if eth36.DOMData.Voltage.CurrentValue != 3.28 {
				t.Errorf("Ethernet1/36/1 voltage: expected 3.28, got %f", eth36.DOMData.Voltage.CurrentValue)
			}
			if eth36.DOMData.Voltage.Unit != "V" {
				t.Errorf("Ethernet1/36/1 voltage unit: expected 'V', got '%s'", eth36.DOMData.Voltage.Unit)
			}
		}

		// Check DOM Current
		if eth36.DOMData.Current == nil {
			t.Error("Ethernet1/36/1 DOM should have current data")
		} else {
			if eth36.DOMData.Current.CurrentValue != 6.49 {
				t.Errorf("Ethernet1/36/1 current: expected 6.49, got %f", eth36.DOMData.Current.CurrentValue)
			}
			if eth36.DOMData.Current.Unit != "mA" {
				t.Errorf("Ethernet1/36/1 current unit: expected 'mA', got '%s'", eth36.DOMData.Current.Unit)
			}
		}

		// Check DOM TX Power
		if eth36.DOMData.TxPower == nil {
			t.Error("Ethernet1/36/1 DOM should have TX power data")
		} else {
			if eth36.DOMData.TxPower.CurrentValue != -1.78 {
				t.Errorf("Ethernet1/36/1 TX power: expected -1.78, got %f", eth36.DOMData.TxPower.CurrentValue)
			}
			if eth36.DOMData.TxPower.Unit != "dBm" {
				t.Errorf("Ethernet1/36/1 TX power unit: expected 'dBm', got '%s'", eth36.DOMData.TxPower.Unit)
			}
		}

		// Check DOM RX Power
		if eth36.DOMData.RxPower == nil {
			t.Error("Ethernet1/36/1 DOM should have RX power data")
		} else {
			if eth36.DOMData.RxPower.CurrentValue != -1.24 {
				t.Errorf("Ethernet1/36/1 RX power: expected -1.24, got %f", eth36.DOMData.RxPower.CurrentValue)
			}
			if eth36.DOMData.RxPower.Unit != "dBm" {
				t.Errorf("Ethernet1/36/1 RX power unit: expected 'dBm', got '%s'", eth36.DOMData.RxPower.Unit)
			}
		}

		// Check DOM Lane Number
		if eth36.DOMData.LaneNumber != 1 {
			t.Errorf("Ethernet1/36/1 lane number: expected 1, got %d", eth36.DOMData.LaneNumber)
		}

		// Check Transmit Fault Count
		if eth36.DOMData.TransmitFaultCount != 0 {
			t.Errorf("Ethernet1/36/1 transmit fault count: expected 0, got %d", eth36.DOMData.TransmitFaultCount)
		}
	} else {
		t.Error("Ethernet1/36/1 transceiver not found in parsed data")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(transceivers)
	if err != nil {
		t.Errorf("Failed to marshal transceivers to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledTransceivers []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledTransceivers)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if len(unmarshaledTransceivers) != len(transceivers) {
		t.Errorf("JSON round-trip failed: expected %d transceivers, got %d",
			len(transceivers), len(unmarshaledTransceivers))
	}
}