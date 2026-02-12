package ip_route_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseRouteType(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"C", "connected"},
		{"S", "static"},
		{"B", "bgp"},
		{"B IN", "bgp-internal"},
		{"B EX", "bgp-external"},
		{"O", "ospf"},
		{"O IA", "ospf-inter-area"},
		{"O N1", "ospf-nssa-type1"},
		{"O N2", "ospf-nssa-type2"},
		{"O E1", "ospf-external-type1"},
		{"O E2", "ospf-external-type2"},
		{"X", "unknown"},
	}

	for _, test := range tests {
		result := parseRouteType(test.code)
		if result != test.expected {
			t.Errorf("parseRouteType(%s) = %s; expected %s", test.code, result, test.expected)
		}
	}
}

func TestParsePrefix(t *testing.T) {
	tests := []struct {
		network        string
		expectedPrefix string
		expectedLength int
	}{
		{"10.1.1.0/24", "10.1.1.0", 24},
		{"192.168.0.0/16", "192.168.0.0", 16},
		{"0.0.0.0/0", "0.0.0.0", 0},
		{"10.1.1.1/32", "10.1.1.1", 32},
		{"10.1.1.1", "10.1.1.1", 32}, // Default to /32 for host routes
	}

	for _, test := range tests {
		prefix, length := parsePrefix(test.network)
		if prefix != test.expectedPrefix {
			t.Errorf("parsePrefix(%s) prefix = %s; expected %s", test.network, prefix, test.expectedPrefix)
		}
		if length != test.expectedLength {
			t.Errorf("parsePrefix(%s) length = %d; expected %d", test.network, length, test.expectedLength)
		}
	}
}

func TestParseDistMetric(t *testing.T) {
	tests := []struct {
		input          string
		expectedDist   int
		expectedMetric int
	}{
		{"0/0", 0, 0},
		{"20/100", 20, 100},
		{"110/10", 110, 10},
		{"invalid", 0, 0},
	}

	for _, test := range tests {
		dist, metric := parseDistMetric(test.input)
		if dist != test.expectedDist {
			t.Errorf("parseDistMetric(%s) dist = %d; expected %d", test.input, dist, test.expectedDist)
		}
		if metric != test.expectedMetric {
			t.Errorf("parseDistMetric(%s) metric = %d; expected %d", test.input, metric, test.expectedMetric)
		}
	}
}

func TestParseIPRoutes(t *testing.T) {
	// Sample Dell OS10 show ip route output
	sampleInput := `Codes: C - connected S - static B - BGP O - OSPF
       > - best * - candidate default + - summary route

VRF: default
---------------------------------------------------
Destination        Gateway             Interface   Dist/Metric   Last Change
---------------------------------------------------
C   10.1.1.0/24      via 10.1.1.1      vlan100      0/0           01:16:56
B EX 10.1.2.0/24     via 10.1.2.1      vlan101      20/0          01:16:56
S   192.168.0.0/16   via 10.1.1.254    ethernet1/1/1  1/0         02:30:00
C   172.16.0.0/24    Direct-connect    vlan200      0/0           00:45:30`

	routes := parseIPRoutes(sampleInput)

	if len(routes) != 4 {
		t.Errorf("Expected 4 routes, got %d", len(routes))
	}

	// Verify first route (connected)
	if len(routes) > 0 {
		route := routes[0]
		if route.DataType != "dell_os10_ip_route" {
			t.Errorf("data_type: expected 'dell_os10_ip_route', got '%s'", route.DataType)
		}
		if route.Message.Destination != "10.1.1.0/24" {
			t.Errorf("Destination: expected '10.1.1.0/24', got '%s'", route.Message.Destination)
		}
		if route.Message.Prefix != "10.1.1.0" {
			t.Errorf("Prefix: expected '10.1.1.0', got '%s'", route.Message.Prefix)
		}
		if route.Message.PrefixLength != 24 {
			t.Errorf("PrefixLength: expected 24, got %d", route.Message.PrefixLength)
		}
		if route.Message.Gateway != "10.1.1.1" {
			t.Errorf("Gateway: expected '10.1.1.1', got '%s'", route.Message.Gateway)
		}
		if route.Message.Interface != "vlan100" {
			t.Errorf("Interface: expected 'vlan100', got '%s'", route.Message.Interface)
		}
		if route.Message.RouteType != "connected" {
			t.Errorf("RouteType: expected 'connected', got '%s'", route.Message.RouteType)
		}
		if route.Message.RouteTypeCode != "C" {
			t.Errorf("RouteTypeCode: expected 'C', got '%s'", route.Message.RouteTypeCode)
		}
		if route.Message.Distance != 0 {
			t.Errorf("Distance: expected 0, got %d", route.Message.Distance)
		}
		if route.Message.VRF != "default" {
			t.Errorf("VRF: expected 'default', got '%s'", route.Message.VRF)
		}
	}

	// Verify BGP external route
	if len(routes) > 1 {
		route := routes[1]
		if route.Message.RouteType != "bgp-external" {
			t.Errorf("RouteType for BGP: expected 'bgp-external', got '%s'", route.Message.RouteType)
		}
		if route.Message.Distance != 20 {
			t.Errorf("Distance for BGP: expected 20, got %d", route.Message.Distance)
		}
	}

	// Verify static route
	if len(routes) > 2 {
		route := routes[2]
		if route.Message.RouteType != "static" {
			t.Errorf("RouteType for static: expected 'static', got '%s'", route.Message.RouteType)
		}
	}

	// Verify direct connect route
	if len(routes) > 3 {
		route := routes[3]
		if route.Message.Gateway != "direct" {
			t.Errorf("Gateway for Direct-connect: expected 'direct', got '%s'", route.Message.Gateway)
		}
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(routes)
	if err != nil {
		t.Errorf("Failed to marshal routes to JSON: %v", err)
	}

	var unmarshaledRoutes []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledRoutes)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
}

func TestParseIPRoutesEmptyInput(t *testing.T) {
	routes := parseIPRoutes("")
	if len(routes) != 0 {
		t.Errorf("Expected 0 routes for empty input, got %d", len(routes))
	}
}

func TestParseIPRoutesMultipleVRFs(t *testing.T) {
	sampleInput := `VRF: default
---------------------------------------------------
C   10.1.1.0/24      via 10.1.1.1      vlan100      0/0           01:16:56

VRF: management
---------------------------------------------------
C   192.168.1.0/24   via 192.168.1.1   mgmt1/1/1    0/0           02:00:00`

	routes := parseIPRoutes(sampleInput)

	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}

	// Verify VRF assignment
	vrfFound := map[string]bool{"default": false, "management": false}
	for _, route := range routes {
		vrfFound[route.Message.VRF] = true
	}

	if !vrfFound["default"] {
		t.Error("Expected to find route in 'default' VRF")
	}
	if !vrfFound["management"] {
		t.Error("Expected to find route in 'management' VRF")
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "ip-route",
				"command": "show ip route"
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

	command, err := findIPRouteCommand(config)
	if err != nil {
		t.Errorf("Failed to find command: %v", err)
	}

	if command != "show ip route" {
		t.Errorf("Expected command 'show ip route', got '%s'", command)
	}
}

func TestLoadCommandsFromFileNotFound(t *testing.T) {
	_, err := loadCommandsFromFile("nonexistent_file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
