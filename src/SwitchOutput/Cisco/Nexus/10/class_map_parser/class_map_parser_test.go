package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseClassMaps(t *testing.T) {
	// Sample input data
	sampleInput := `CONTOSO-TOR1# show class-map 

  Type qos class-maps
  ====================

    class-map type qos match-all RDMA
      match cos 3

    class-map type qos match-all CLUSTER
      match cos 7

    class-map type qos match-any class-ndb-default
      Description: system ndb default

    class-map type qos match-any c-dflt-mpls-qosgrp1
      Description: This is an ingress default qos class-map that classify traffic with prec  1
      match precedence 1

  Type queuing class-maps
  ========================

    class-map type queuing match-any c-out-q3
      Description: Classifier for Egress queue 3
      match qos-group 3

    class-map type queuing match-any c-out-q2
      Description: Classifier for Egress queue 2
      match qos-group 2

  Type network-qos class-maps
  ===========================
  class-map type network-qos match-any c-nq1
      Description: Default class on qos-group 1
    match qos-group 1
  class-map type network-qos match-any c-nq2
      Description: Default class on qos-group 2
    match qos-group 2`

	// Parse the data
	classMaps := parseClassMaps(sampleInput)

	// Should have 8 class maps
	expectedCount := 8
	if len(classMaps) != expectedCount {
		t.Errorf("Expected %d class maps, got %d", expectedCount, len(classMaps))
	}

	// Create map for easier testing
	classMapsByName := make(map[string]StandardizedEntry)
	for _, entry := range classMaps {
		classMapsByName[entry.Message.ClassName] = entry
	}

	// Test RDMA class map
	if rdmaEntry, exists := classMapsByName["RDMA"]; exists {
		rdma := rdmaEntry.Message
		// Check standardized fields
		if rdmaEntry.DataType != "cisco_nexus_class_map" {
			t.Errorf("RDMA data_type: expected 'cisco_nexus_class_map', got '%s'", rdmaEntry.DataType)
		}
		if rdmaEntry.Timestamp == "" {
			t.Errorf("RDMA timestamp should not be empty")
		}
		if rdmaEntry.Date == "" {
			t.Errorf("RDMA date should not be empty")
		}
		// Check message fields
		if rdma.ClassType != "qos" {
			t.Errorf("RDMA class type: expected 'qos', got '%s'", rdma.ClassType)
		}
		if rdma.MatchType != "match-all" {
			t.Errorf("RDMA match type: expected 'match-all', got '%s'", rdma.MatchType)
		}
		if len(rdma.MatchRules) != 1 {
			t.Errorf("RDMA: expected 1 match rule, got %d", len(rdma.MatchRules))
		} else {
			if rdma.MatchRules[0].MatchType != "cos" {
				t.Errorf("RDMA match rule type: expected 'cos', got '%s'", rdma.MatchRules[0].MatchType)
			}
			if rdma.MatchRules[0].MatchValue != "3" {
				t.Errorf("RDMA match rule value: expected '3', got '%s'", rdma.MatchRules[0].MatchValue)
			}
		}
	} else {
		t.Error("RDMA class map not found in parsed data")
	}

	// Test c-dflt-mpls-qosgrp1 with description
	if qosgrp1Entry, exists := classMapsByName["c-dflt-mpls-qosgrp1"]; exists {
		qosgrp1 := qosgrp1Entry.Message
		if qosgrp1.Description == "" {
			t.Errorf("c-dflt-mpls-qosgrp1 should have a description")
		}
		if !strings.Contains(qosgrp1.Description, "ingress default qos class-map") {
			t.Errorf("c-dflt-mpls-qosgrp1 description incorrect: %s", qosgrp1.Description)
		}
		if len(qosgrp1.MatchRules) != 1 {
			t.Errorf("c-dflt-mpls-qosgrp1: expected 1 match rule, got %d", len(qosgrp1.MatchRules))
		}
	} else {
		t.Error("c-dflt-mpls-qosgrp1 class map not found in parsed data")
	}

	// Test c-out-q3 (queuing type)
	if outq3Entry, exists := classMapsByName["c-out-q3"]; exists {
		outq3 := outq3Entry.Message
		if outq3.ClassType != "queuing" {
			t.Errorf("c-out-q3 class type: expected 'queuing', got '%s'", outq3.ClassType)
		}
		if outq3.MatchType != "match-any" {
			t.Errorf("c-out-q3 match type: expected 'match-any', got '%s'", outq3.MatchType)
		}
	} else {
		t.Error("c-out-q3 class map not found in parsed data")
	}

	// Test c-nq1 (network-qos type)
	if nq1Entry, exists := classMapsByName["c-nq1"]; exists {
		nq1 := nq1Entry.Message
		if nq1.ClassType != "network-qos" {
			t.Errorf("c-nq1 class type: expected 'network-qos', got '%s'", nq1.ClassType)
		}
	} else {
		t.Error("c-nq1 class map not found in parsed data")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(classMaps)
	if err != nil {
		t.Errorf("Failed to marshal class maps to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledClassMaps []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledClassMaps)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if len(unmarshaledClassMaps) != len(classMaps) {
		t.Errorf("JSON round-trip failed: expected %d class maps, got %d", len(classMaps), len(unmarshaledClassMaps))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	// Create a temporary commands file
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "class-map",
				"command": "show class-map"
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

	// Test finding class-map command
	command, err := findClassMapCommand(config)
	if err != nil {
		t.Errorf("Failed to find class-map command: %v", err)
	}

	expectedCommand := "show class-map"
	if command != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, command)
	}
}